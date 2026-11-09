# SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
# SPDX-License-Identifier: MIT

"""Check GoReleaser snapshots and export a combined image without publication."""

import json
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import time
import uuid

ARCHES = {'amd64', 'arm64'}
PROVENANCE = {'https://slsa.dev/provenance/v0.2', 'https://slsa.dev/provenance/v1'}


def select_builds(artifacts):
    binaries, images = {}, {}
    for arch in sorted(ARCHES):
        matches = [a for a in artifacts if a.get('type') == 'Binary'
                   and a.get('extra', {}).get('ID') == 'linux'
                   and a.get('goos') == 'linux' and a.get('goarch') == arch]
        if len(matches) != 1 or (arch == 'amd64' and matches[0].get('goamd64') != 'v3'):
            raise ValueError(f'Expected one release binary for {arch}, with amd64 v3')
        binaries[arch] = matches[0]['path']
        matches = [a for a in artifacts if a.get('type') == 'Docker Image'
                   and a.get('extra', {}).get('ID') == 'multiarch'
                   and a.get('extra', {}).get('Platforms') == [f'linux/{arch}']]
        if not matches:
            raise ValueError(f'Missing snapshot image for {arch}')
        images[arch] = sorted(a['name'] for a in matches)[0]
    return binaries, images


def verify_index(index, blob):
    images = [m for m in index['manifests'] if m.get('platform', {}).get('os') == 'linux']
    if len(images) != 2 or {m['platform']['architecture'] for m in images} != ARCHES:
        raise ValueError('OCI index must contain exactly linux/amd64 and linux/arm64')
    expected = {m['digest'] for m in images}
    covered = set()
    for manifest in index['manifests']:
        annotations = manifest.get('annotations', {})
        if annotations.get('vnd.docker.reference.type') != 'attestation-manifest':
            continue
        subject = annotations.get('vnd.docker.reference.digest')
        if subject not in expected:
            raise ValueError('Attestation references an unexpected image')
        predicates = set()
        for layer in blob(manifest['digest'])['layers']:
            statement = blob(layer['digest'])
            predicate = statement.get('predicateType')
            if predicate != layer.get('annotations', {}).get('in-toto.io/predicate-type'):
                raise ValueError('Attestation annotation does not match its statement')
            if not any(s.get('digest', {}).get('sha256') == subject.removeprefix('sha256:')
                       for s in statement.get('subject', [])):
                raise ValueError(f'Attestation statement does not identify {subject}: {statement.get("subject")}')
            predicates.add(predicate)
        if not predicates.intersection(PROVENANCE) or 'https://spdx.dev/Document' not in predicates:
            raise ValueError('Each architecture requires provenance and an SPDX SBOM')
        covered.add(subject)
    if covered != expected:
        raise ValueError('Missing architecture attestation')


def production_contract(config):
    profiles = config.get('dockers_v2', [])
    expected = {'multiarch': {'linux/amd64', 'linux/arm64'},
                'amd64': {'linux/amd64'}, 'arm64': {'linux/arm64'}}
    selected = {}
    for identity, platforms in expected.items():
        matches = [p for p in profiles if p.get('id') == identity]
        if len(matches) != 1 or set(matches[0].get('platforms', [])) != platforms:
            raise ValueError(f'Unexpected production platforms for {identity}')
        profile = matches[0]
        flags = profile.get('flags', [])
        # This allowlist keeps publication/output overrides out of the PR build.
        allowed = {'--pull=true', '--provenance=mode=min', '--provenance=mode=max', '--provenance=true'}
        if not set(flags).issubset(allowed) or not any(f.startswith('--provenance=') for f in flags):
            raise ValueError(f'{identity} must explicitly enable provenance using supported validation flags')
        if str(profile.get('sbom', '')).lower() != 'true':
            raise ValueError(f'{identity} must enable production SBOMs')
        if '{{' in profile.get('dockerfile', 'Dockerfile') or profile.get('build_args'):
            raise ValueError('Adapt the release gate for templated Dockerfiles or build arguments')
        selected[identity] = profile
    combined = selected['multiarch']
    for identity in ('amd64', 'arm64'):
        for field in ('dockerfile', 'ids', 'extra_files', 'build_args', 'flags', 'sbom'):
            if selected[identity].get(field) != combined.get(field):
                raise ValueError(f'Adapt the release gate for different {identity} {field}')
    return combined


def run(*args):
    subprocess.run(args, check=True)


def smoke(arch, image):
    # Never pull a published image as a fallback for a missing snapshot.
    run('docker', 'run', '--rm', '--pull=never', '--network', 'none', '--platform',
        f'linux/{arch}', image, '/usr/local/bin/cloud-init-server', '--help')
    name = f'release-smoke-{arch}-{uuid.uuid4().hex[:8]}'
    with tempfile.TemporaryFile(mode='w+') as logs:
        proc = subprocess.Popen([
            'docker', 'run', '--rm', '--pull=never', '--name', name, '--network', 'none',
            '--platform', f'linux/{arch}', image, '/usr/local/bin/cloud-init-server',
            '--listen', '127.0.0.1:27777', '--cluster-name', 'release-validation',
            '--smd-simulator', '--storage-backend', 'quack', '--db-path', '/data/validation.duckdb',
        ], stdout=logs, stderr=subprocess.STDOUT)
        try:
            deadline = time.monotonic() + 30
            while True:
                if proc.poll() is not None:
                    raise RuntimeError(f'{arch} server exited during startup')
                try:
                    response = subprocess.run([
                    'docker', 'exec', name, 'bash', '-c',
                    "exec 3<>/dev/tcp/127.0.0.1/27777; "
                    "printf 'GET /version HTTP/1.1\\r\\nHost: localhost\\r\\nConnection: close\\r\\n\\r\\n' >&3; cat <&3",
                    ], capture_output=True, text=True, timeout=5)
                except subprocess.TimeoutExpired:
                    response = None
                if response is not None and response.returncode == 0 and response.stdout.startswith('HTTP/1.1 200'):
                    break
                if time.monotonic() >= deadline:
                    raise RuntimeError(f'{arch} server did not return HTTP 200')
                time.sleep(.25)
            print(f'{arch}: DuckDB startup and GET /version passed', flush=True)
        finally:
            if proc.poll() is None:
                # --rm also removes the anonymous /data volume.
                subprocess.run(['docker', 'stop', '--timeout', '3', name], check=True)
            proc.wait(timeout=10)
            logs.seek(0)
            print(logs.read(), flush=True)


def main():
    import yaml  # Installed only in the release verification environment.
    config = yaml.safe_load(Path('dist/config.yaml').read_text())
    profile = production_contract(config)
    binaries, images = select_builds(json.loads(Path('dist/artifacts.json').read_text()))
    for arch in sorted(ARCHES):
        smoke(arch, images[arch])
    with tempfile.TemporaryDirectory(prefix='release-oci-') as directory:
        context = Path(directory) / 'context'
        context.mkdir()
        shutil.copy2(profile.get('dockerfile', 'Dockerfile'), context / 'Dockerfile')
        for extra in profile.get('extra_files', []):
            source = Path(extra).resolve()
            if not source.is_relative_to(Path.cwd()):
                raise ValueError('Extra file must be inside the repository')
            target = context / source.relative_to(Path.cwd())
            target.parent.mkdir(parents=True, exist_ok=True)
            if source.is_dir():
                shutil.copytree(source, target)
            else:
                shutil.copy2(source, target)
        for arch, source in binaries.items():
            source = Path(source).resolve()
            if not source.is_relative_to(Path('dist').resolve()):
                raise ValueError('Release binary must be inside dist')
            target = context / 'linux' / arch
            target.mkdir(parents=True)
            shutil.copy2(source, target / 'cloud-init-server')
        archive = Path(directory) / 'image.tar'
        # OCI export exercises combined indexes and attestations without --push.
        # Snapshot mode alone loads separate images and omits production SBOMs.
        run('docker', 'buildx', 'build', '--platform', ','.join(profile['platforms']),
            *profile['flags'], '--attest=type=sbom', '-t', 'cloud-init-release-validation:local', '--output',
            f'type=oci,dest={archive}', str(context))
        with tarfile.open(archive) as tar:
            def blob(digest):
                algorithm, value = digest.split(':', 1)
                if algorithm != 'sha256' or len(value) != 64 or any(c not in '0123456789abcdef' for c in value):
                    raise ValueError('Expected SHA256 OCI blob reference')
                return json.load(tar.extractfile(f'blobs/sha256/{value}'))
            index = json.load(tar.extractfile('index.json'))
            # OCI exporters wrap the image index in a layout index.
            if len(index['manifests']) == 1 and index['manifests'][0]['mediaType'].endswith('image.index.v1+json'):
                index = blob(index['manifests'][0]['digest'])
            verify_index(index, blob)
        print('Combined OCI index: both architectures, provenance and SBOMs passed', flush=True)


if __name__ == '__main__':
    main()
