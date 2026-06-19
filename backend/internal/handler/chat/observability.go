package chat

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"wenDao/internal/model"
	"wenDao/internal/pkg/aiobserve"
	"wenDao/internal/pkg/pagination"
	"wenDao/internal/pkg/response"
	"wenDao/internal/repository"
)

type AIObservabilityHandler struct {
	runRepo     repository.ConversationRunRepository
	runStepRepo repository.ConversationRunStepRepository
}

func NewAIObservabilityHandler(
	runRepo repository.ConversationRunRepository,
	runStepRepo repository.ConversationRunStepRepository,
) *AIObservabilityHandler {
	return &AIObservabilityHandler{runRepo: runRepo, runStepRepo: runStepRepo}
}

type AIObservabilityRunResponse struct {
	ID                 int64                            `json:"id"`
	ConversationID     int64                            `json:"conversation_id"`
	UserID             int64                            `json:"user_id"`
	Status             string                           `json:"status"`
	CurrentStage       string                           `json:"current_stage"`
	OriginalQuestion   string                           `json:"original_question"`
	NormalizedQuestion string                           `json:"normalized_question"`
	LastError          string                           `json:"last_error,omitempty"`
	DurationSeconds    int64                            `json:"duration_seconds"`
	StepCount          int                              `json:"step_count"`
	FailedStepCount    int                              `json:"failed_step_count"`
	ToolUsage          AIObservabilityToolUsageResponse `json:"tool_usage"`
	Sources            AIObservabilitySourceSummary     `json:"sources"`
	Cost               AIObservabilityCostEstimate      `json:"cost"`
	Feedback           AIObservabilityFeedbackSummary   `json:"feedback"`
	FailureCategory    string                           `json:"failure_category,omitempty"`
	FailureFingerprint string                           `json:"failure_fingerprint,omitempty"`
	FailedSteps        []AIObservabilityFailedStep      `json:"failed_steps"`
	FailureClusters    []AIObservabilityFailureCluster  `json:"failure_clusters"`
	Steps              []AIObservabilityStepResponse    `json:"steps"`
	CreatedAt          string                           `json:"created_at"`
	UpdatedAt          string                           `json:"updated_at"`
	CompletedAt        string                           `json:"completed_at,omitempty"`
	HeartbeatAt        string                           `json:"heartbeat_at,omitempty"`
}

type AIObservabilityToolUsageResponse struct {
	LocalSearch int `json:"local_search"`
	WebSearch   int `json:"web_search"`
	WebFetch    int `json:"web_fetch"`
	DocWriter   int `json:"doc_writer"`
	Other       int `json:"other"`
}

type AIObservabilitySourceSummary struct {
	LocalHits    int                             `json:"local_hits"`
	WebHits      int                             `json:"web_hits"`
	QualityScore int                             `json:"quality_score"`
	ExternalURLs []AIObservabilityExternalSource `json:"external_urls"`
}

type AIObservabilityExternalSource struct {
	URL          string `json:"url"`
	QualityScore int    `json:"quality_score"`
}

type AIObservabilityCostEstimate struct {
	Status           string  `json:"status"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	EstimatedCost    float64 `json:"estimated_cost"`
	Currency         string  `json:"currency"`
}

type AIObservabilityFeedbackSummary struct {
	Status string `json:"status"`
	Score  *int   `json:"score,omitempty"`
}

type AIObservabilityFailedStep struct {
	ID        int64  `json:"id"`
	AgentName string `json:"agent_name"`
	Type      string `json:"type"`
	Summary   string `json:"summary"`
	Detail    string `json:"detail"`
	Category  string `json:"category"`
	CreatedAt string `json:"created_at"`
}

type AIObservabilityFailureCluster struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

type AIObservabilityStepResponse struct {
	ID        int64  `json:"id"`
	AgentName string `json:"agent_name"`
	Type      string `json:"type"`
	Summary   string `json:"summary"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type AIObservabilityBatchDeleteRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1"`
}

func (h *AIObservabilityHandler) ListRuns(c *gin.Context) {
	if h.runRepo == nil || h.runStepRepo == nil {
		response.InternalError(c, "AI observability is not configured")
		return
	}

	p := pagination.FromQuery(c)
	runs, total, err := h.runRepo.ListRecent(repository.ConversationRunFilter{
		Status:   strings.TrimSpace(c.Query("status")),
		Keyword:  strings.TrimSpace(c.Query("keyword")),
		Page:     p.Page,
		PageSize: p.PageSize,
	})
	if err != nil {
		response.InternalErrorWithErr(c, "Failed to get AI run records", err)
		return
	}

	runIDs := make([]int64, 0, len(runs))
	for _, run := range runs {
		runIDs = append(runIDs, run.ID)
	}
	steps, err := h.runStepRepo.GetByRunIDs(runIDs)
	if err != nil {
		response.InternalErrorWithErr(c, "Failed to get AI run steps", err)
		return
	}

	stepsByRunID := make(map[int64][]model.ConversationRunStep)
	for _, step := range steps {
		stepsByRunID[step.RunID] = append(stepsByRunID[step.RunID], step)
	}

	items := make([]AIObservabilityRunResponse, 0, len(runs))
	for _, run := range runs {
		items = append(items, buildAIObservabilityRunResponse(run, stepsByRunID[run.ID]))
	}

	response.Success(c, gin.H{
		"data":       items,
		"total":      total,
		"page":       p.Page,
		"pageSize":   p.PageSize,
		"totalPages": pagination.TotalPages(total, p.PageSize),
	})
}

func (h *AIObservabilityHandler) BatchDeleteRuns(c *gin.Context) {
	if h.runRepo == nil || h.runStepRepo == nil {
		response.InternalError(c, "AI observability is not configured")
		return
	}

	var req AIObservabilityBatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "请选择要删除的 AI 运行记录")
		return
	}
	ids, ok := normalizeAIObservabilityRunIDs(req.IDs)
	if !ok {
		response.InvalidParams(c, "AI 运行记录 ID 无效")
		return
	}

	if err := h.runStepRepo.DeleteByRunIDs(ids); err != nil {
		response.InternalErrorWithErr(c, "Failed to delete AI run steps", err)
		return
	}
	if err := h.runRepo.DeleteBatch(ids); err != nil {
		response.InternalErrorWithErr(c, "Failed to delete AI run records", err)
		return
	}

	response.Success(c, gin.H{"message": "AI run records deleted successfully", "deleted_count": len(ids)})
}

func buildAIObservabilityRunResponse(run model.ConversationRun, steps []model.ConversationRunStep) AIObservabilityRunResponse {
	toolUsage := AIObservabilityToolUsageResponse{}
	sources := AIObservabilitySourceSummary{}
	failedSteps := make([]AIObservabilityFailedStep, 0)
	failureClusters := make(map[string]int)
	stepResponses := make([]AIObservabilityStepResponse, 0, len(steps))
	urls := make(map[string]struct{})

	for _, step := range steps {
		toolKind := classifyToolStep(step)
		switch toolKind {
		case "local_search":
			toolUsage.LocalSearch++
			sources.LocalHits++
		case "web_search":
			toolUsage.WebSearch++
			sources.WebHits++
		case "web_fetch":
			toolUsage.WebFetch++
			sources.WebHits++
		case "doc_writer":
			toolUsage.DocWriter++
		case "other":
			if step.Type == "tool_use" || strings.Contains(strings.ToLower(step.Summary+" "+step.Detail), "tool") {
				toolUsage.Other++
			}
		}
		for _, url := range extractHTTPURLs(step.Summary + " " + step.Detail) {
			if _, exists := urls[url]; !exists {
				urls[url] = struct{}{}
				sources.ExternalURLs = append(sources.ExternalURLs, AIObservabilityExternalSource{
					URL:          url,
					QualityScore: aiobserve.ScoreSourceQuality([]string{url}, 0, 1),
				})
			}
		}
		if step.Status == "failed" {
			category := aiobserve.ClassifyFailure(step.Summary + " " + step.Detail)
			if category == "" {
				category = "unknown"
			}
			failureClusters[category]++
			failedSteps = append(failedSteps, AIObservabilityFailedStep{
				ID:        step.ID,
				AgentName: step.AgentName,
				Type:      step.Type,
				Summary:   step.Summary,
				Detail:    truncateForObservability(step.Detail, 800),
				Category:  category,
				CreatedAt: formatObservedTime(step.CreatedAt),
			})
		}
		stepResponses = append(stepResponses, AIObservabilityStepResponse{
			ID:        step.ID,
			AgentName: step.AgentName,
			Type:      step.Type,
			Summary:   step.Summary,
			Status:    step.Status,
			CreatedAt: formatObservedTime(step.CreatedAt),
		})
	}
	if sources.QualityScore == 0 {
		sourceURLs := make([]string, 0, len(sources.ExternalURLs))
		for _, source := range sources.ExternalURLs {
			sourceURLs = append(sourceURLs, source.URL)
		}
		sources.QualityScore = aiobserve.ScoreSourceQuality(sourceURLs, sources.LocalHits, sources.WebHits)
	}
	if run.SourceQualityScore > 0 {
		sources.QualityScore = run.SourceQualityScore
	}
	clusters := make([]AIObservabilityFailureCluster, 0, len(failureClusters))
	if run.FailureCategory != "" {
		failureClusters[run.FailureCategory]++
	}
	for category, count := range failureClusters {
		clusters = append(clusters, AIObservabilityFailureCluster{Category: category, Count: count})
	}
	if run.FailureCategory == "" && derefString(run.LastError) != "" {
		run.FailureCategory = aiobserve.ClassifyFailure(derefString(run.LastError))
		run.FailureFingerprint = aiobserve.FailureFingerprint(run.FailureCategory, derefString(run.LastError))
	}

	return AIObservabilityRunResponse{
		ID:                 run.ID,
		ConversationID:     run.ConversationID,
		UserID:             run.UserID,
		Status:             run.Status,
		CurrentStage:       run.CurrentStage,
		OriginalQuestion:   run.OriginalQuestion,
		NormalizedQuestion: run.NormalizedQuestion,
		LastError:          derefString(run.LastError),
		DurationSeconds:    observedRunDurationSeconds(run),
		StepCount:          len(steps),
		FailedStepCount:    len(failedSteps),
		ToolUsage:          toolUsage,
		Sources:            sources,
		Cost: AIObservabilityCostEstimate{
			Status:           observedCostStatus(run),
			PromptTokens:     run.PromptTokens,
			CompletionTokens: run.CompletionTokens,
			EstimatedCost:    run.EstimatedCost,
			Currency:         observedCostCurrency(run),
		},
		Feedback:           AIObservabilityFeedbackSummary{Status: "not_collected"},
		FailureCategory:    run.FailureCategory,
		FailureFingerprint: run.FailureFingerprint,
		FailedSteps:        failedSteps,
		FailureClusters:    clusters,
		Steps:              stepResponses,
		CreatedAt:          formatObservedTime(run.CreatedAt),
		UpdatedAt:          formatObservedTime(run.UpdatedAt),
		CompletedAt:        formatObservedTimePtr(run.CompletedAt),
		HeartbeatAt:        formatObservedTimePtr(run.HeartbeatAt),
	}
}

func normalizeAIObservabilityRunIDs(ids []int64) ([]int64, bool) {
	seen := make(map[int64]struct{}, len(ids))
	normalized := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, false
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized, len(normalized) > 0
}

func classifyToolStep(step model.ConversationRunStep) string {
	text := strings.ToLower(step.AgentName + " " + step.Type + " " + step.Summary + " " + step.Detail)
	switch {
	case strings.Contains(text, "localsearch") || strings.Contains(text, "local search") || strings.Contains(text, "redis 知识库"):
		return "local_search"
	case strings.Contains(text, "websearch") || strings.Contains(text, "web search") || strings.Contains(text, "联网搜索"):
		return "web_search"
	case strings.Contains(text, "webfetch") || strings.Contains(text, "web fetch") || strings.Contains(text, "网页抓取"):
		return "web_fetch"
	case strings.Contains(text, "docwriter") || strings.Contains(text, "doc writer") || strings.Contains(text, "调研文档"):
		return "doc_writer"
	default:
		return "other"
	}
}

func extractHTTPURLs(text string) []string {
	return aiobserve.ExtractHTTPURLs(text)
}

func observedRunDurationSeconds(run model.ConversationRun) int64 {
	end := run.UpdatedAt
	if run.CompletedAt != nil {
		end = *run.CompletedAt
	}
	if end.Before(run.CreatedAt) {
		return 0
	}
	return int64(end.Sub(run.CreatedAt).Seconds())
}

func truncateForObservability(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "..."
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func formatObservedTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02 15:04:05")
}

func formatObservedTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatObservedTime(*value)
}

func observedCostStatus(run model.ConversationRun) string {
	if strings.TrimSpace(run.CostStatus) != "" {
		return run.CostStatus
	}
	if run.PromptTokens > 0 || run.CompletionTokens > 0 {
		return "estimated"
	}
	return "not_collected"
}

func observedCostCurrency(run model.ConversationRun) string {
	if strings.TrimSpace(run.CostCurrency) != "" {
		return run.CostCurrency
	}
	return "USD"
}
