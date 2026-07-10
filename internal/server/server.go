package server

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/Super-Sky/athena-fund-assistant/internal/account"
	"github.com/Super-Sky/athena-fund-assistant/internal/athena"
	"github.com/Super-Sky/athena-fund-assistant/internal/conversation"
	"github.com/Super-Sky/athena-fund-assistant/internal/data"
	"github.com/Super-Sky/athena-fund-assistant/internal/decision"
	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
	"github.com/Super-Sky/athena-fund-assistant/internal/journal"
	"github.com/Super-Sky/athena-fund-assistant/internal/preference"
)

// Dependencies groups the app services used by HTTP handlers.
// Dependencies 汇总 HTTP handler 使用的应用服务。
type Dependencies struct {
	Provider      data.Provider
	DecisionMaker *decision.Engine
	Journals      *journal.MemoryStore
	Accounts      account.Store
	Conversations conversation.Store
	Preferences   preference.Store
	Athena        athena.Client
}

// Server maps fund-assistant MVP workflows to HTTP endpoints.
// Server 将基金助手 MVP 工作流映射为 HTTP 接口。
type Server struct {
	deps Dependencies
}

// New creates a server from explicit dependencies.
// New 使用显式依赖创建服务。
func New(deps Dependencies) *Server {
	return &Server{deps: deps}
}

// Routes returns the HTTP mux for the local MVP API.
// Routes 返回本地 MVP API 的 HTTP mux。
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/accounts/{user_id}/overview", s.handleAccountOverview)
	mux.HandleFunc("POST /api/accounts/{user_id}/holdings", s.handleReplaceAccountHoldings)
	mux.HandleFunc("GET /api/conversations/skills", s.handleConversationSkills)
	mux.HandleFunc("POST /api/conversations", s.handleCreateConversation)
	mux.HandleFunc("GET /api/conversations/{conversation_id}", s.handleConversationDetail)
	mux.HandleFunc("POST /api/conversations/{conversation_id}/messages", s.handleAddConversationMessage)
	mux.HandleFunc("POST /api/conversations/{conversation_id}/attachments", s.handleUploadConversationAttachment)
	mux.HandleFunc("GET /api/users/{user_id}/knowledge", s.handleKnowledgeWorkspace)
	mux.HandleFunc("POST /api/users/{user_id}/preferences/drafts", s.handlePreferenceDraft)
	mux.HandleFunc("POST /api/users/{user_id}/preferences/activate", s.handlePreferenceActivate)
	mux.HandleFunc("POST /api/users/{user_id}/knowledge/drafts", s.handleKnowledgeDraft)
	mux.HandleFunc("POST /api/users/{user_id}/knowledge/{item_id}/activate", s.handleKnowledgeActivate)
	mux.HandleFunc("POST /api/users/{user_id}/knowledge/{item_id}/rollback", s.handleKnowledgeRollback)
	mux.HandleFunc("POST /api/analysis/fund", s.handleFundAnalysis)
	mux.HandleFunc("POST /api/journals", s.handleCreateJournal)
	mux.HandleFunc("GET /internal/tools/catalog", s.handleRemoteToolCatalog)
	mux.HandleFunc("POST /internal/tools/execute", s.handleRemoteToolExecution)
	return withCORS(withJSON(mux))
}

type analysisRequest struct {
	InstrumentCode string                 `json:"instrument_code"`
	Profile        domain.InvestorProfile `json:"profile"`
	Portfolio      domain.Portfolio       `json:"portfolio"`
}

type analysisResponse struct {
	Profile        domain.InvestorProfile `json:"profile"`
	Portfolio      domain.Portfolio       `json:"portfolio"`
	FundSnapshot   domain.FundSnapshot    `json:"fund_snapshot"`
	Diagnosis      domain.Diagnosis       `json:"diagnosis"`
	DecisionMatrix domain.DecisionMatrix  `json:"decision_matrix"`
}

type journalRequest struct {
	Matrix           domain.DecisionMatrix `json:"matrix"`
	SelectedOptionID string                `json:"selected_option_id"`
	UserNotes        string                `json:"user_notes"`
}

type journalResponse struct {
	Journal domain.JournalEntry `json:"journal"`
	Review  domain.ReviewTask   `json:"review"`
}

type replaceHoldingsRequest struct {
	Holdings []domain.AccountHoldingSnapshot `json:"holdings"`
}

type createConversationRequest struct {
	UserID  string `json:"user_id"`
	SkillID string `json:"skill_id"`
	Title   string `json:"title"`
}

type addConversationMessageRequest struct {
	Role          string   `json:"role"`
	Content       string   `json:"content"`
	SkillID       string   `json:"skill_id"`
	AttachmentIDs []string `json:"attachment_ids"`
}

type activateRevisionRequest struct {
	RevisionID string `json:"revision_id"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleAccountOverview(w http.ResponseWriter, r *http.Request) {
	if s.deps.Accounts == nil {
		writeError(w, http.StatusServiceUnavailable, errText("account store is not configured"))
		return
	}
	overview, err := s.deps.Accounts.Overview(r.Context(), strings.TrimSpace(r.PathValue("user_id")))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func (s *Server) handleReplaceAccountHoldings(w http.ResponseWriter, r *http.Request) {
	if s.deps.Accounts == nil {
		writeError(w, http.StatusServiceUnavailable, errText("account store is not configured"))
		return
	}
	var req replaceHoldingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	overview, err := s.deps.Accounts.ReplaceHoldings(r.Context(), strings.TrimSpace(r.PathValue("user_id")), req.Holdings)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func (s *Server) handleConversationSkills(w http.ResponseWriter, r *http.Request) {
	if s.deps.Conversations == nil {
		writeError(w, http.StatusServiceUnavailable, errText("conversation store is not configured"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.deps.Conversations.Skills(r.Context())})
}

func (s *Server) handleCreateConversation(w http.ResponseWriter, r *http.Request) {
	if s.deps.Conversations == nil {
		writeError(w, http.StatusServiceUnavailable, errText("conversation store is not configured"))
		return
	}
	var req createConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	detail, err := s.deps.Conversations.Create(r.Context(), conversation.CreateInput{
		UserID:  strings.TrimSpace(req.UserID),
		SkillID: strings.TrimSpace(req.SkillID),
		Title:   strings.TrimSpace(req.Title),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, detail)
}

func (s *Server) handleConversationDetail(w http.ResponseWriter, r *http.Request) {
	if s.deps.Conversations == nil {
		writeError(w, http.StatusServiceUnavailable, errText("conversation store is not configured"))
		return
	}
	detail, err := s.deps.Conversations.Detail(r.Context(), strings.TrimSpace(r.PathValue("conversation_id")))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleAddConversationMessage(w http.ResponseWriter, r *http.Request) {
	if s.deps.Conversations == nil {
		writeError(w, http.StatusServiceUnavailable, errText("conversation store is not configured"))
		return
	}
	var req addConversationMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	conversationID := strings.TrimSpace(r.PathValue("conversation_id"))
	detail, err := s.deps.Conversations.AddMessage(r.Context(), conversationID, conversation.MessageInput{
		Role:          strings.TrimSpace(req.Role),
		Content:       strings.TrimSpace(req.Content),
		SkillID:       strings.TrimSpace(req.SkillID),
		AttachmentIDs: compactStrings(req.AttachmentIDs),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	role := strings.TrimSpace(req.Role)
	if s.deps.Athena != nil && (role == "" || role == "user") {
		if updated, runErr := s.startAthenaRunForMessage(r.Context(), conversationID, req); runErr == nil {
			detail = updated
		} else if updated, traceErr := s.deps.Conversations.RecordTrace(r.Context(), conversationID, conversation.TraceInput{
			Kind:    "athena_agent_run",
			Status:  "error",
			Summary: "Athena agent run request failed.",
			Metadata: map[string]string{
				"error": runErr.Error(),
			},
		}); traceErr == nil {
			detail = updated
		}
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleUploadConversationAttachment(w http.ResponseWriter, r *http.Request) {
	if s.deps.Conversations == nil {
		writeError(w, http.StatusServiceUnavailable, errText("conversation store is not configured"))
		return
	}
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	defer file.Close()
	attachment, err := s.deps.Conversations.SaveAttachment(r.Context(), strings.TrimSpace(r.PathValue("conversation_id")), conversation.AttachmentInput{
		UserID:      strings.TrimSpace(r.FormValue("user_id")),
		FileName:    header.Filename,
		ContentType: attachmentContentType(header),
		SizeBytes:   header.Size,
		Reader:      file,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, attachment)
}

func (s *Server) handleKnowledgeWorkspace(w http.ResponseWriter, r *http.Request) {
	if s.deps.Preferences == nil {
		writeError(w, http.StatusServiceUnavailable, errText("preference store is not configured"))
		return
	}
	workspace, err := s.deps.Preferences.Workspace(r.Context(), strings.TrimSpace(r.PathValue("user_id")))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, workspace)
}

func (s *Server) handlePreferenceDraft(w http.ResponseWriter, r *http.Request) {
	if s.deps.Preferences == nil {
		writeError(w, http.StatusServiceUnavailable, errText("preference store is not configured"))
		return
	}
	var input preference.PreferenceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	workspace, err := s.deps.Preferences.SavePreferenceDraft(r.Context(), strings.TrimSpace(r.PathValue("user_id")), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, workspace)
}

func (s *Server) handlePreferenceActivate(w http.ResponseWriter, r *http.Request) {
	if s.deps.Preferences == nil {
		writeError(w, http.StatusServiceUnavailable, errText("preference store is not configured"))
		return
	}
	var req activateRevisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	workspace, err := s.deps.Preferences.ActivatePreference(r.Context(), strings.TrimSpace(r.PathValue("user_id")), strings.TrimSpace(req.RevisionID))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, workspace)
}

func (s *Server) handleKnowledgeDraft(w http.ResponseWriter, r *http.Request) {
	if s.deps.Preferences == nil {
		writeError(w, http.StatusServiceUnavailable, errText("preference store is not configured"))
		return
	}
	var input preference.KnowledgeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	workspace, err := s.deps.Preferences.SaveKnowledgeDraft(r.Context(), strings.TrimSpace(r.PathValue("user_id")), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, workspace)
}

func (s *Server) handleKnowledgeActivate(w http.ResponseWriter, r *http.Request) {
	if s.deps.Preferences == nil {
		writeError(w, http.StatusServiceUnavailable, errText("preference store is not configured"))
		return
	}
	var req activateRevisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	workspace, err := s.deps.Preferences.ActivateKnowledge(r.Context(), strings.TrimSpace(r.PathValue("user_id")), strings.TrimSpace(r.PathValue("item_id")), strings.TrimSpace(req.RevisionID))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, workspace)
}

func (s *Server) handleKnowledgeRollback(w http.ResponseWriter, r *http.Request) {
	if s.deps.Preferences == nil {
		writeError(w, http.StatusServiceUnavailable, errText("preference store is not configured"))
		return
	}
	var req activateRevisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	workspace, err := s.deps.Preferences.RollbackKnowledge(r.Context(), strings.TrimSpace(r.PathValue("user_id")), strings.TrimSpace(r.PathValue("item_id")), strings.TrimSpace(req.RevisionID))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, workspace)
}

func (s *Server) handleFundAnalysis(w http.ResponseWriter, r *http.Request) {
	var req analysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.InstrumentCode == "" {
		writeError(w, http.StatusBadRequest, errText("instrument_code is required"))
		return
	}

	snapshot, err := s.deps.Provider.GetFundSnapshot(r.Context(), req.InstrumentCode)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	diagnosis, matrix, err := s.deps.DecisionMaker.Generate(req.Profile, req.Portfolio, snapshot)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, analysisResponse{
		Profile:        req.Profile,
		Portfolio:      req.Portfolio,
		FundSnapshot:   snapshot,
		Diagnosis:      diagnosis,
		DecisionMatrix: matrix,
	})
}

func (s *Server) handleCreateJournal(w http.ResponseWriter, r *http.Request) {
	var req journalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	entry, review, err := s.deps.Journals.Create(req.Matrix, req.SelectedOptionID, req.UserNotes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, journalResponse{Journal: entry, Review: review})
}

type errText string

func (e errText) Error() string {
	return string(e)
}

func withJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); isLocalOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions && (strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz") {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isLocalOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	return strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:")
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func attachmentContentType(header *multipart.FileHeader) string {
	if header == nil {
		return ""
	}
	if values := header.Header.Values("Content-Type"); len(values) > 0 {
		return values[0]
	}
	return ""
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
