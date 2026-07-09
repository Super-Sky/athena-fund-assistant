package athena

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPClientStartRun(t *testing.T) {
	var captured StartRunRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/agent/runs" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(RunResponse{
			RequestID:      "req_1",
			RunID:          "run_1",
			SessionID:      "session_1",
			Status:         "completed",
			TraceAvailable: true,
		})
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL, "test-token")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	response, err := client.StartRun(context.Background(), StartRunRequest{
		Goal:         "Review fund account",
		SessionID:    "session_1",
		AppID:        "athena-fund-assistant",
		EnabledTools: []string{"account_overview"},
		Context:      []ContextAsset{{AssetID: "conversation.conv_1", AssetType: "conversation"}},
		Tools:        []ToolDeclaration{{Type: "function", Function: ToolFunction{Name: "account_overview"}}},
		Governance:   []string{"fund_assistant_no_auto_trading"},
		InputPayload: map[string]any{"skill_id": "portfolio_review"},
		Success:      []string{"has evidence"},
		Constraints:  map[string]any{"no_auto_trading": true},
		MemoryScope:  map[string]any{"user_id": "demo-user"},
		AppSessionID: "conv_1",
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if response.RunID != "run_1" || !response.TraceAvailable {
		t.Fatalf("response = %#v", response)
	}
	if captured.Goal != "Review fund account" || captured.Tools[0].Function.Name != "account_overview" {
		t.Fatalf("captured request = %#v", captured)
	}
	if captured.Constraints["no_auto_trading"] != true {
		t.Fatalf("constraints = %#v", captured.Constraints)
	}
}

func TestMockClientStartRun(t *testing.T) {
	response, err := (MockClient{}).StartRun(context.Background(), StartRunRequest{
		SessionID: "conv_1",
		AppID:     "athena-fund-assistant",
	})
	if err != nil {
		t.Fatalf("mock start run: %v", err)
	}
	if response.RunID == "" || response.SessionID != "conv_1" || response.Status != "completed" || !response.TraceAvailable {
		t.Fatalf("response = %#v", response)
	}
}
