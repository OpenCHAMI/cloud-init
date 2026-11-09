// SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
// SPDX-License-Identifier: MIT

const assert = require('node:assert/strict');
const test = require('node:test');
const { selectDigests } = require('./release-image-digests.cjs');
const registry = 'ghcr.io/openchami/cloud-init';
const tag = 'v1.4.15';
const digest = n => `sha256:${n.repeat(64)}`;
const artifacts = () => [
  { name: `${registry}:${tag}-arm64`, extra: { Digest: digest('c') } },
  { name: 'unrelated', extra: { Digest: digest('f') } },
  { name: `${registry}:${tag}`, extra: { Digest: digest('a') } },
  { name: `${registry}:latest`, extra: { Digest: digest('a') } },
  { name: `${registry}:${tag}-amd64`, extra: { Digest: digest('b') } },
];

test('selects each exact versioned image independently of artifact order', () => {
  const expected = { multiarch: digest('a'), amd64: digest('b'), arm64: digest('c') };
  assert.deepEqual(selectDigests(artifacts(), tag), expected);
  assert.deepEqual(selectDigests(artifacts().reverse(), tag), expected);
});

test('rejects missing, duplicate and invalid digest subjects', () => {
  assert.throws(() => selectDigests(artifacts().slice(1), tag), /arm64/);
  const duplicate = artifacts();
  duplicate.push(duplicate[2]);
  assert.throws(() => selectDigests(duplicate, tag), /multiarch/);
  const invalid = artifacts();
  invalid[2].extra.Digest = 'undefined';
  assert.throws(() => selectDigests(invalid, tag), /multiarch/);
  assert.throws(() => selectDigests({}, tag), /array/);
  assert.throws(() => selectDigests(artifacts(), 'latest'), /release tag/);
});
