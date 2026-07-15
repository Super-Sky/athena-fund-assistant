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
			"run_id":                result.RunID,
			"run_status":            status,
			"trace_available":       fmt.Sprintf("%t", result.TraceAvailable),
			"stop_reason":           result.StopReason,
			"tool_call_count":       fmt.Sprintf("%d", len(result.ToolCalls)),
			"output_present":        fmt.Sprintf("%t", strings.TrimSpace(result.Output) != ""),
			"consent_contract":      "read_only_grant_ref_v1",
			"consent_grant_ref":     strings.TrimSpace(req.ConsentGrantRef),
			"authorization_subject": strings.TrimSpace(req.UserID),
		},
	})
}

func buildAthenaRunRequest(conversationID string, req addConversationMessageRequest) athena.StartRunRequest {
	skillID := strings.TrimSpace(req.SkillID)
	if skillID == "" {
		skillID = "fund_research"
	}
	attachmentIDs := compactStrings(req.AttachmentIDs)
	userID := strings.TrimSpace(req.UserID)
	consentGrantRef := strings.TrimSpace(req.ConsentGrantRef)
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
					"conversation_id":   conversationID,
					"skill_id":          skillID,
					"attachment_ids":    attachmentIDs,
					"user_id":           userID,
					"consent_grant_ref": consentGrantRef,
				},
				Metadata: map[string]any{
					"attachment_content_policy": "metadata_only_until_parser_confirms",
					"consent_contract":          "read_only_grant_ref_v1",
				},
			},
		},
		Tools:        fundAthenaToolDeclarations(),
		EnabledTools: fundAthenaToolNames(),
		MemoryScope: map[string]any{
			"user_id": userID,
			"scope":   "fund_assistant_mvp",
		},
		Governance: []string{
			"fund_assistant_no_auto_trading",
			"fund_assistant_no_guaranteed_returns",
			"fund_assistant_source_metadata_required",
		},
		AppID:        "athena-fund-assistant",
		AppSessionID: conversationID,
		InputPayload: map[string]any{
			"role":              strings.TrimSpace(req.Role),
			"skill_id":          skillID,
			"attachment_ids":    attachmentIDs,
			"message":           strings.TrimSpace(req.Content),
			"user_id":           userID,
			"consent_grant_ref": consentGrantRef,
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
					"user_id":           stringSchema("Fund assistant user ID from the authenticated conversation context."),
					"consent_grant_ref": stringSchema("Opaque read-only consent grant reference from the conversation context."),
				}, []string{"user_id", "consent_grant_ref"}),
			},
			Metadata: map[string]any{
				"side_effect_level": "none",
				"tool_scope":        "fund.account.summary.read",
				"required_scopes": []string{
					"fund.account.summary.read",
					"fund.holding.snapshot.read",
				},
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
