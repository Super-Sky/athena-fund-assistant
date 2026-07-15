'use strict';

const MINIMUM_NODE = Object.freeze({ major: 22, minor: 22 });

// assertSupportedNode keeps Promptfoo execution on the repository-pinned runtime floor.
// assertSupportedNode 确保 Promptfoo 只在仓库锁定的最低 Node 版本以上运行。
function assertSupportedNode(version = process.versions.node) {
  const [major, minor] = version.split('.').map(Number);
  const supported = major > MINIMUM_NODE.major ||
    (major === MINIMUM_NODE.major && minor >= MINIMUM_NODE.minor);

  if (!supported) {
    throw new Error(
      `Node >=22.22.0 is required for deterministic evals; received ${version}`,
    );
  }
}

if (require.main === module) {
  try {
    assertSupportedNode();
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}

module.exports = { assertSupportedNode, MINIMUM_NODE };
