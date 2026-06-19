package chat

import (
	"strings"
	"time"

	"wenDao/config"
	"wenDao/internal/model"
	"wenDao/internal/pkg/aiobserve"
	"wenDao/internal/repository"
)

type RunMetricsConfig struct {
	Provider             string
	ModelName            string
	PromptPricePer1K     float64
	CompletionPricePer1K float64
	CostCurrency         string
}

func NewRunMetricsConfig(cfg config.AIConfig) RunMetricsConfig {
	return RunMetricsConfig{
		Provider:             strings.TrimSpace(cfg.Provider),
		ModelName:            strings.TrimSpace(cfg.LLMModel),
		PromptPricePer1K:     cfg.PromptPricePer1K,
		CompletionPricePer1K: cfg.CompletionPricePer1K,
		CostCurrency:         strings.TrimSpace(cfg.CostCurrency),
	}
}

type thinkTankRunRecorder struct {
	runRepo     repository.ConversationRunRepository
	runStepRepo repository.ConversationRunStepRepository
	logger      AILogger
	metrics     RunMetricsConfig
}

type thinkTankStepTracker struct {
	recorder       *thinkTankRunRecorder
	conversationID int64
	step           *model.ConversationRunStep
}

func newThinkTankRunRecorder(
	runRepo repository.ConversationRunRepository,
	runStepRepo repository.ConversationRunStepRepository,
	logger AILogger,
	metrics RunMetricsConfig,
) *thinkTankRunRecorder {
	return &thinkTankRunRecorder{
		runRepo:     runRepo,
		runStepRepo: runStepRepo,
		logger:      logger,
		metrics:     metrics,
	}
}

func (r *thinkTankRunRecorder) activeRun(conversationID int64) *model.ConversationRun {
	if r == nil || r.runRepo == nil || conversationID <= 0 {
		return nil
	}
	run, _ := r.runRepo.GetActiveByConversationID(conversationID)
	return run
}

func (r *thinkTankRunRecorder) startADKRun(conversationID int64, userID int64, question string, decision PlannerDecision, checkpointID string, pending *model.ConversationRun) int64 {
	if r == nil || r.runRepo == nil || conversationID <= 0 {
		return 0
	}
	run := &model.ConversationRun{
		ConversationID:   conversationID,
		UserID:           userID,
		Status:           "running",
		CurrentStage:     "analyzing",
		OriginalQuestion: question,
		LastPlan:         decision.PlanSummary,
		PendingContext:   marshalADKPendingContext(checkpointID, question),
		Provider:         r.metrics.Provider,
		ModelName:        r.metrics.ModelName,
		CostCurrency:     r.costCurrency(),
		CostStatus:       "not_collected",
	}
	if pending != nil {
		run.ID = pending.ID
		_ = r.runRepo.Update(run)
		return run.ID
	}
	_ = r.runRepo.Create(run)
	return run.ID
}

func (r *thinkTankRunRecorder) persistADKClarification(conversationID int64, userID int64, runID int64, question string, clarification string, checkpointID string, decision PlannerDecision) {
	if r == nil || r.runRepo == nil {
		return
	}
	run := &model.ConversationRun{
		ID:               runID,
		ConversationID:   conversationID,
		UserID:           userID,
		Status:           "waiting_user",
		CurrentStage:     "clarifying",
		OriginalQuestion: question,
		PendingQuestion:  &clarification,
		LastAnswer:       clarification,
		LastPlan:         decision.PlanSummary,
		PendingContext:   marshalADKPendingContext(checkpointID, question, clarification),
		Provider:         r.metrics.Provider,
		ModelName:        r.metrics.ModelName,
		CostCurrency:     r.costCurrency(),
		CostStatus:       "not_collected",
	}
	if run.ID > 0 {
		_ = r.runRepo.Update(run)
		return
	}
	_ = r.runRepo.Create(run)
}

func (r *thinkTankRunRecorder) persistAgentClarification(conversationID int64, userID int64, runID int64, question string, clarification string, stage string, pendingContext string, decision PlannerDecision) {
	if r == nil || r.runRepo == nil {
		return
	}
	if stage == "" {
		stage = "clarifying"
	}
	run := &model.ConversationRun{
		ID:               runID,
		ConversationID:   conversationID,
		UserID:           userID,
		Status:           "waiting_user",
		CurrentStage:     stage,
		OriginalQuestion: question,
		PendingQuestion:  &clarification,
		LastAnswer:       clarification,
		LastPlan:         decision.PlanSummary,
		PendingContext:   pendingContext,
		Provider:         r.metrics.Provider,
		ModelName:        r.metrics.ModelName,
		CostCurrency:     r.costCurrency(),
		CostStatus:       "not_collected",
	}
	if run.ID > 0 {
		_ = r.runRepo.Update(run)
		return
	}
	_ = r.runRepo.Create(run)
}

func (r *thinkTankRunRecorder) persistCompletedRun(conversationID int64, userID int64, question string, answer string, decision PlannerDecision) {
	if r == nil || r.runRepo == nil {
		return
	}
	now := time.Now()
	run := &model.ConversationRun{
		ConversationID:   conversationID,
		UserID:           userID,
		Status:           "completed",
		CurrentStage:     "completed",
		OriginalQuestion: question,
		LastAnswer:       answer,
		LastPlan:         decision.PlanSummary,
		PendingContext:   answer,
		CompletedAt:      &now,
		Provider:         r.metrics.Provider,
		ModelName:        r.metrics.ModelName,
	}
	r.applyCompletionMetrics(run, question, answer)
	if existing, _ := r.runRepo.GetActiveByConversationID(conversationID); existing != nil {
		run.ID = existing.ID
		run.SourceQualityScore = r.sourceQualityScore(existing.ID)
		_ = r.runRepo.Update(run)
		return
	}
	_ = r.runRepo.Create(run)
}

func (r *thinkTankRunRecorder) logStage(conv *model.Conversation, userID *int64, stage string, message string, detail string) {
	if r == nil || r.logger == nil {
		return
	}
	entry := AILogEntry{Stage: stage, Message: message, Detail: detail}
	if conv != nil {
		entry.ConversationID = conv.ID
	}
	if userID != nil {
		entry.UserID = *userID
	}
	if stage == "failed" {
		r.logger.LogError(entry)
		return
	}
	r.logger.LogStage(entry)
}

func (r *thinkTankRunRecorder) newStepTracker(conversationID int64, runID int64, agent string, summary string) *thinkTankStepTracker {
	step := &model.ConversationRunStep{
		ConversationID: conversationID,
		RunID:          runID,
		AgentName:      agent,
		Type:           "thinking",
		Summary:        summary,
		Status:         "running",
	}
	if r != nil && r.runStepRepo != nil && conversationID > 0 {
		_ = r.runStepRepo.Create(step)
	}
	return &thinkTankStepTracker{recorder: r, conversationID: conversationID, step: step}
}

func (r *thinkTankRunRecorder) updateStep(step *model.ConversationRunStep) {
	if r == nil || r.runStepRepo == nil || step == nil || step.ConversationID <= 0 {
		return
	}
	_ = r.runStepRepo.Update(step)
}

func (t *thinkTankStepTracker) snapshot() *model.ConversationRunStep {
	if t == nil || t.step == nil {
		return nil
	}
	clone := *t.step
	return &clone
}

func (t *thinkTankStepTracker) appendDetail(detail string) {
	if t == nil || t.step == nil {
		return
	}
	appendStepDetail(t.step, detail)
	if t.recorder != nil {
		t.recorder.updateStep(t.step)
	}
}

func (t *thinkTankStepTracker) setStatus(status string) {
	if t == nil || t.step == nil {
		return
	}
	t.step.Status = status
	if t.recorder != nil {
		t.recorder.updateStep(t.step)
	}
}

func (t *thinkTankStepTracker) fail(detail string) {
	if t == nil || t.step == nil {
		return
	}
	t.step.Status = "failed"
	appendStepDetail(t.step, detail)
	if t.recorder != nil {
		t.recorder.updateStep(t.step)
	}
}

func (t *thinkTankStepTracker) complete() {
	if t == nil || t.step == nil {
		return
	}
	t.step.Status = "completed"
	if t.recorder != nil {
		t.recorder.updateStep(t.step)
	}
}

func (s *thinkTankService) persistADKClarification(conversationID int64, userID int64, runID int64, question string, clarification string, checkpointID string, decision PlannerDecision) {
	s.runs.persistADKClarification(conversationID, userID, runID, question, clarification, checkpointID, decision)
}

func (s *thinkTankService) persistCompletedRun(conversationID int64, userID int64, question string, answer string, decision PlannerDecision) {
	s.runs.persistCompletedRun(conversationID, userID, question, answer, decision)
}

func (s *thinkTankService) logStage(conv *model.Conversation, userID *int64, stage string, message string, detail string) {
	s.runs.logStage(conv, userID, stage, message, detail)
}

func (r *thinkTankRunRecorder) updateProgress(runID int64, stage string, answer string) {
	if r == nil || r.runRepo == nil || runID <= 0 {
		return
	}
	run, err := r.runRepo.GetByID(runID)
	if err != nil || run == nil {
		return
	}
	if run.Status == "completed" || run.Status == "failed" {
		return
	}
	if strings.TrimSpace(stage) != "" {
		run.CurrentStage = stage
	}
	if answer != "" {
		run.LastAnswer = answer
	}
	now := time.Now()
	run.HeartbeatAt = &now
	_ = r.runRepo.Update(run)
}

func (r *thinkTankRunRecorder) touchHeartbeat(runID int64) {
	if r == nil || r.runRepo == nil || runID <= 0 {
		return
	}
	run, err := r.runRepo.GetByID(runID)
	if err != nil || run == nil {
		return
	}
	now := time.Now()
	run.HeartbeatAt = &now
	_ = r.runRepo.Update(run)
}

func (r *thinkTankRunRecorder) persistFailure(runID int64, err error) {
	if r == nil || r.runRepo == nil || runID <= 0 {
		return
	}
	run, getErr := r.runRepo.GetByID(runID)
	if getErr != nil || run == nil {
		return
	}
	run.Status = "failed"
	run.CurrentStage = "failed"
	if err != nil {
		message := err.Error()
		run.LastError = &message
		run.FailureCategory = aiobserve.ClassifyFailure(message)
		run.FailureFingerprint = aiobserve.FailureFingerprint(run.FailureCategory, message)
	}
	_ = r.runRepo.Update(run)
}

func (r *thinkTankRunRecorder) applyCompletionMetrics(run *model.ConversationRun, question string, answer string) {
	if run == nil {
		return
	}
	cost := aiobserve.EstimateCost(config.AIConfig{
		PromptPricePer1K:     r.metrics.PromptPricePer1K,
		CompletionPricePer1K: r.metrics.CompletionPricePer1K,
		CostCurrency:         r.costCurrency(),
	}, question, answer)
	run.PromptTokens = cost.PromptTokens
	run.CompletionTokens = cost.CompletionTokens
	run.EstimatedCost = cost.EstimatedCost
	run.CostCurrency = cost.Currency
	run.CostStatus = cost.Status
	run.FailureCategory = ""
	run.FailureFingerprint = ""
}

func (r *thinkTankRunRecorder) sourceQualityScore(runID int64) int {
	if r == nil || r.runStepRepo == nil || runID <= 0 {
		return 0
	}
	steps, err := r.runStepRepo.GetByRunID(runID)
	if err != nil {
		return 0
	}
	localHits := 0
	webHits := 0
	urls := make([]string, 0)
	for _, step := range steps {
		text := step.AgentName + " " + step.Type + " " + step.Summary + " " + step.Detail
		classification := strings.ToLower(text)
		if strings.Contains(classification, "localsearch") || strings.Contains(classification, "local search") || strings.Contains(classification, "redis 知识库") {
			localHits++
		}
		if strings.Contains(classification, "websearch") || strings.Contains(classification, "web search") || strings.Contains(classification, "webfetch") || strings.Contains(classification, "web fetch") || strings.Contains(classification, "联网搜索") || strings.Contains(classification, "网页抓取") {
			webHits++
		}
		urls = append(urls, aiobserve.ExtractHTTPURLs(text)...)
	}
	return aiobserve.ScoreSourceQuality(urls, localHits, webHits)
}

func (r *thinkTankRunRecorder) costCurrency() string {
	if strings.TrimSpace(r.metrics.CostCurrency) == "" {
		return "USD"
	}
	return strings.TrimSpace(r.metrics.CostCurrency)
}
