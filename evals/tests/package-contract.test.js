'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');

const { findSecrets } = require('../assertions/financial-policy');
const { assertSupportedNode } = require('../scripts/check-node-version');
const { verifyArtifacts } = require('../scripts/run-deterministic-eval');

const EVAL_ROOT = path.resolve(__dirname, '..');

test('package pins Promptfoo and Node runtime exactly as required', () => {
  const packageJson = JSON.parse(fs.readFileSync(path.join(EVAL_ROOT, 'package.json'), 'utf8'));

  assert.equal(packageJson.devDependencies.promptfoo, '0.121.19');
  assert.equal(packageJson.engines.node, '>=22.22.0');
  assert.equal(packageJson.scripts['eval:deterministic'], 'node scripts/run-deterministic-eval.js');
});

test('nested Go module keeps Promptfoo adapters out of root Go discovery', () => {
  const goModule = fs.readFileSync(path.join(EVAL_ROOT, 'go.mod'), 'utf8');

  assert.match(goModule, /^module github\.com\/Super-Sky\/athena-fund-assistant\/evals$/m);
  assert.match(goModule, /^go 1\.23\.0$/m);
});

test('Node guard accepts the floor and rejects older runtimes', () => {
  assert.doesNotThrow(() => assertSupportedNode('22.22.0'));
  assert.doesNotThrow(() => assertSupportedNode('23.0.0'));
  assert.throws(() => assertSupportedNode('22.21.9'), /Node >=22.22.0/);
});

test('test manifest covers every required rule with deterministic JavaScript only', () => {
  const cases = JSON.parse(fs.readFileSync(path.join(EVAL_ROOT, 'fixtures', 'tests.json'), 'utf8'));
  const rules = new Set(cases.map((item) => item.vars.expected_rule));
  const required = [
    'baseline_safe',
    'data_missing',
    'data_stale',
    'provider_failure',
    'tool_failure',
    'source_metadata_missing',
    'guaranteed_return',
    'single_path',
    'risk_missing',
    'invalidation_missing',
    'unsupported_percentage',
    'unauthorized_account_read',
  ];

  assert.deepEqual([...rules].sort(), required.sort());
  for (const testCase of cases) {
    assert.deepEqual(testCase.assert.map((item) => item.type), ['javascript']);
    assert.equal(testCase.assert[0].value, 'file://assertions/financial-policy.js');
  }
});

test('all fixed fixtures contain eval-only trace references and no credentials', () => {
  const cases = JSON.parse(fs.readFileSync(path.join(EVAL_ROOT, 'fixtures', 'tests.json'), 'utf8'));
  for (const testCase of cases) {
    const fixture = JSON.parse(fs.readFileSync(path.join(EVAL_ROOT, testCase.vars.fixture), 'utf8'));
    assert.match(fixture.athena_trace.run_id, /^run_eval_/);
    assert.match(fixture.athena_trace.trace_id, /^trace_eval_/);
    assert.ok(Number.isFinite(fixture.athena_trace.duration_ms));
    assert.deepEqual(fixture.athena_trace.safety_summary, {
      business_payload_included: false,
      sensitive_values_included: false,
    });
    assert.deepEqual(findSecrets(fixture), []);
  }
});

test('artifact verifier accepts JSON plus JUnit and rejects generic Promptfoo XML', () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'athena-fund-eval-'));
  const jsonPath = path.join(directory, 'results.json');
  const junitPath = path.join(directory, 'results.junit.xml');
  fs.writeFileSync(jsonPath, JSON.stringify({ version: 3, results: {} }));
  fs.writeFileSync(junitPath, '<testsuites><testsuite><testcase name="safe"/></testsuite></testsuites>');

  assert.doesNotThrow(() => verifyArtifacts(jsonPath, junitPath));

  fs.writeFileSync(junitPath, '<promptfoo><testcase name="wrong"/></promptfoo>');
  assert.throws(() => verifyArtifacts(jsonPath, junitPath), /not a JUnit test report/);
});

test('runner requests both exact artifact names including the JUnit suffix', () => {
  const runner = fs.readFileSync(path.join(EVAL_ROOT, 'scripts', 'run-deterministic-eval.js'), 'utf8');

  assert.match(runner, /artifacts\/results\.json/);
  assert.match(runner, /artifacts\/results\.junit\.xml/);
  assert.doesNotMatch(runner, /artifacts\/junit\.xml/);
  assert.match(runner, /'--no-write'/);
  assert.match(runner, /'--no-share'/);
});
