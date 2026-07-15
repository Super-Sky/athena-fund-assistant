'use strict';

const fs = require('node:fs/promises');
const path = require('node:path');

// FixtureProvider returns repository-owned JSON and never calls a network or production service.
// FixtureProvider 只返回仓库内 JSON，绝不调用网络或生产服务。
class FixtureProvider {
  constructor(options = {}) {
    this.providerId = options.id || 'fixed-json-fixture-provider';
    this.evalRoot = path.resolve(__dirname, '..');
    this.fixtureRoot = path.resolve(
      this.evalRoot,
      options.config?.fixtureRoot || 'fixtures',
    );
  }

  id() {
    return this.providerId;
  }

  // callApi loads exactly one declared fixture and returns a structured JSON string.
  // callApi 精确加载一个声明的 fixture，并返回结构化 JSON 字符串。
  async callApi(prompt, context = {}) {
    const fixtureRef = context.vars?.fixture;
    if (typeof fixtureRef !== 'string' || fixtureRef.trim() === '') {
      return { error: 'Fixture provider requires vars.fixture' };
    }
    if (typeof prompt === 'string' && prompt.trim() !== fixtureRef) {
      return { error: `Rendered prompt does not match fixture reference ${fixtureRef}` };
    }

    const fixturePath = path.resolve(this.evalRoot, fixtureRef);
    const fixturePrefix = `${this.fixtureRoot}${path.sep}`;
    if (!fixturePath.startsWith(fixturePrefix) || path.extname(fixturePath) !== '.json') {
      return { error: `Fixture path must stay under ${this.fixtureRoot}` };
    }

    try {
      const fixture = JSON.parse(await fs.readFile(fixturePath, 'utf8'));
      return {
        output: JSON.stringify(fixture),
        metadata: {
          fixture_id: fixture.id,
          fixture_path: path.relative(this.evalRoot, fixturePath),
          source: 'fixed_repository_fixture',
        },
      };
    } catch (error) {
      return { error: `Unable to load fixture ${fixtureRef}: ${error.message}` };
    }
  }
}

module.exports = FixtureProvider;
