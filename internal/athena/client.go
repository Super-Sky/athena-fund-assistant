package athena

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client defines the fund app boundary for starting Athena agent runs.
// Client 定义基金应用发起 Athena agent run 的边界。
type Client interface {
	StartRun(context.Context, StartRunRequest) (RunResponse, error)
}

// StartRunRequest is the generic app-facing Athena run request used by this app.
// StartRunRequest 是本应用使用的通用 Athena run 请求。
type StartRunRequest struct {
	Goal         string            `json:"goal,omitempty"`
	Success      []string          `json:"success_criteria,omitempty"`
	Constraints  map[string]any    `json:"constraints,omitempty"`
	Budget       map[string]any    `json:"budget,omitempty"`
	Context      []ContextAsset    `json:"context_assets,omitempty"`
	Tools        []ToolDeclaration `json:"tools,omitempty"`
	MemoryScope  map[string]any    `json:"memory_scope,omitempty"`
	Governance   []string          `json:"governance_refs,omitempty"`
	SessionID    string            `json:"session_id,omitempty"`
	AppID        string            `json:"app_instance_id,omitempty"`
	AppSessionID string            `json:"app_session_id,omitempty"`
	InputPayload map[string]any    `json:"input_payload,omitempty"`
	EnabledTools []string          `json:"enabled_tools,omitempty"`
}

// ContextAsset carries business context as generic Athena input assets.
// ContextAsset 将业务上下文作为 Athena 通用输入资产传递。
type ContextAsset struct {
	AssetID   string         `json:"asset_id"`
	AssetType string         `json:"asset_type"`
	Content   map[string]any `json:"content,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// ToolDeclaration is an OpenAI-compatible function tool declaration.
// ToolDeclaration 是 OpenAI-compatible function tool 声明。
type ToolDeclaration struct {
	Type     string         `json:"type"`
	Function ToolFunction   `json:"function"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ToolFunction describes one function tool for Athena tool resolution.
// ToolFunction 描述一个用于 Athena tool resolution 的 function tool。
type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// RunResponse is the stable subset of Athena Agent Run response consumed by this app.
// RunResponse 是本应用消费的 Athena Agent Run 响应稳定子集。
type RunResponse struct {
	RequestID      string           `json:"request_id,omitempty"`
	RunID          string           `json:"run_id,omitempty"`
	SessionID      string           `json:"session_id,omitempty"`
	Status         string           `json:"status"`
	StopReason     string           `json:"stop_reason,omitempty"`
	Output         string           `json:"output,omitempty"`
	ToolCalls      []map[string]any `json:"tool_calls,omitempty"`
	TraceAvailable bool             `json:"trace_available"`
	Metadata       map[string]any   `json:"metadata,omitempty"`
}

// HTTPClient calls Athena through its app-facing Agent Run API.
// HTTPClient 通过 Athena 面向应用的 Agent Run API 发起调用。
type HTTPClient struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

// NewHTTPClient creates a real Athena client for configured deployments.
// NewHTTPClient 为已配置部署创建真实 Athena client。
func NewHTTPClient(baseURL, token string) (*HTTPClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("athena base url is invalid")
	}
	return &HTTPClient{
		BaseURL: strings.TrimRight(parsed.String(), "/"),
		Token:   strings.TrimSpace(token),
		Client:  &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// StartRun posts one goal-driven run to Athena.
// StartRun 向 Athena 提交一次目标驱动 run。
func (c *HTTPClient) StartRun(ctx context.Context, req StartRunRequest) (RunResponse, error) {
	if c == nil {
		return RunResponse{}, fmt.Errorf("athena client is nil")
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return RunResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/agent/runs", bytes.NewReader(payload))
	if err != nil {
		return RunResponse{}, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return RunResponse{}, err
	}
	defer resp.Body.Close()
	var result RunResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return RunResponse{}, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if result.Status == "" {
			result.Status = "error"
		}
		return result, fmt.Errorf("athena agent run returned HTTP %d", resp.StatusCode)
	}
	return result, nil
}

// MockClient keeps local development runnable without a live Athena service.
// MockClient 让本地开发在没有 Athena 服务时仍可运行。
type MockClient struct{}

// StartRun returns a deterministic local run marker.
// StartRun 返回确定性的本地 run 标记。
func (MockClient) StartRun(_ context.Context, req StartRunRequest) (RunResponse, error) {
	now := time.Now().UTC()
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = "mock_athena_session"
	}
	return RunResponse{
		RequestID:      "mock_athena_request",
		RunID:          "mock_athena_run_" + strings.ReplaceAll(now.Format("20060102150405.000000000"), ".", ""),
		SessionID:      sessionID,
		Status:         "completed",
		StopReason:     "mock_local_development",
		Output:         "Mock Athena run accepted the fund assistant goal.",
		TraceAvailable: true,
		Metadata: map[string]any{
			"mock":        true,
			"app_id":      strings.TrimSpace(req.AppID),
			"tool_count":  len(req.Tools),
			"asset_count": len(req.Context),
		},
	}, nil
}
