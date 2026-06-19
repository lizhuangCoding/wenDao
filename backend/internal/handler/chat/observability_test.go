package chat

import (
	"strings"
	"testing"
	"time"

	"wenDao/internal/model"
)

func TestBuildAIObservabilityRunResponseSummarizesRun(t *testing.T) {
	start := time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)
	completed := start.Add(95 * time.Second)
	lastError := "web fetch timeout"

	run := model.ConversationRun{
		ID:                 11,
		ConversationID:     22,
		UserID:             33,
		Status:             "failed",
		CurrentStage:       "research",
		OriginalQuestion:   "解释 RAG 命中来源",
		LastError:          &lastError,
		PromptTokens:       120,
		CompletionTokens:   80,
		EstimatedCost:      0.03,
		CostCurrency:       "USD",
		CostStatus:         "estimated",
		SourceQualityScore: 72,
		FailureCategory:    "timeout",
		FailureFingerprint: "abc123",
		CreatedAt:          start,
		UpdatedAt:          start.Add(10 * time.Second),
		CompletedAt:        &completed,
	}
	steps := []model.ConversationRunStep{
		{
			ID:        101,
			RunID:     run.ID,
			AgentName: "ThinkTank",
			Type:      "tool_use",
			Summary:   "LocalSearch matched knowledge documents",
			Status:    "completed",
			CreatedAt: start.Add(time.Second),
		},
		{
			ID:        102,
			RunID:     run.ID,
			AgentName: "Researcher",
			Type:      "tool_use",
			Summary:   "WebSearch found https://example.com/rag.",
			Status:    "completed",
			CreatedAt: start.Add(2 * time.Second),
		},
		{
			ID:        103,
			RunID:     run.ID,
			AgentName: "Fetcher",
			Type:      "tool_use",
			Summary:   "WebFetch failed",
			Detail:    strings.Repeat("x", 900),
			Status:    "failed",
			CreatedAt: start.Add(3 * time.Second),
		},
	}

	resp := buildAIObservabilityRunResponse(run, steps)

	if resp.DurationSeconds != 95 {
		t.Fatalf("expected duration 95s, got %d", resp.DurationSeconds)
	}
	if resp.StepCount != 3 || resp.FailedStepCount != 1 {
		t.Fatalf("unexpected step counts: total=%d failed=%d", resp.StepCount, resp.FailedStepCount)
	}
	if resp.ToolUsage.LocalSearch != 1 || resp.ToolUsage.WebSearch != 1 || resp.ToolUsage.WebFetch != 1 {
		t.Fatalf("unexpected tool usage: %#v", resp.ToolUsage)
	}
	if resp.Sources.LocalHits != 1 || resp.Sources.WebHits != 2 {
		t.Fatalf("unexpected source counts: %#v", resp.Sources)
	}
	if len(resp.Sources.ExternalURLs) != 1 || resp.Sources.ExternalURLs[0].URL != "https://example.com/rag" {
		t.Fatalf("unexpected external URLs: %#v", resp.Sources.ExternalURLs)
	}
	if resp.Sources.QualityScore != 72 {
		t.Fatalf("expected persisted source quality, got %d", resp.Sources.QualityScore)
	}
	if len(resp.FailedSteps) != 1 || len([]rune(resp.FailedSteps[0].Detail)) != 803 || resp.FailedSteps[0].Category != "tool" {
		t.Fatalf("expected one truncated failed step, got %#v", resp.FailedSteps)
	}
	if resp.Cost.Status != "estimated" || resp.Cost.PromptTokens != 120 || resp.Cost.CompletionTokens != 80 || resp.Cost.EstimatedCost != 0.03 {
		t.Fatalf("expected cost and feedback placeholders, got cost=%#v feedback=%#v", resp.Cost, resp.Feedback)
	}
	if resp.FailureCategory != "timeout" || resp.FailureFingerprint != "abc123" || len(resp.FailureClusters) == 0 {
		t.Fatalf("expected failure classification in response, got %#v", resp)
	}
}
