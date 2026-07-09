package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/Super-Sky/athena-fund-assistant/internal/athena"
	"github.com/Super-Sky/athena-fund-assistant/internal/conversation"
	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

// startAthenaRunForMessage maps a fund conversation message into a generic Athena Agent Run.
// startAthenaRunForMessage 将基金对话消息映射为通用 Athena Agent Run。
func (s *Server) startAthenaRunForMessage(ctx context.Context, conversationID string, req addConversationMessageRequest) (domain.ConversationDetail, error) {
	runReq := buildAthenaRunRequest(conversationID, req)
	result, err := s.deps.Athena.StartRun(ctx, runReq)
	if err != nil {
		return domain.ConversationDetail{}, err
	}
	status := strings.TrimSpace(result.Status)
	if status == "" {
		status = "unknown"
	}
	return s.deps.Conversations.RecordTrace(ctx, conversationID, conversation.TraceInput{
		Kind:    "athena_agent_run",
		Status:  "ok",
		Summary: "Athena agent run accepted the workspace message.",
		Metadata: map[string]string{
			"run_id":          result.RunID,
			"run_status":      status,
			"trace_available": fmt.Sprintf("%t", result.TraceAvailable),
			"stop_reason":     result.StopReason,
		},
	})
}

func buildAthenaRunRequest(conversationID string, req addConversationMessageRequest) athena.StartRunRequest {
	skillID := strings.TrimSpace(req.SkillID)
	if skillID == "" {
		skillID = "fund_research"
	}
	attachmentIDs := compactStrings(req.AttachmentIDs)
	return athena.StartRunRequest{
		Goal: strings.TrimSpace(req.Content),
		Success: []string{
			"preserve data source and freshness metadata",
			"avoid automatic trading or guaranteed-return claims",
			"produce conservative, balanced, and aggressive options when a decision is requested",
		},
		Constraints: map[string]any{
			"no_auto_trading":            true,
			"no_brokerage_order_access":  true,
			"must_include_risk":          true,
			"must_include_invalidation":  true,
			"attachment_metadata_only":   true,
			"no_single_absolute_outcome": true,
		},
		Budget: map[string]any{
			"timeout_after_seconds": 30,
		},
		Context: []athena.ContextAsset{
			{
				AssetID:   "conversation." + conversationID,
				AssetType: "fund_assistant_conversation",
				Content: map[string]any{
					"conversation_id": conversationID,
					"skill_id":        skillID,
					"attachment_ids":  attachmentIDs,
				},
				Metadata: map[string]any{
					"attachment_content_policy": "metadata_only_until_parser_confirms",
				},
			},
		},
		Tools:        fundAthenaToolDeclarations(),
		EnabledTools: fundAthenaToolNames(),
		MemoryScope: map[string]any{
			"user_id": "demo-user",
			"scope":   "fund_assistant_mvp",
		},
		Governance: []string{
			"fund_assistant_no_auto_trading",
			"fund_assistant_no_guaranteed_returns",
			"fund_assistant_source_metadata_required",
		},
		SessionID:    conversationID,
		AppID:        "athena-fund-assistant",
		AppSessionID: conversationID,
		InputPayload: map[string]any{
			"role":           strings.TrimSpace(req.Role),
			"skill_id":       skillID,
			"attachment_ids": attachmentIDs,
			"message":        strings.TrimSpace(req.Content),
		},
	}
}

func fundAthenaToolNames() []string {
	return []string{"account_overview", "fund_market_snapshot"}
}

func fundAthenaToolDeclarations() []athena.ToolDeclaration {
	return []athena.ToolDeclaration{
		{
			Type: "function",
			Function: athena.ToolFunction{
				Name:        "account_overview",
				Description: "Read the fund assistant account overview as read-only context.",
				Parameters: objectSchema(map[string]any{
					"user_id": stringSchema("Fund assistant user ID. Defaults to demo-user when omitted."),
				}, []string{}),
			},
			Metadata: map[string]any{
				"side_effect_level": "none",
				"tool_scope":        "fund.account.read",
			},
		},
		{
			Type: "function",
			Function: athena.ToolFunction{
				Name:        "fund_market_snapshot",
				Description: "Read a fund or ETF snapshot with source and freshness metadata.",
				Parameters: objectSchema(map[string]any{
					"instrument_code": stringSchema("Fund, ETF, or mock-provider-supported instrument code."),
				}, []string{"instrument_code"}),
			},
			Metadata: map[string]any{
				"side_effect_level": "none",
				"tool_scope":        "fund.market.read",
			},
		},
	}
}
