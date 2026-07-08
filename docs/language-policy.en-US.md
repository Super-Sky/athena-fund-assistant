# Documentation And Code Comment Language Policy

## Goal

This repository serves Chinese product planning and English engineering collaboration. Durable documentation must support both Chinese and English versions so product, governance, and implementation semantics stay aligned.

## Documentation Rules

Stable documents should use paired files:

- `topic.zh-CN.md`
- `topic.en-US.md`

Examples:

- `product-boundary.zh-CN.md`
- `product-boundary.en-US.md`

Requirements:

- Both versions must describe the same facts, boundaries, and acceptance criteria.
- Wording may be adapted for each language, but conclusions must stay aligned.
- When one version changes, check whether the other version needs the same update.
- `docs/README.md` should link both language versions.
- Historical single-language documents may remain as seed drafts, but completed delivery should converge to paired versions.

## Code Comment Rules

Core types, exported functions, complex business rules, and easy-to-misuse logic must use bilingual comments.

Required format:

```ts
// DecisionOption describes one actionable fund decision branch.
// DecisionOption 描述一个可执行的基金决策分支。
export interface DecisionOption {}
```

Requirements:

- English first, Chinese second.
- Use two separate lines.
- Do not add empty comments for obvious code.
- Comments should explain why the code exists, what boundary it protects, or what changes may be affected.

## Naming Rules

- Code filenames, package names, and config names should stay English by default.
- User-facing text can be provided by product-level i18n resources.
- Non-tooling docs should prefer `*.zh-CN.md` and `*.en-US.md` suffixes.

## Delivery Gate

Before delivery, bilingual sync must be checked when touching:

- product boundary
- financial governance rules
- data-source and licensing assumptions
- Athena integration contract
- domain models
- MVP acceptance criteria
- user-facing risk notices

