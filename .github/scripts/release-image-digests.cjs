// SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
// SPDX-License-Identifier: MIT

const fs = require('node:fs');

function selectDigests(artifacts, tag) {
  if (!Array.isArray(artifacts)) throw new Error('Expected an artifacts array');
  if (!/^v\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$/.test(tag)) {
    throw new Error('Expected a versioned release tag');
  }
  const result = {};
  for (const [id, suffix] of Object.entries({ multiarch: '', amd64: '-amd64', arm64: '-arm64' })) {
    const name = `ghcr.io/openchami/cloud-init:${tag}${suffix}`;
    const matches = artifacts.filter(artifact => artifact?.name === name);
    if (matches.length !== 1 || !/^sha256:[0-9a-f]{64}$/.test(matches[0]?.extra?.Digest)) {
      throw new Error(`Expected one valid digest for ${id}: ${name}`);
    }
    result[id] = matches[0].extra.Digest;
  }
  return result;
}

if (require.main === module) {
  const digests = selectDigests(JSON.parse(process.env.ARTIFACTS), process.env.GITHUB_REF_NAME);
  fs.appendFileSync(process.env.GITHUB_OUTPUT,
    Object.entries(digests).map(([name, digest]) => `${name}=${digest}\n`).join(''));
}

module.exports = { selectDigests };
