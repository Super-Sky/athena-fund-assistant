package server

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/account"
	"github.com/Super-Sky/athena-fund-assistant/internal/athena"
	"github.com/Super-Sky/athena-fund-assistant/internal/authorization"
	"github.com/Super-Sky/athena-fund-assistant/internal/conversation"
	"github.com/Super-Sky/athena-fund-assistant/internal/data"
	"github.com/Super-Sky/athena-fund-assistant/internal/decision"
	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
	"github.com/Super-Sky/athena-fund-assistant/internal/journal"
	"github.com/Super-Sky/athena-fund-assistant/internal/preference"
)

func TestFundAnalysisAndJournalWorkflow(t *testing.T) {
	conversations, err := conversation.NewMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("new conversation store: %v", err)
	}
	srv := newAuthenticatedTestHarness(t, Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      account.NewMemoryStore(),
		Conversations: conversations,
	}).Handler

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
	if !analysisResp.Governance.Allowed() {
		t.Fatalf("analysis governance = %#v, want deliverable output", analysisResp.Governance)
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
	journalRead := performJSON(t, srv, http.MethodGet, "/api/journals/"+journalResp.Journal.ID, nil)
	if journalRead.Code != http.StatusOK || !bytes.Contains(journalRead.Body.Bytes(), []byte(journalResp.Journal.ID)) {
		t.Fatalf("journal read status=%d body=%s", journalRead.Code, journalRead.Body.String())
	}
	reviewRead := performJSON(t, srv, http.MethodGet, "/api/reviews/"+journalResp.Review.ID, nil)
	if reviewRead.Code != http.StatusOK || !bytes.Contains(reviewRead.Body.Bytes(), []byte(journalResp.Review.ID)) {
		t.Fatalf("review read status=%d body=%s", reviewRead.Code, reviewRead.Body.String())
	}
	ready := performJSON(t, srv, http.MethodGet, "/readyz", nil)
	if ready.Code != http.StatusOK {
		t.Fatalf("ready status=%d body=%s", ready.Code, ready.Body.String())
	}
}

func TestCORSPreflightForLocalWeb(t *testing.T) {
	conversations, err := conversation.NewMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("new conversation store: %v", err)
	}
	srv := newAuthenticatedTestHarness(t, Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      account.NewMemoryStore(),
		Conversations: conversations,
	}).Handler

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
	srv := newAuthenticatedTestHarness(t, Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      account.NewMemoryStore(),
		Conversations: conversations,
	}).Handler

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
	srv := newAuthenticatedTestHarness(t, Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      account.NewMemoryStore(),
		Conversations: conversations,
	}).Handler

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
	if detail.Messages == nil || detail.Attachments == nil || detail.Trace == nil {
		t.Fatalf("conversation API collections must decode from JSON arrays: body=%s", createRR.Body.String())
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

func TestRemoteToolCatalogAndExecution(t *testing.T) {
	conversations, err := conversation.NewMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("new conversation store: %v", err)
	}
	harness := newAuthenticatedTestHarness(t, Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      account.NewMemoryStore(),
		Conversations: conversations,
	})
	srv := harness.Handler

	catalogRR := httptest.NewRecorder()
	catalogReq := httptest.NewRequest(http.MethodGet, "/internal/tools/catalog?base_url=http://127.0.0.1:8081", nil)
	srv.ServeHTTP(catalogRR, catalogReq)
	if catalogRR.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", catalogRR.Code, catalogRR.Body.String())
	}
	var catalog struct {
		ContractVersion string                   `json:"contract_version"`
		Items           []remoteToolRegistration `json:"items"`
	}
	if err := json.Unmarshal(catalogRR.Body.Bytes(), &catalog); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	if catalog.ContractVersion != remoteToolContractVersion || len(catalog.Items) != 2 {
		t.Fatalf("unexpected catalog = %#v", catalog)
	}
	if catalog.Items[0].Endpoint != "http://127.0.0.1:8081/internal/tools/execute" {
		t.Fatalf("unexpected endpoint %q", catalog.Items[0].Endpoint)
	}

	overviewRR := performJSON(t, srv, http.MethodPost, "/internal/tools/execute", remoteToolExecutionRequest{
		ContractVersion: remoteToolContractVersion,
		RequestID:       "req_account",
		ToolCallID:      "call_account",
		RegistrationID:  "fund_account_overview_v1",
		AppID:           "athena-fund-assistant",
		ToolName:        "account_overview",
		Arguments:       mustJSON(t, accountOverviewToolArgs{UserID: "demo-user", ConsentGrantRef: harness.Grant.Ref}),
		Attempt:         1,
	})
	if overviewRR.Code != http.StatusOK {
		t.Fatalf("overview tool status=%d body=%s", overviewRR.Code, overviewRR.Body.String())
	}
	var overviewResult remoteToolExecutionResponse
	if err := json.Unmarshal(overviewRR.Body.Bytes(), &overviewResult); err != nil {
		t.Fatalf("decode overview tool: %v", err)
	}
	if overviewResult.Status != "ok" || overviewResult.ToolCallID != "call_account" || overviewResult.RequestID != "req_account" {
		t.Fatalf("unexpected overview result = %#v", overviewResult)
	}
	var overviewContent struct {
		Tool     string                 `json:"tool"`
		Overview domain.AccountOverview `json:"overview"`
	}
	if err := json.Unmarshal([]byte(overviewResult.Content), &overviewContent); err != nil {
		t.Fatalf("decode overview content: %v", err)
	}
	if overviewContent.Tool != "account_overview" || overviewContent.Overview.Account.UserID != "demo-user" {
		t.Fatalf("unexpected overview content = %#v", overviewContent)
	}

	snapshotRR := performJSON(t, srv, http.MethodPost, "/internal/tools/execute", remoteToolExecutionRequest{
		ContractVersion: remoteToolContractVersion,
		RequestID:       "req_snapshot",
		ToolCallID:      "call_snapshot",
		RegistrationID:  "fund_market_snapshot_v1",
		AppID:           "athena-fund-assistant",
		ToolName:        "fund_market_snapshot",
		Arguments:       json.RawMessage(`{"instrument_code":"QQQ"}`),
		Attempt:         1,
	})
	if snapshotRR.Code != http.StatusOK {
		t.Fatalf("snapshot tool status=%d body=%s", snapshotRR.Code, snapshotRR.Body.String())
	}
	var snapshotResult remoteToolExecutionResponse
	if err := json.Unmarshal(snapshotRR.Body.Bytes(), &snapshotResult); err != nil {
		t.Fatalf("decode snapshot tool: %v", err)
	}
	var snapshotContent struct {
		Tool         string              `json:"tool"`
		FundSnapshot domain.FundSnapshot `json:"fund_snapshot"`
	}
	if err := json.Unmarshal([]byte(snapshotResult.Content), &snapshotContent); err != nil {
		t.Fatalf("decode snapshot content: %v", err)
	}
	if snapshotContent.Tool != "fund_market_snapshot" || snapshotContent.FundSnapshot.Instrument.Code != "QQQ" {
		t.Fatalf("unexpected snapshot content = %#v", snapshotContent)
	}
	if snapshotContent.FundSnapshot.Metadata.Provider == "" || snapshotContent.FundSnapshot.Metadata.Timezone != "America/New_York" {
		t.Fatalf("snapshot metadata missing provider/timezone: %#v", snapshotContent.FundSnapshot.Metadata)
	}

	unknownRR := performJSON(t, srv, http.MethodPost, "/internal/tools/execute", remoteToolExecutionRequest{
		ContractVersion: remoteToolContractVersion,
		RequestID:       "req_unknown",
		ToolCallID:      "call_unknown",
		AppID:           "athena-fund-assistant",
		ToolName:        "place_order",
		Arguments:       json.RawMessage(`{}`),
		Attempt:         1,
	})
	if unknownRR.Code != http.StatusBadRequest {
		t.Fatalf("unknown status=%d body=%s", unknownRR.Code, unknownRR.Body.String())
	}
	var unknownResult remoteToolExecutionResponse
	if err := json.Unmarshal(unknownRR.Body.Bytes(), &unknownResult); err != nil {
		t.Fatalf("decode unknown tool: %v", err)
	}
	if unknownResult.Status != "error" || unknownResult.Error == nil || unknownResult.Error.Code != "unknown_tool" {
		t.Fatalf("unexpected unknown result = %#v", unknownResult)
	}
}

func TestConversationMessageStartsAthenaRun(t *testing.T) {
	conversations, err := conversation.NewMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("new conversation store: %v", err)
	}
	srv := newAuthenticatedTestHarness(t, Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      account.NewMemoryStore(),
		Conversations: conversations,
		Athena:        athena.MockClient{},
	}).Handler

	createRR := performJSON(t, srv, http.MethodPost, "/api/conversations", createConversationRequest{
		UserID:  "demo-user",
		SkillID: "portfolio_review",
		Title:   "Athena run",
	})
	if createRR.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createRR.Code, createRR.Body.String())
	}
	var detail domain.ConversationDetail
	if err := json.Unmarshal(createRR.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode conversation: %v", err)
	}

	messageRR := performJSON(t, srv, http.MethodPost, "/api/conversations/"+detail.Session.ID+"/messages", addConversationMessageRequest{
		Role:    "user",
		Content: "请读取账户概览并给我一个复盘重点。",
		SkillID: "portfolio_review",
	})
	if messageRR.Code != http.StatusOK {
		t.Fatalf("message status=%d body=%s", messageRR.Code, messageRR.Body.String())
	}
	var updated domain.ConversationDetail
	if err := json.Unmarshal(messageRR.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated detail: %v", err)
	}
	if !hasTraceStatus(updated.Trace, "athena_agent_run", "ok") {
		t.Fatalf("trace missing accepted Athena run: %#v", updated.Trace)
	}
}

func TestPreferenceKnowledgeWorkflow(t *testing.T) {
	conversations, err := conversation.NewMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("new conversation store: %v", err)
	}
	srv := newAuthenticatedTestHarness(t, Dependencies{
		Provider:      data.NewMockProvider(),
		DecisionMaker: decision.NewEngine(),
		Journals:      journal.NewMemoryStore(),
		Accounts:      account.NewMemoryStore(),
		Conversations: conversations,
		Preferences:   preference.NewMemoryStore(),
	}).Handler

	workspaceRR := httptest.NewRecorder()
	srv.ServeHTTP(workspaceRR, httptest.NewRequest(http.MethodGet, "/api/users/demo-user/knowledge", nil))
	if workspaceRR.Code != http.StatusOK {
		t.Fatalf("workspace status=%d body=%s", workspaceRR.Code, workspaceRR.Body.String())
	}
	var workspace domain.KnowledgeWorkspace
	if err := json.Unmarshal(workspaceRR.Body.Bytes(), &workspace); err != nil {
		t.Fatalf("decode workspace: %v", err)
	}
	if workspace.Preference.ActiveRevisionID == "" || len(workspace.Items) == 0 || len(workspace.Audit) == 0 {
		t.Fatalf("unexpected seeded workspace: %#v", workspace)
	}

	draftRR := performJSON(t, srv, http.MethodPost, "/api/users/demo-user/knowledge/drafts", preference.KnowledgeInput{
		Title:      "Volatility review rule",
		Category:   "review_rule",
		Content:    "When volatility is above 18%, require a seven-day review before adding exposure.",
		Tags:       []string{"volatility", "review"},
		Source:     "server_test",
		Author:     "tester",
		Confidence: 0.8,
		Summary:    "Add volatility review rule",
	})
	if draftRR.Code != http.StatusCreated {
		t.Fatalf("draft status=%d body=%s", draftRR.Code, draftRR.Body.String())
	}
	var drafted domain.KnowledgeWorkspace
	if err := json.Unmarshal(draftRR.Body.Bytes(), &drafted); err != nil {
		t.Fatalf("decode draft: %v", err)
	}
	var item domain.KnowledgeItem
	for _, candidate := range drafted.Items {
		if candidate.Title == "Volatility review rule" {
			item = candidate
			break
		}
	}
	if item.ID == "" {
		t.Fatalf("draft item not found: %#v", drafted.Items)
	}
	revision := drafted.Revisions[len(drafted.Revisions)-1]
	if item.Status != domain.KnowledgeDraft || revision.Status != domain.KnowledgeDraft {
		t.Fatalf("expected draft item and revision, got item=%s revision=%s", item.Status, revision.Status)
	}

	activateRR := performJSON(t, srv, http.MethodPost, "/api/users/demo-user/knowledge/"+item.ID+"/activate", activateRevisionRequest{RevisionID: revision.ID})
	if activateRR.Code != http.StatusOK {
		t.Fatalf("activate status=%d body=%s", activateRR.Code, activateRR.Body.String())
	}
	var activated domain.KnowledgeWorkspace
	if err := json.Unmarshal(activateRR.Body.Bytes(), &activated); err != nil {
		t.Fatalf("decode activated: %v", err)
	}
	found := false
	for _, value := range activated.Items {
		if value.ID == item.ID && value.Status == domain.KnowledgeActive && value.ActiveRevisionID == revision.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("activated item not found: %#v", activated.Items)
	}
}

type authenticatedTestHarness struct {
	Handler http.Handler
	Grant   authorization.ConsentGrant
}

func newAuthenticatedTestHarness(t *testing.T, deps Dependencies) authenticatedTestHarness {
	t.Helper()
	store := authorization.NewMemoryStore()
	service := authorization.NewService(store)
	session, err := service.IssueSession(context.Background(), "demo-user", time.Hour)
	if err != nil {
		t.Fatalf("issue test session: %v", err)
	}
	grant, err := service.CreateGrant(context.Background(), authorization.CreateGrantRequest{
		Subject:  "demo-user",
		Audience: athenaAudience,
		Scopes: []authorization.Scope{
			authorization.ScopeAccountSummaryRead,
			authorization.ScopeHoldingSnapshotRead,
		},
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("create test consent grant: %v", err)
	}
	deps.Authorization = service
	deps.LocalAuthSubject = "demo-user"
	deps.RemoteToolToken = "server-test-service-token"
	routes := New(deps).Routes()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			switch {
			case r.URL.Path == "/internal/tools/execute":
				r.Header.Set("Authorization", "Bearer server-test-service-token")
			case strings.HasPrefix(r.URL.Path, "/api/"):
				r.Header.Set("Authorization", "Bearer "+session.Token)
			}
		}
		routes.ServeHTTP(w, r)
	})
	return authenticatedTestHarness{Handler: handler, Grant: grant}
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

func hasTraceStatus(events []domain.ConversationTraceEvent, kind, status string) bool {
	for _, event := range events {
		if event.Kind == kind && event.Status == status {
			return true
		}
	}
	return false
}
