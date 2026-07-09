package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

	journalRead := performJSON(t, srv, "GET", "/api/journals/"+journalResp.Journal.ID, nil)
	if journalRead.Code != http.StatusOK {
		t.Fatalf("journal read status=%d body=%s", journalRead.Code, journalRead.Body.String())
	}
	if !bytes.Contains(journalRead.Body.Bytes(), []byte(journalResp.Journal.ID)) {
		t.Fatalf("journal read body=%s, want journal id", journalRead.Body.String())
	}

	reviewRead := performJSON(t, srv, "GET", "/api/reviews/"+journalResp.Review.ID, nil)
	if reviewRead.Code != http.StatusOK {
		t.Fatalf("review read status=%d body=%s", reviewRead.Code, reviewRead.Body.String())
	}
	if !bytes.Contains(reviewRead.Body.Bytes(), []byte(journalResp.Review.ID)) {
		t.Fatalf("review read body=%s, want review id", reviewRead.Body.String())
	}
}

func TestReadinessAndMissingJournal(t *testing.T) {
	srv := New(Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
	}).Routes()

	ready := performJSON(t, srv, "GET", "/readyz", nil)
	if ready.Code != http.StatusOK {
		t.Fatalf("ready status=%d body=%s", ready.Code, ready.Body.String())
	}

	missing := performJSON(t, srv, "GET", "/api/journals/missing", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing journal status=%d body=%s", missing.Code, missing.Body.String())
	}
}

func TestCORSPreflightForLocalWeb(t *testing.T) {
	srv := New(Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
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
