package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Super-Sky/athena-fund-assistant/internal/authorization"
)

const remoteToolContractVersion = "remote_tool_execution.v1"

type remoteToolRegistration struct {
	RegistrationID   string         `json:"registration_id"`
	AppID            string         `json:"app_id"`
	Name             string         `json:"name"`
	Description      string         `json:"description,omitempty"`
	Parameters       map[string]any `json:"parameters,omitempty"`
	Endpoint         string         `json:"endpoint"`
	ToolScope        string         `json:"tool_scope,omitempty"`
	Operation        string         `json:"operation,omitempty"`
	RiskLevel        string         `json:"risk_level,omitempty"`
	SideEffectLevel  string         `json:"side_effect_level,omitempty"`
	Idempotent       bool           `json:"idempotent,omitempty"`
	TimeoutMS        int            `json:"timeout_ms,omitempty"`
	RetryMaxAttempts int            `json:"retry_max_attempts,omitempty"`
	Enabled          bool           `json:"enabled"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type remoteToolExecutionRequest struct {
	ContractVersion string          `json:"contract_version"`
	RequestID       string          `json:"request_id"`
	ToolCallID      string          `json:"tool_call_id"`
	RegistrationID  string          `json:"registration_id"`
	AppID           string          `json:"app_id"`
	ToolName        string          `json:"tool_name"`
	Arguments       json.RawMessage `json:"arguments"`
	Attempt         int             `json:"attempt"`
	Metadata        map[string]any  `json:"metadata,omitempty"`
}

type remoteToolExecutionResponse struct {
	ContractVersion string                    `json:"contract_version"`
	RequestID       string                    `json:"request_id,omitempty"`
	ToolCallID      string                    `json:"tool_call_id"`
	Status          string                    `json:"status"`
	Content         string                    `json:"content,omitempty"`
	Error           *remoteToolExecutionError `json:"error,omitempty"`
	Metadata        map[string]any            `json:"metadata,omitempty"`
}

type remoteToolExecutionError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable,omitempty"`
}

type accountOverviewToolArgs struct {
	UserID          string `json:"user_id"`
	ConsentGrantRef string `json:"consent_grant_ref"`
}

type fundMarketSnapshotToolArgs struct {
	InstrumentCode string `json:"instrument_code"`
}

// handleRemoteToolCatalog exposes fund-owned tool registrations for Athena setup.
// handleRemoteToolCatalog 暴露基金应用自有工具注册信息，供 Athena 配置使用。
func (s *Server) handleRemoteToolCatalog(w http.ResponseWriter, r *http.Request) {
	baseURL := strings.TrimRight(r.URL.Query().Get("base_url"), "/")
	endpoint := "/internal/tools/execute"
	if baseURL != "" {
		endpoint = baseURL + endpoint
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"contract_version": remoteToolContractVersion,
		"app_id":           "athena-fund-assistant",
		"items": []remoteToolRegistration{
			{
				RegistrationID: "fund_account_overview_v1",
				AppID:          "athena-fund-assistant",
				Name:           "account_overview",
				Description:    "Read a user's fund assistant account overview, holdings, recent operations, and performance trend.",
				Parameters: objectSchema(map[string]any{
					"user_id":           stringSchema("Fund assistant user ID from the authenticated conversation context."),
					"consent_grant_ref": stringSchema("Opaque read-only consent grant reference from the conversation context."),
				}, []string{"user_id", "consent_grant_ref"}),
				Endpoint:         endpoint,
				ToolScope:        "fund.account.summary.read",
				Operation:        "read_account_overview",
				RiskLevel:        "low",
				SideEffectLevel:  "none",
				Idempotent:       true,
				TimeoutMS:        5000,
				RetryMaxAttempts: 1,
				Enabled:          true,
				Metadata: map[string]any{
					"data_boundary":   "manual_account_and_demo_snapshots",
					"no_trading":      true,
					"required_scopes": []string{string(authorization.ScopeAccountSummaryRead), string(authorization.ScopeHoldingSnapshotRead)},
				},
			},
			{
				RegistrationID:   "fund_market_snapshot_v1",
				AppID:            "athena-fund-assistant",
				Name:             "fund_market_snapshot",
				Description:      "Read a normalized fund or ETF snapshot with source, freshness, timezone, delay, license, and confidence metadata.",
				Parameters:       objectSchema(map[string]any{"instrument_code": stringSchema("Fund, ETF, or mock-provider-supported instrument code.")}, []string{"instrument_code"}),
				Endpoint:         endpoint,
				ToolScope:        "fund.market.read",
				Operation:        "read_fund_market_snapshot",
				RiskLevel:        "low",
				SideEffectLevel:  "none",
				Idempotent:       true,
				TimeoutMS:        5000,
				RetryMaxAttempts: 1,
				Enabled:          true,
				Metadata: map[string]any{
					"provider_contract": "internal/data.Provider",
					"no_trading":        true,
				},
			},
		},
	})
}

// handleRemoteToolExecution executes Athena remote_tool_execution.v1 callbacks.
// handleRemoteToolExecution 执行 Athena remote_tool_execution.v1 回调。
func (s *Server) handleRemoteToolExecution(w http.ResponseWriter, r *http.Request) {
	var req remoteToolExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRemoteToolError(w, http.StatusBadRequest, req, "invalid_request", err.Error(), false)
		return
	}
	if strings.TrimSpace(req.ContractVersion) != remoteToolContractVersion {
		writeRemoteToolError(w, http.StatusBadRequest, req, "contract_version_mismatch", "remote tool request contract_version mismatch", false)
		return
	}
	if !s.requireRemoteService(w, r, req) {
		return
	}
	switch strings.TrimSpace(req.ToolName) {
	case "account_overview":
		s.executeAccountOverviewTool(w, r, req)
	case "fund_market_snapshot":
		s.executeFundMarketSnapshotTool(w, r, req)
	default:
		writeRemoteToolError(w, http.StatusBadRequest, req, "unknown_tool", "fund assistant does not expose the requested tool", false)
	}
}

func (s *Server) executeAccountOverviewTool(w http.ResponseWriter, r *http.Request, req remoteToolExecutionRequest) {
	if s.deps.Accounts == nil {
		writeRemoteToolError(w, http.StatusServiceUnavailable, req, "account_store_unconfigured", "account store is not configured", true)
		return
	}
	var args accountOverviewToolArgs
	if err := decodeRemoteToolArguments(req.Arguments, &args); err != nil {
		writeRemoteToolError(w, http.StatusBadRequest, req, "invalid_arguments", err.Error(), false)
		return
	}
	userID := strings.TrimSpace(args.UserID)
	grantRef := strings.TrimSpace(args.ConsentGrantRef)
	if s.deps.Authorization == nil && userID == "" {
		userID = "demo-user"
	}
	if userID == "" {
		writeRemoteToolError(w, http.StatusBadRequest, req, "user_id_required", "user_id is required", false)
		return
	}
	if s.deps.Authorization != nil && grantRef == "" {
		writeRemoteToolAuthorizationError(w, req, authorization.AuthorizationDecision{
			Code:  authorization.DenialMissingGrant,
			Scope: authorization.ScopeAccountSummaryRead,
		})
		return
	}
	decisions := make([]authorization.AuthorizationDecision, 0, 2)
	for _, scope := range []authorization.Scope{authorization.ScopeAccountSummaryRead, authorization.ScopeHoldingSnapshotRead} {
		if s.deps.Authorization == nil {
			break
		}
		decision, err := s.deps.Authorization.AuthorizeGrant(r.Context(), authorization.GrantAuthorizationRequest{
			GrantRef: grantRef,
			Subject:  userID,
			Audience: athenaAudience,
			Scope:    scope,
		})
		if err != nil {
			code, ok := authorizationDenied(err)
			if !ok {
				writeRemoteToolError(w, http.StatusInternalServerError, req, "authorization_check_failed", "account authorization check failed", true)
				return
			}
			decision.Code = code
			decision.Scope = scope
			decision.GrantRef = grantRef
		}
		if !decision.Allowed {
			writeRemoteToolAuthorizationError(w, req, decision)
			return
		}
		decisions = append(decisions, decision)
	}
	overview, err := s.deps.Accounts.Overview(r.Context(), userID)
	if err != nil {
		writeRemoteToolError(w, http.StatusNotFound, req, "account_not_found", err.Error(), false)
		return
	}
	writeRemoteToolContent(w, req, map[string]any{
		"tool":     "account_overview",
		"overview": overview,
		"safety": map[string]any{
			"no_auto_trading":   true,
			"read_only":         true,
			"consent_grant_ref": grantRef,
			"consent_revision":  highestAuthorizationRevision(decisions),
		},
	})
}

func (s *Server) executeFundMarketSnapshotTool(w http.ResponseWriter, r *http.Request, req remoteToolExecutionRequest) {
	if s.deps.Provider == nil {
		writeRemoteToolError(w, http.StatusServiceUnavailable, req, "provider_unconfigured", "data provider is not configured", true)
		return
	}
	var args fundMarketSnapshotToolArgs
	if err := decodeRemoteToolArguments(req.Arguments, &args); err != nil {
		writeRemoteToolError(w, http.StatusBadRequest, req, "invalid_arguments", err.Error(), false)
		return
	}
	if strings.TrimSpace(args.InstrumentCode) == "" {
		writeRemoteToolError(w, http.StatusBadRequest, req, "instrument_code_required", "instrument_code is required", false)
		return
	}
	snapshot, err := s.deps.Provider.GetFundSnapshot(r.Context(), args.InstrumentCode)
	if err != nil {
		writeRemoteToolError(w, http.StatusNotFound, req, "instrument_not_found", err.Error(), false)
		return
	}
	writeRemoteToolContent(w, req, map[string]any{
		"tool":          "fund_market_snapshot",
		"fund_snapshot": snapshot,
		"safety": map[string]any{
			"no_auto_trading": true,
			"read_only":       true,
		},
	})
}

func writeRemoteToolContent(w http.ResponseWriter, req remoteToolExecutionRequest, content any) {
	payload, err := json.Marshal(content)
	if err != nil {
		writeRemoteToolError(w, http.StatusInternalServerError, req, "content_encode_failed", err.Error(), true)
		return
	}
	writeJSON(w, http.StatusOK, remoteToolExecutionResponse{
		ContractVersion: remoteToolContractVersion,
		RequestID:       strings.TrimSpace(req.RequestID),
		ToolCallID:      strings.TrimSpace(req.ToolCallID),
		Status:          "ok",
		Content:         string(payload),
		Metadata: map[string]any{
			"app_id": "athena-fund-assistant",
		},
	})
}

func writeRemoteToolError(w http.ResponseWriter, status int, req remoteToolExecutionRequest, code, message string, retryable bool) {
	writeJSON(w, status, remoteToolExecutionResponse{
		ContractVersion: remoteToolContractVersion,
		RequestID:       strings.TrimSpace(req.RequestID),
		ToolCallID:      strings.TrimSpace(req.ToolCallID),
		Status:          "error",
		Error: &remoteToolExecutionError{
			Code:      code,
			Message:   message,
			Retryable: retryable,
		},
		Metadata: map[string]any{
			"app_id": "athena-fund-assistant",
		},
	})
}

func writeRemoteToolAuthorizationError(w http.ResponseWriter, req remoteToolExecutionRequest, decision authorization.AuthorizationDecision) {
	code := decision.Code
	if code == "" {
		code = authorization.DenialMissingGrant
	}
	writeJSON(w, http.StatusForbidden, remoteToolExecutionResponse{
		ContractVersion: remoteToolContractVersion,
		RequestID:       strings.TrimSpace(req.RequestID),
		ToolCallID:      strings.TrimSpace(req.ToolCallID),
		Status:          "error",
		Error: &remoteToolExecutionError{
			Code:    "authorization_denied",
			Message: "read-only account consent denied",
		},
		Metadata: map[string]any{
			"app_id":             "athena-fund-assistant",
			"authorization_code": code,
			"consent_grant_ref":  decision.GrantRef,
			"consent_revision":   decision.Revision,
			"required_scope":     decision.Scope,
		},
	})
}

func highestAuthorizationRevision(decisions []authorization.AuthorizationDecision) int64 {
	var revision int64
	for _, decision := range decisions {
		if decision.Revision > revision {
			revision = decision.Revision
		}
	}
	return revision
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
		"required":             required,
	}
}

func stringSchema(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
	}
}

func decodeRemoteToolArguments(arguments json.RawMessage, target any) error {
	if len(arguments) == 0 {
		arguments = json.RawMessage(`{}`)
	}
	return json.Unmarshal(arguments, target)
}
