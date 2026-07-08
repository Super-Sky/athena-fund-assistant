package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/providerprobe"
)

func main() {
	provider := flag.String("provider", "alpha_vantage", "provider to probe: alpha_vantage or tushare")
	timeout := flag.Duration("timeout", 20*time.Second, "HTTP timeout")
	alphaKey := flag.String("alpha-key", os.Getenv("ALPHA_VANTAGE_API_KEY"), "Alpha Vantage API key")
	tushareToken := flag.String("tushare-token", os.Getenv("TUSHARE_TOKEN"), "Tushare Pro token")
	flag.Parse()

	ctx := context.Background()
	var report providerprobe.Report
	switch *provider {
	case "alpha_vantage":
		report = providerprobe.ProbeAlphaVantage(ctx, providerprobe.AlphaVantageConfig{
			APIKey:  *alphaKey,
			Timeout: *timeout,
		})
	case "tushare":
		report = providerprobe.ProbeTushare(ctx, providerprobe.TushareConfig{
			Token:   *tushareToken,
			Timeout: *timeout,
		})
	default:
		fmt.Fprintf(os.Stderr, "unsupported provider %q\n", *provider)
		os.Exit(2)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "encode report: %v\n", err)
		os.Exit(2)
	}
	if !report.Passed {
		os.Exit(1)
	}
}
