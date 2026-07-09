package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/account"
	"github.com/Super-Sky/athena-fund-assistant/internal/conversation"
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
	var accountStore account.Store = account.NewMemoryStore()
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		postgresStore, err := account.NewPostgresStore(context.Background(), databaseURL)
		if err != nil {
			log.Fatalf("postgres account store initialization failed: %v", err)
		}
		defer func() {
			if err := postgresStore.Close(); err != nil {
				log.Printf("postgres account store close failed: %v", err)
			}
		}()
		accountStore = postgresStore
		log.Printf("account store using PostgreSQL")
	} else {
		log.Printf("account store using in-memory demo data")
	}
	conversationStore, err := conversation.NewMemoryStore(getenv("ATHENA_FUND_UPLOAD_DIR", ""))
	if err != nil {
		log.Fatalf("conversation store initialization failed: %v", err)
	}

	svc := server.New(server.Dependencies{
		Provider:      provider,
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      accountStore,
		Conversations: conversationStore,
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
