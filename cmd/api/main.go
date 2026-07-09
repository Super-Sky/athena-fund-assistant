package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/account"
	"github.com/Super-Sky/athena-fund-assistant/internal/data"
	"github.com/Super-Sky/athena-fund-assistant/internal/decision"
	"github.com/Super-Sky/athena-fund-assistant/internal/journal"
	"github.com/Super-Sky/athena-fund-assistant/internal/server"
)

func main() {
	addr := getenv("ATHENA_FUND_API_ADDR", ":8081")
	provider := data.NewMockProvider()
	report := data.ValidateProvider(context.Background(), provider, data.ValidationOptions{
		FundCodes:     []string{"QQQ"},
		EquitySymbols: []string{"AAPL"},
		IndexCodes:    []string{"NDX"},
		FXPairs:       []data.FXPair{{BaseCurrency: "USD", QuoteCurrency: "CNY"}},
		Calendars:     []data.CalendarProbe{{Market: "US", Date: time.Now().UTC()}},
	})
	if !report.Passed {
		log.Fatalf("data provider validation failed: %+v", report.Checks)
	}
	log.Printf("data provider validation passed with %d checks", len(report.Checks))

	svc := server.New(server.Dependencies{
		Provider:      provider,
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      account.NewMemoryStore(),
	})

	log.Printf("athena fund assistant api listening on %s", addr)
	if err := http.ListenAndServe(addr, svc.Routes()); err != nil {
		log.Fatalf("api server stopped: %v", err)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
