package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/data"
	"github.com/Super-Sky/athena-fund-assistant/internal/decision"
	"github.com/Super-Sky/athena-fund-assistant/internal/journal"
	"github.com/Super-Sky/athena-fund-assistant/internal/server"
)

func main() {
	addr := getenv("ATHENA_FUND_API_ADDR", ":8081")
	store, err := openJournalStore(context.Background())
	if err != nil {
		log.Fatalf("initialize journal store: %v", err)
	}
	defer func() {
		if err := store.Close(context.Background()); err != nil {
			log.Printf("close journal store: %v", err)
		}
	}()

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
		Journals:      store,
	})

	log.Printf("athena fund assistant api listening on %s", addr)
	if err := http.ListenAndServe(addr, svc.Routes()); err != nil {
		log.Fatalf("api server stopped: %v", err)
	}
}

func openJournalStore(parent context.Context) (journal.Store, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Print("DATABASE_URL is empty; using non-durable in-memory journal store")
		return journal.NewMemoryStore(), nil
	}

	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	store, err := journal.NewPostgresStore(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	log.Print("using PostgreSQL journal store")
	return store, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
