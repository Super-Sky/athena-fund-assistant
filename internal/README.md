# Internal Modules

This directory contains fund assistant application modules. Business objects live here, not in Athena core.

## Subdirectory Index

- `../cmd/`
  - Runnable process entrypoints.
- `account/`
  - Account dashboard store boundary and local demo read model.
- `conversation/`
  - Agent workspace session, message, attachment metadata, upload, and trace boundary.
- `data/`
  - Data provider interface plus mock provider implementations for MVP development.
- `decision/`
  - Deterministic decision matrix generation.
- `domain/`
  - Versionable domain models and validation rules.
- `journal/`
  - Decision journal persistence boundary.
- `providerprobe/`
  - Validation-only probes for real data sources before business provider coding.
- `server/`
  - Standard-library HTTP API mapping.

## Maintenance Notes

- Keep fund business objects in this repository.
- Keep Athena integration at API / SDK / tool-contract boundaries.
- Preserve source metadata for every data-driven output.
- The web console lives under `../apps/web/` and must consume these modules through HTTP, not Go package imports.
