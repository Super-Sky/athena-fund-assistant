package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/account"
	"github.com/Super-Sky/athena-fund-assistant/internal/conversation"
	"github.com/Super-Sky/athena-fund-assistant/internal/data"
	"github.com/Super-Sky/athena-fund-assistant/internal/decision"
	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
	"github.com/Super-Sky/athena-fund-assistant/internal/journal"
)

func TestFundAnalysisAndJournalWorkflow(t *testing.T) {
	conversations, err := conversation.NewMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("new conversation store: %v", err)
	}
	srv := New(Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      account.NewMemoryStore(),
		Conversations: conversations,
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
	conversations, err := conversation.NewMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("new conversation store: %v", err)
	}
	srv := New(Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      account.NewMemoryStore(),
		Conversations: conversations,
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
	conversations, err := conversation.NewMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("new conversation store: %v", err)
	}
	srv := New(Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      account.NewMemoryStore(),
		Conversations: conversations,
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

func TestConversationWorkspaceWorkflow(t *testing.T) {
	conversations, err := conversation.NewMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("new conversation store: %v", err)
	}
	srv := New(Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      account.NewMemoryStore(),
		Conversations: conversations,
	}).Routes()

	skillsRR := httptest.NewRecorder()
	srv.ServeHTTP(skillsRR, httptest.NewRequest(http.MethodGet, "/api/conversations/skills", nil))
	if skillsRR.Code != http.StatusOK {
		t.Fatalf("skills status=%d body=%s", skillsRR.Code, skillsRR.Body.String())
	}

	createRR := performJSON(t, srv, http.MethodPost, "/api/conversations", createConversationRequest{
		UserID:  "demo-user",
		SkillID: "document_intake",
		Title:   "账单解析",
	})
	if createRR.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createRR.Code, createRR.Body.String())
	}
	var detail domain.ConversationDetail
	if err := json.Unmarshal(createRR.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode conversation: %v", err)
	}

	var uploadBody bytes.Buffer
	writer := multipart.NewWriter(&uploadBody)
	part, err := writer.CreateFormFile("file", "statement.txt")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte("510300 holding note")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.WriteField("user_id", "demo-user"); err != nil {
		t.Fatalf("write multipart field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	uploadReq := httptest.NewRequest(http.MethodPost, "/api/conversations/"+detail.Session.ID+"/attachments", &uploadBody)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadRR := httptest.NewRecorder()
	srv.ServeHTTP(uploadRR, uploadReq)
	if uploadRR.Code != http.StatusCreated {
		t.Fatalf("upload status=%d body=%s", uploadRR.Code, uploadRR.Body.String())
	}
	var attachment domain.ConversationAttachment
	if err := json.Unmarshal(uploadRR.Body.Bytes(), &attachment); err != nil {
		t.Fatalf("decode attachment: %v", err)
	}
	if attachment.Status != "pending_parse" {
		t.Fatalf("attachment = %#v", attachment)
	}

	messageRR := performJSON(t, srv, http.MethodPost, "/api/conversations/"+detail.Session.ID+"/messages", addConversationMessageRequest{
		Role:          "user",
		Content:       "请结合附件做一个复盘计划。",
		SkillID:       "document_intake",
		AttachmentIDs: []string{attachment.ID},
	})
	if messageRR.Code != http.StatusOK {
		t.Fatalf("message status=%d body=%s", messageRR.Code, messageRR.Body.String())
	}
	var updated domain.ConversationDetail
	if err := json.Unmarshal(messageRR.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated detail: %v", err)
	}
	if len(updated.Messages) != 1 || len(updated.Attachments) != 1 {
		t.Fatalf("updated detail = %#v", updated)
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
