# SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
# SPDX-License-Identifier: MIT

import copy
import unittest

from release_build import select_builds, verify_index, production_contract


def artifacts():
    result = []
    for arch in ('amd64', 'arm64'):
        result.append({'type': 'Binary', 'goos': 'linux', 'goarch': arch,
                       'goamd64': 'v3' if arch == 'amd64' else '',
                       'extra': {'ID': 'linux'}, 'path': f'dist/{arch}/cloud-init-server'})
        result.append({'type': 'Docker Image', 'name': f'ghcr.io/openchami/cloud-init:latest-{arch}',
                       'extra': {'ID': 'multiarch', 'Platforms': [f'linux/{arch}']}})
    return result


def index():
    return {'manifests': [
        {'digest': 'sha256:amd64', 'platform': {'os': 'linux', 'architecture': 'amd64'}},
        {'digest': 'sha256:arm64', 'platform': {'os': 'linux', 'architecture': 'arm64'}},
        *[{'digest': f'sha256:attest-{arch}', 'annotations': {
            'vnd.docker.reference.type': 'attestation-manifest',
            'vnd.docker.reference.digest': f'sha256:{arch}'}} for arch in ('amd64', 'arm64')],
    ]}


class ReleaseBuildTests(unittest.TestCase):
    def test_selects_release_binaries_and_snapshot_images(self):
        binaries, images = select_builds(artifacts()[::-1])
        self.assertEqual(set(binaries), {'amd64', 'arm64'})
        self.assertEqual(images['amd64'], 'ghcr.io/openchami/cloud-init:latest-amd64')

    def test_rejects_missing_platform_duplicate_binary_and_wrong_amd64_variant(self):
        for mutate in (
            lambda a: a.pop(),
            lambda a: a.append(copy.deepcopy(a[0])),
            lambda a: a[0].update(goamd64='v1'),
        ):
            data = artifacts()
            mutate(data)
            with self.assertRaises(ValueError):
                select_builds(data)

    def test_requires_actual_production_attestation_settings(self):
        profiles = [{'id': identity, 'platforms': platforms, 'sbom': 'true',
                     'flags': ['--pull=true', '--provenance=mode=min']}
                    for identity, platforms in [('multiarch', ['linux/amd64', 'linux/arm64']),
                                                ('amd64', ['linux/amd64']), ('arm64', ['linux/arm64'])]]
        production_contract({'dockers_v2': profiles})
        for field, value in [('sbom', 'false'), ('flags', ['--provenance=false']),
                             ('flags', ['--provenance=mode=min', '--push=true']),
                             ('dockerfile', 'DifferentDockerfile')]:
            data = copy.deepcopy(profiles)
            data[-1][field] = value
            with self.subTest(field=field, value=value), self.assertRaises(ValueError):
                production_contract({'dockers_v2': data})

    def test_requires_provenance_and_sbom_statements_bound_to_each_platform(self):
        def blobs(digest):
            if digest.startswith('sha256:attest-'):
                arch = digest.removeprefix('sha256:attest-')
                return {'layers': [{'digest': f'sha256:{arch}-{kind}', 'annotations': {
                    'in-toto.io/predicate-type': predicate}} for kind, predicate in (
                    ('provenance', 'https://slsa.dev/provenance/v1'), ('sbom', 'https://spdx.dev/Document'))]}
            arch, kind = digest.removeprefix('sha256:').split('-')
            return {'predicateType': 'https://slsa.dev/provenance/v1' if kind == 'provenance'
                    else 'https://spdx.dev/Document', 'subject': [{'digest': {'sha256': arch}}]}
        verify_index(index(), blobs)
        for bad in ('missing-image', 'missing-attestation', 'wrong-subject', 'missing-provenance',
                    'missing-sbom', 'wrong-statement-subject', 'wrong-statement-predicate'):
            data = index()
            if bad == 'missing-image':
                data['manifests'].pop(0)
            elif bad == 'missing-attestation':
                data['manifests'].pop()
            elif bad == 'wrong-subject':
                data['manifests'][-1]['annotations']['vnd.docker.reference.digest'] = 'sha256:other'
            def modified_blob(digest):
                document = blobs(digest)
                if digest.startswith('sha256:attest-'):
                    if bad == 'missing-provenance':
                        document['layers'].pop(0)
                    elif bad == 'missing-sbom':
                        document['layers'].pop()
                elif bad == 'wrong-statement-subject':
                    document['subject'][0]['digest']['sha256'] = 'other'
                elif bad == 'wrong-statement-predicate':
                    document['predicateType'] = 'other'
                return document
            with self.subTest(bad=bad), self.assertRaises(ValueError):
                verify_index(data, modified_blob)


if __name__ == '__main__':
    unittest.main()
