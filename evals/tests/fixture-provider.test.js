'use strict';

const assert = require('node:assert/strict');
const test = require('node:test');

const FixtureProvider = require('../providers/fixture-provider');

test('fixture provider returns structured repository JSON without fetch', async (t) => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = () => {
    throw new Error('network access is forbidden in deterministic fixtures');
  };
  t.after(() => {
    globalThis.fetch = originalFetch;
  });

  const provider = new FixtureProvider({ config: { fixtureRoot: 'fixtures' } });
  const fixtureRef = 'fixtures/cases/baseline-safe.json';
  const response = await provider.callApi(fixtureRef, { vars: { fixture: fixtureRef } });

  assert.equal(response.error, undefined);
  assert.equal(JSON.parse(response.output).id, 'baseline-safe');
  assert.equal(response.metadata.source, 'fixed_repository_fixture');
});

test('fixture provider rejects path traversal', async () => {
  const provider = new FixtureProvider({ config: { fixtureRoot: 'fixtures' } });
  const fixtureRef = '../AGENTS.md';
  const response = await provider.callApi(fixtureRef, { vars: { fixture: fixtureRef } });

  assert.match(response.error, /must stay under/);
});

test('fixture provider rejects prompt and fixture mismatches', async () => {
  const provider = new FixtureProvider({ config: { fixtureRoot: 'fixtures' } });
  const response = await provider.callApi('different.json', {
    vars: { fixture: 'fixtures/cases/baseline-safe.json' },
  });

  assert.match(response.error, /does not match fixture reference/);
});
