# Fund Assistant Web

React + TypeScript + Vite web console for the fund assistant MVP.

## Boundary

This package owns the user-facing fund research workflow. It calls the Go API through HTTP and must not import backend internals.

## Local Development

```bash
yarn install
yarn dev
```

The Vite dev server proxies `/api` and `/healthz` to `http://127.0.0.1:8081` by default.

## Build

```bash
yarn build
```

## File Index

- `README.md`
  - Describes the web package boundary, local commands, and file map.
- `index.html`
  - Vite HTML entry that mounts the React application.
- `package.json`
  - Declares React, Vite, TypeScript, and package scripts.
- `yarn.lock`
  - Locks web dependency versions for reproducible local and Docker builds.
- `tsconfig.json`
  - TypeScript compiler configuration for strict React code.
- `vite.config.ts`
  - Vite dev-server configuration, including API and health-check proxy rules.
- `nginx.conf`
  - Container runtime config that serves the built app and proxies API calls to the `api` service.
- `src/main.tsx`
  - Implements the MVP research workflow: profile and holding input, analysis call, strategy cards, trace display, and journal creation.
- `src/styles.css`
  - Defines the responsive operational UI styling and CSS-only icon system.
- `src/vite-env.d.ts`
  - Provides Vite client-side type references.
