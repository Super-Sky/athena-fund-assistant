package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/data"
	"github.com/Super-Sky/athena-fund-assistant/internal/decision"
	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
	"github.com/Super-Sky/athena-fund-assistant/internal/journal"
)

// Dependencies groups the app services used by HTTP handlers.
// Dependencies 汇总 HTTP handler 使用的应用服务。
type Dependencies struct {
	Provider      data.Provider
	DecisionMaker *decision.Engine
	Journals      journal.Store
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
	mux.HandleFunc("GET /readyz", s.handleReady)
	mux.HandleFunc("POST /api/analysis/fund", s.handleFundAnalysis)
	mux.HandleFunc("POST /api/journals", s.handleCreateJournal)
	mux.HandleFunc("GET /api/journals/{journalID}", s.handleGetJournal)
	mux.HandleFunc("GET /api/reviews/{reviewID}", s.handleGetReview)
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

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.deps.Journals.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, errText("journal store unavailable"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
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
	if err := req.Matrix.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !hasDecisionOption(req.Matrix, req.SelectedOptionID) {
		writeError(w, http.StatusBadRequest, errText("selected option not found"))
		return
	}
	entry, review, err := s.deps.Journals.Create(r.Context(), req.Matrix, req.SelectedOptionID, req.UserNotes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errText("journal store write failed"))
		return
	}
	writeJSON(w, http.StatusCreated, journalResponse{Journal: entry, Review: review})
}

func hasDecisionOption(matrix domain.DecisionMatrix, optionID string) bool {
	for _, option := range matrix.Options {
		if option.ID == optionID {
			return true
		}
	}
	return false
}

func (s *Server) handleGetJournal(w http.ResponseWriter, r *http.Request) {
	entry, err := s.deps.Journals.Entry(r.Context(), r.PathValue("journalID"))
	if err != nil {
		writeStoreReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]domain.JournalEntry{"journal": entry})
}

func (s *Server) handleGetReview(w http.ResponseWriter, r *http.Request) {
	review, err := s.deps.Journals.Review(r.Context(), r.PathValue("reviewID"))
	if err != nil {
		writeStoreReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]domain.ReviewTask{"review": review})
}

func writeStoreReadError(w http.ResponseWriter, err error) {
	if errors.Is(err, journal.ErrEntryNotFound) || errors.Is(err, journal.ErrReviewNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeError(w, http.StatusInternalServerError, errText("journal store read failed"))
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
		if r.Method == http.MethodOptions && (strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz" || r.URL.Path == "/readyz") {
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
