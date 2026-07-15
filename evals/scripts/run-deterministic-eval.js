'use strict';

const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const { assertSupportedNode } = require('./check-node-version');

const EVAL_ROOT = path.resolve(__dirname, '..');
const JSON_RESULT = path.join(EVAL_ROOT, 'artifacts', 'results.json');
const JUNIT_RESULT = path.join(EVAL_ROOT, 'artifacts', 'results.junit.xml');

// verifyArtifacts prevents a successful command from publishing stale or misclassified XML.
// verifyArtifacts 防止成功命令发布陈旧产物或被误分类的 XML。
function verifyArtifacts(jsonPath = JSON_RESULT, junitPath = JUNIT_RESULT) {
  const json = JSON.parse(fs.readFileSync(jsonPath, 'utf8'));
  if (!json || typeof json !== 'object' || Array.isArray(json)) {
    throw new Error(`${jsonPath} is not a Promptfoo JSON result object`);
  }

  const junit = fs.readFileSync(junitPath, 'utf8').trim();
  if (!junit.includes('<testsuites') || !junit.includes('<testcase')) {
    throw new Error(`${junitPath} is not a JUnit test report`);
  }
  if (junit.includes('<promptfoo>')) {
    throw new Error(`${junitPath} was emitted as generic Promptfoo XML, not JUnit XML`);
  }
}

// run executes one offline fixture matrix and writes both diagnostic formats.
// run 执行一次离线 fixture 矩阵并写出两种可诊断格式。
function run() {
  assertSupportedNode();
  fs.mkdirSync(path.dirname(JSON_RESULT), { recursive: true });
  fs.rmSync(JSON_RESULT, { force: true });
  fs.rmSync(JUNIT_RESULT, { force: true });

  const packagePath = path.join(EVAL_ROOT, 'node_modules', 'promptfoo', 'package.json');
  const promptfooPackage = JSON.parse(fs.readFileSync(packagePath, 'utf8'));
  if (promptfooPackage.version !== '0.121.19') {
    throw new Error(`Expected Promptfoo 0.121.19, received ${promptfooPackage.version}`);
  }
  const cliPath = path.resolve(path.dirname(packagePath), promptfooPackage.bin.promptfoo);
  const args = [
    cliPath,
    'eval',
    '--config',
    'promptfooconfig.yaml',
    '--no-cache',
    '--no-write',
    '--no-share',
    '--no-table',
    '--no-progress-bar',
    '--output',
    'artifacts/results.json',
    '--output',
    'artifacts/results.junit.xml',
  ];
  const child = spawnSync(process.execPath, args, {
    cwd: EVAL_ROOT,
    env: {
      ...process.env,
      PROMPTFOO_CACHE_ENABLED: 'false',
      PROMPTFOO_CONFIG_DIR: path.join(EVAL_ROOT, '.promptfoo'),
      PROMPTFOO_DISABLE_REMOTE_GENERATION: 'true',
      PROMPTFOO_DISABLE_SHARING: 'true',
      PROMPTFOO_DISABLE_TELEMETRY: '1',
      PROMPTFOO_DISABLE_UPDATE: '1',
      FORCE_COLOR: '0',
    },
    stdio: 'inherit',
  });

  let artifactError;
  try {
    verifyArtifacts();
  } catch (error) {
    artifactError = error;
  }

  if (child.error) {
    throw child.error;
  }
  if (child.status !== 0) {
    if (artifactError) {
      console.error(`Artifact verification also failed: ${artifactError.message}`);
    }
    process.exitCode = child.status ?? 1;
    return;
  }
  if (artifactError) {
    throw artifactError;
  }
}

if (require.main === module) {
  try {
    run();
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}

module.exports = { run, verifyArtifacts, JSON_RESULT, JUNIT_RESULT };
