# Repository Working Notes

## Role

This repository is the fund research assistant application layer for Athena. Keep Athena core generic; put fund-specific product behavior, UI, data adapters, and decision workflows here.

## Boundaries

- Do not implement automatic trading.
- Do not store brokerage credentials in this repository.
- Do not present outputs as guaranteed returns or regulated advisory conclusions.
- Do not copy Athena internal runtime code into this repository.
- Prefer API / SDK / tool-contract integration with Athena.

## Documentation

Planning and product decisions live in `docs/`. Update the relevant document when changing:

- product boundary
- governance behavior
- data provider assumptions
- domain object model
- Athena integration contract
- MVP scope

Documentation must support Chinese and English versions:

- Write durable docs as paired files: `topic.zh-CN.md` and `topic.en-US.md`.
- Keep both versions semantically aligned; do not let one version become stale.
- `docs/README.md` should link both language versions when they exist.
- Short operational files such as `AGENTS.md`, issue templates, config files, and generated metadata may stay English if they are tooling entrypoints.

## Engineering Defaults

- Keep domain models explicit and versionable.
- Every data-driven conclusion should preserve source, timestamp, and freshness metadata.
- Every decision option should include evidence, risk, invalidation condition, and review timing.
- Business objects belong here; generic agent runtime capabilities belong in Athena.
- Code comments for exported or core behavior must be bilingual, with English first and Chinese second on separate lines.
