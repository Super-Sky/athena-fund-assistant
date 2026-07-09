# Provider Probe Command

`providerprobe` validates real data-source response shapes before a provider is connected to business workflows.

## Usage

```bash
go run ./cmd/providerprobe --provider alpha_vantage
ALPHA_VANTAGE_API_KEY=... go run ./cmd/providerprobe --provider alpha_vantage
TUSHARE_TOKEN=... go run ./cmd/providerprobe --provider tushare
```

The command emits a JSON validation report and exits non-zero when required probes fail.

The report also records governance fields such as `coverage`, `credential_required`, `production_default_allowed`, `failure_policy`, and `validation_notes`. A failed report means the source is not admitted into the default decision workflow; it may still be kept as a candidate until credentials, quota, network, and terms are validated.
