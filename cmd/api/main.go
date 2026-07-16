package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/account"
	"github.com/Super-Sky/athena-fund-assistant/internal/athena"
	"github.com/Super-Sky/athena-fund-assistant/internal/authorization"
	"github.com/Super-Sky/athena-fund-assistant/internal/conversation"
	"github.com/Super-Sky/athena-fund-assistant/internal/data"
	"github.com/Super-Sky/athena-fund-assistant/internal/decision"
	"github.com/Super-Sky/athena-fund-assistant/internal/journal"
	"github.com/Super-Sky/athena-fund-assistant/internal/preference"
	"github.com/Super-Sky/athena-fund-assistant/internal/server"
)

func main() {
	addr := getenv("ATHENA_FUND_API_ADDR", ":8081")
	provider, validationOptions := loadProvider()
	report := data.ValidateProvider(context.Background(), provider, validationOptions)
	if !report.Passed {
		log.Fatalf("data provider validation failed: %+v", report.Checks)
	}
	log.Printf("data provider validation passed with %d checks", len(report.Checks))
	var accountStore account.Store = account.NewMemoryStore()
	var journalStore journal.Store = journal.NewMemoryStore()
	var authorizationStore authorization.Store = authorization.NewMemoryStore()
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
		postgresJournalStore, err := journal.NewPostgresStore(context.Background(), databaseURL)
		if err != nil {
			log.Fatalf("postgres journal store initialization failed: %v", err)
		}
		defer func() {
			if err := postgresJournalStore.Close(context.Background()); err != nil {
				log.Printf("postgres journal store close failed: %v", err)
			}
		}()
		journalStore = postgresJournalStore
		log.Printf("journal store using PostgreSQL")
		postgresAuthorizationStore, err := authorization.NewPostgresStore(context.Background(), databaseURL)
		if err != nil {
			log.Fatalf("postgres authorization store initialization failed: %v", err)
		}
		defer func() {
			if err := postgresAuthorizationStore.Close(context.Background()); err != nil {
				log.Printf("postgres authorization store close failed: %v", err)
			}
		}()
		authorizationStore = postgresAuthorizationStore
		log.Printf("authorization store using PostgreSQL")
	} else {
		log.Printf("account store using in-memory demo data")
		log.Printf("journal store using non-durable in-memory demo data")
		log.Printf("authorization store using non-durable in-memory data")
	}
	conversationStore, err := conversation.NewMemoryStore(getenv("ATHENA_FUND_UPLOAD_DIR", ""))
	if err != nil {
		log.Fatalf("conversation store initialization failed: %v", err)
	}
	var athenaClient athena.Client = athena.MockClient{}
	if athenaBaseURL := os.Getenv("ATHENA_BASE_URL"); athenaBaseURL != "" {
		client, err := athena.NewHTTPClient(athenaBaseURL, os.Getenv("ATHENA_AUTH_TOKEN"))
		if err != nil {
			log.Fatalf("athena client initialization failed: %v", err)
		}
		athenaClient = client
		log.Printf("athena client using %s", athenaBaseURL)
	} else {
		log.Printf("athena client using local mock")
	}

	svc := server.New(server.Dependencies{
		Provider:         provider,
		DecisionMaker:    decision.NewEngine(),
		Journals:         journalStore,
		Accounts:         accountStore,
		Conversations:    conversationStore,
		Preferences:      preference.NewMemoryStore(),
		Athena:           athenaClient,
		Authorization:    authorization.NewService(authorizationStore),
		LocalAuthSubject: getenv("ATHENA_FUND_LOCAL_AUTH_SUBJECT", "demo-user"),
		RemoteToolToken:  os.Getenv("ATHENA_FUND_REMOTE_TOOL_TOKEN"),
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

func loadProvider() (data.Provider, data.ValidationOptions) {
	switch os.Getenv("ATHENA_FUND_PROVIDER") {
	case "", "mock":
		log.Printf("data provider using mock provider")
		return data.NewMockProvider(), data.ValidationOptions{
			FundCodes:     []string{"QQQ"},
			EquitySymbols: []string{"AAPL"},
			IndexCodes:    []string{"NDX"},
			FXPairs:       []data.FXPair{{BaseCurrency: "USD", QuoteCurrency: "CNY"}},
			Calendars:     []data.CalendarProbe{{Market: "US", Date: time.Now().UTC()}},
		}
	case "csv":
		csvPath := os.Getenv("ATHENA_FUND_CSV_PATH")
		provider, err := data.NewCSVProvider(csvPath)
		if err != nil {
			log.Fatalf("csv data provider initialization failed: %v", err)
		}
		log.Printf("data provider using user-supplied csv data from %s", csvPath)
		probeDate := time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC)
		return provider, data.ValidationOptions{
			FundCodes:     []string{"510300", "QQQ"},
			EquitySymbols: []string{"AAPL"},
			IndexCodes:    []string{"000300", "NDX"},
			FXPairs:       []data.FXPair{{BaseCurrency: "USD", QuoteCurrency: "CNY"}},
			Calendars: []data.CalendarProbe{
				{Market: "CN", Date: probeDate},
				{Market: "US", Date: probeDate},
			},
		}
	default:
		log.Fatalf("unsupported ATHENA_FUND_PROVIDER %q", os.Getenv("ATHENA_FUND_PROVIDER"))
		return nil, data.ValidationOptions{}
	}
}
