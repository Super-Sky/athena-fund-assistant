# Provider Probe Command

`providerprobe` validates real data-source response shapes before a provider is connected to business workflows.

## Usage

```bash
go run ./cmd/providerprobe --provider alpha_vantage
ALPHA_VANTAGE_API_KEY=... go run ./cmd/providerprobe --provider alpha_vantage
TUSHARE_TOKEN=... go run ./cmd/providerprobe --provider tushare
```

The command emits a JSON validation report and exits non-zero when required probes fail.
