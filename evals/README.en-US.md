# Deterministic Financial Eval Pack

## Scope

This package is the #31A release gate for fixed financial-safety fixtures. It uses Promptfoo `0.121.19`, Node.js `>=22.22.0`, one local file provider, and JavaScript assertions only. It has no production account, API key, LLM provider, network request, or LLM-as-judge dependency.

The repository pins Node.js `22.22.0` through `.node-version`. Use a version manager that honors this file or any standard Node.js distribution satisfying `>=22.22.0`.

The fixture matrix covers fresh baseline output, missing and stale data, provider and tool failures, absent source metadata, guaranteed-return wording, single-path conclusions, missing risk and invalidation disclosures, unsupported percentages, and unauthorized account reads. Each case also carries a linkable Athena run summary that excludes fund business payloads and credentials.

The nested `go.mod` is only a tooling boundary that keeps Go commands at the repository root from scanning Go adapters bundled in Promptfoo dependencies. This package contains no fund-domain Go code.

## Commands

```bash
npm ci
npm test
npm run eval:deterministic
```

`npm run eval:deterministic` removes stale result files, disables Promptfoo cache, database writes, result sharing, telemetry, and remote generation, and writes both:

- `artifacts/results.json`: full Promptfoo diagnostics and assertion reasons.
- `artifacts/results.junit.xml`: JUnit report selected by the required `.junit.xml` suffix.

## Gate

The deterministic threshold is 100%: every fixture and every active component assertion must pass. Any failed assertion or provider/runtime error makes the command non-zero. Blocking cases must fail closed; flagged cases must preserve a machine-readable disclosure.

Human review remains necessary for usefulness, tone, suitability, provider licensing, and any future model-assisted rubric. Such review may supplement this gate, but it must not replace or weaken the deterministic compliance rules.
