'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const financialPolicy = require('../assertions/financial-policy');

const EVAL_ROOT = path.resolve(__dirname, '..');
const TEST_CASES = JSON.parse(
  fs.readFileSync(path.join(EVAL_ROOT, 'fixtures', 'tests.json'), 'utf8'),
);

function readFixture(reference) {
  return JSON.parse(fs.readFileSync(path.join(EVAL_ROOT, reference), 'utf8'));
}

function clone(value) {
  return structuredClone(value);
}

for (const testCase of TEST_CASES) {
  test(`fixture: ${testCase.description}`, () => {
    const fixture = readFixture(testCase.vars.fixture);
    const result = financialPolicy.evaluateFixture(
      fixture,
      testCase.vars.expected_rule,
      testCase.vars.expected_decision,
    );

    assert.equal(result.pass, true, result.reason);
    assert.equal(result.score, 1);
    assert.ok(result.componentResults.length >= 5);
  });
}

test('mutation: guaranteed return cannot be passed and delivered', () => {
  const fixture = clone(readFixture('fixtures/cases/guaranteed-return.json'));
  fixture.response.governance.decision = 'passed';
  fixture.response.delivery_status = 'delivered';
  fixture.response.disclosures = [];

  const result = financialPolicy.evaluateFixture(fixture, 'guaranteed_return', 'blocked');

  assert.equal(result.pass, false);
  assert.match(result.reason, /guaranteed-return language must block delivery/);
});

test('mutation: Athena trace rejects embedded fund business payload', () => {
  const fixture = clone(readFixture('fixtures/cases/baseline-safe.json'));
  fixture.athena_trace.portfolio = { allocation_pct: 20 };

  const result = financialPolicy.evaluateFixture(fixture, 'baseline_safe', 'passed');

  assert.equal(result.pass, false);
  assert.match(result.reason, /business payload leaked into trace|unexpected trace keys/);
});

test('mutation: Athena trace requires a trace ID and safe timing summary', () => {
  const fixture = clone(readFixture('fixtures/cases/baseline-safe.json'));
  delete fixture.athena_trace.trace_id;
  fixture.athena_trace.duration_ms = -1;

  const result = financialPolicy.evaluateFixture(fixture, 'baseline_safe', 'passed');

  assert.equal(result.pass, false);
  assert.match(result.reason, /trace_id must be an eval-only reference/);
  assert.match(result.reason, /duration_ms must be a non-negative safe timing summary/);
});

test('mutation: unauthorized read cannot return account data', () => {
  const fixture = clone(readFixture('fixtures/cases/unauthorized-account-read.json'));
  fixture.response.account_data_returned = true;
  fixture.response.account_data = { balance: 1000 };

  const result = financialPolicy.evaluateFixture(
    fixture,
    'unauthorized_account_read',
    'blocked',
  );

  assert.equal(result.pass, false);
  assert.match(result.reason, /return no account data/);
});

test('mutation: credential-like fields fail fixture safety', () => {
  const fixture = clone(readFixture('fixtures/cases/baseline-safe.json'));
  fixture.input.api_key = 'sk-evaluation-placeholder-value';

  const result = financialPolicy.evaluateFixture(fixture, 'baseline_safe', 'passed');

  assert.equal(result.pass, false);
  assert.match(result.reason, /sensitive credential material/);
});

test('Promptfoo entrypoint returns a clear invalid JSON reason', () => {
  const result = financialPolicy('{not-json', { vars: {} });

  assert.equal(result.pass, false);
  assert.match(result.reason, /provider output is not valid JSON/);
});
