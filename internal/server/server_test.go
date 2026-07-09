package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/account"
	"github.com/Super-Sky/athena-fund-assistant/internal/data"
	"github.com/Super-Sky/athena-fund-assistant/internal/decision"
	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
	"github.com/Super-Sky/athena-fund-assistant/internal/journal"
)

func TestFundAnalysisAndJournalWorkflow(t *testing.T) {
	srv := New(Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      account.NewMemoryStore(),
	}).Routes()

	analysis := analysisRequest{
		InstrumentCode: "510300",
		Profile: domain.InvestorProfile{
			RiskPreference:                   domain.RiskBalanced,
			InvestmentHorizonMonths:          24,
			MaxAcceptableDrawdownPct:         25,
			SingleInstrumentMaxAllocationPct: 20,
			CashPreferencePct:                8,
			DefaultDecisionStyle:             "three_options",
		},
		Portfolio: domain.Portfolio{
			Holdings: []domain.PortfolioHolding{{
				InstrumentCode: "510300",
				InstrumentName: "Sample CSI 300 ETF",
				Market:         "CN",
				Currency:       "CNY",
				HoldingAmount:  50000,
				CostBasis:      4.2,
				AllocationPct:  22,
				UserThesis:     "broad China equity beta",
			}},
		},
	}

	rr := performJSON(t, srv, "POST", "/api/analysis/fund", analysis)
	if rr.Code != http.StatusOK {
		t.Fatalf("analysis status=%d body=%s", rr.Code, rr.Body.String())
	}
	var analysisResp analysisResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &analysisResp); err != nil {
		t.Fatalf("decode analysis: %v", err)
	}
	if err := analysisResp.DecisionMatrix.Validate(); err != nil {
		t.Fatalf("matrix invalid: %v", err)
	}

	journalReq := journalRequest{
		Matrix:           analysisResp.DecisionMatrix,
		SelectedOptionID: "option_balanced",
		UserNotes:        "first MVP test journal",
	}
	journalRR := performJSON(t, srv, "POST", "/api/journals", journalReq)
	if journalRR.Code != http.StatusCreated {
		t.Fatalf("journal status=%d body=%s", journalRR.Code, journalRR.Body.String())
	}
	var journalResp journalResponse
	if err := json.Unmarshal(journalRR.Body.Bytes(), &journalResp); err != nil {
		t.Fatalf("decode journal: %v", err)
	}
	if journalResp.Journal.SelectedOptionID != "option_balanced" {
		t.Fatalf("unexpected selected option %s", journalResp.Journal.SelectedOptionID)
	}
	if journalResp.Review.Status != "open" {
		t.Fatalf("unexpected review status %s", journalResp.Review.Status)
	}
}

func TestCORSPreflightForLocalWeb(t *testing.T) {
	srv := New(Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      account.NewMemoryStore(),
	}).Routes()

	req := httptest.NewRequest(http.MethodOptions, "/api/analysis/fund", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("preflight status=%d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5173" {
		t.Fatalf("unexpected allow origin %q", got)
	}
}

func TestAccountOverviewAndManualHoldingWorkflow(t *testing.T) {
	srv := New(Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      account.NewMemoryStore(),
	}).Routes()

	overviewRR := httptest.NewRecorder()
	overviewReq := httptest.NewRequest(http.MethodGet, "/api/accounts/demo-user/overview", nil)
	srv.ServeHTTP(overviewRR, overviewReq)
	if overviewRR.Code != http.StatusOK {
		t.Fatalf("overview status=%d body=%s", overviewRR.Code, overviewRR.Body.String())
	}
	var overview domain.AccountOverview
	if err := json.Unmarshal(overviewRR.Body.Bytes(), &overview); err != nil {
		t.Fatalf("decode overview: %v", err)
	}
	if overview.BaseCurrency != "CNY" || len(overview.Holdings) == 0 || len(overview.PerformanceTrend) == 0 {
		t.Fatalf("unexpected overview = %#v", overview)
	}
	if !overview.Trace.MockDataTemporary {
		t.Fatalf("overview trace = %#v, want mock temporary marker", overview.Trace)
	}

	now := time.Now().UTC()
	replaceRR := performJSON(t, srv, http.MethodPost, "/api/accounts/demo-user/holdings", replaceHoldingsRequest{
		Holdings: []domain.AccountHoldingSnapshot{{
			InstrumentCode:    "QQQ",
			InstrumentName:    "Manual Nasdaq 100 ETF",
			Market:            "US",
			Currency:          "USD",
			Units:             10,
			CostBasis:         400,
			CurrentPrice:      450,
			FXToBase:          7.2,
			DataAuthorization: "manual_entry",
			Metadata: domain.SourceMetadata{
				Source:        "server_test",
				Provider:      "manual",
				FetchedAt:     now,
				MarketTime:    now,
				Timezone:      "America/New_York",
				Delay:         "0m",
				LicenseTerms:  "test",
				Confidence:    0.9,
				SchemaVersion: "account_snapshot.v1",
			},
		}},
	})
	if replaceRR.Code != http.StatusOK {
		t.Fatalf("replace status=%d body=%s", replaceRR.Code, replaceRR.Body.String())
	}
	var replaced domain.AccountOverview
	if err := json.Unmarshal(replaceRR.Body.Bytes(), &replaced); err != nil {
		t.Fatalf("decode replaced overview: %v", err)
	}
	if replaced.TotalMarketValue != 32400 {
		t.Fatalf("total market value = %.2f, want 32400", replaced.TotalMarketValue)
	}
	if replaced.Holdings[0].Metadata.Timezone == "" {
		t.Fatalf("holding metadata missing timezone: %#v", replaced.Holdings[0].Metadata)
	}
}

func performJSON(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}
