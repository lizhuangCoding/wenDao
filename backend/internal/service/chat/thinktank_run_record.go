package chat

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

type thinkTankRunRecorder struct {
	runRepo     repository.ConversationRunRepository
	runStepRepo repository.ConversationRunStepRepository
	logger      AILogger
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
) *thinkTankRunRecorder {
	return &thinkTankRunRecorder{
		runRepo:     runRepo,
		runStepRepo: runStepRepo,
		logger:      logger,
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
		PendingContext:   marshalADKPendingContext(checkpointID),
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
		PendingContext:   marshalADKPendingContext(checkpointID),
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
	}
	if existing, _ := r.runRepo.GetActiveByConversationID(conversationID); existing != nil {
		run.ID = existing.ID
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
	}
	_ = r.runRepo.Update(run)
}

func appendStepDetail(step *model.ConversationRunStep, detail string) {
	detail = normalizeStepDetail(detail)
	if step == nil || detail == "" {
		return
	}
	if strings.TrimSpace(step.Detail) == "" {
		step.Detail = detail
		return
	}
	step.Detail = normalizeStepDetail(strings.TrimSpace(step.Detail) + "\n\n" + detail)
}

func normalizeStepDetail(detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return ""
	}
	runes := []rune(detail)
	if len(runes) <= maxStepDetailRunes {
		return detail
	}
	return string(runes[:maxStepDetailRunes]) + "\n\n[内容过长，已截断；完整原始日志请查看服务端日志或工具结果源。]"
}

func formatLibrarianStepDetail(result LibrarianResult) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("检索完成。找到 %d 个相关来源。覆盖状态：%s", len(result.Sources), result.CoverageStatus))
	if strings.TrimSpace(result.Summary) != "" {
		builder.WriteString("\n\n站内知识摘要：\n")
		builder.WriteString(strings.TrimSpace(result.Summary))
	}
	if len(result.Sources) > 0 {
		builder.WriteString("\n\n站内来源：")
		for _, source := range result.Sources {
			builder.WriteString("\n")
			builder.WriteString(formatSourceRef(source))
		}
	}
	if strings.TrimSpace(result.FollowupHint) != "" {
		builder.WriteString("\n\n后续提示：")
		builder.WriteString(strings.TrimSpace(result.FollowupHint))
	}
	return builder.String()
}

func formatJournalistStepDetail(result *JournalistResult) string {
	if result == nil {
		return "外部调研完成，但没有返回可用结果。"
	}
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("外部调研完成。从 %d 个来源获取了信息。", len(result.Sources)))
	if strings.TrimSpace(result.Summary) != "" {
		builder.WriteString("\n\n外部调研结果：\n")
		builder.WriteString(strings.TrimSpace(result.Summary))
	}
	if len(result.Sources) > 0 {
		builder.WriteString("\n\n外部来源：")
		for _, source := range result.Sources {
			builder.WriteString("\n")
			builder.WriteString(formatSourceRef(source))
		}
	}
	if strings.TrimSpace(result.KnowledgeDraftTitle) != "" {
		builder.WriteString("\n\n知识草稿：")
		builder.WriteString(strings.TrimSpace(result.KnowledgeDraftTitle))
	}
	return builder.String()
}

func formatSynthesizerStepDetail(local LibrarianResult, web *JournalistResult, sources []string) string {
	var builder strings.Builder
	builder.WriteString("已完成多方结果整合，正在生成最终回答。")
	builder.WriteString(fmt.Sprintf("\n\n站内来源数：%d", len(local.Sources)))
	if web != nil {
		builder.WriteString(fmt.Sprintf("\n外部来源数：%d", len(web.Sources)))
	}
	if len(sources) > 0 {
		builder.WriteString("\n\n最终引用：")
		for _, source := range sources {
			if strings.TrimSpace(source) == "" {
				continue
			}
			builder.WriteString("\n- ")
			builder.WriteString(strings.TrimSpace(source))
		}
	}
	return builder.String()
}

func formatSourceRef(source SourceRef) string {
	title := strings.TrimSpace(source.Title)
	if title == "" {
		title = "未命名来源"
	}
	if strings.TrimSpace(source.URL) == "" {
		return "- " + title
	}
	return fmt.Sprintf("- %s (%s)", title, strings.TrimSpace(source.URL))
}

func formatADKActionDetail(action *adk.AgentAction) string {
	if action == nil {
		return ""
	}
	switch {
	case action.Interrupted != nil:
		question := extractADKClarificationQuestion(action.Interrupted)
		if question == "" {
			return "流程中断：等待用户补充信息"
		}
		return "流程中断：等待用户补充信息\n问题：" + question
	case action.TransferToAgent != nil:
		return "流程切换：切换为 " + action.TransferToAgent.DestAgentName + " Agent"
	case action.Exit:
		return "流程动作：当前 Agent 结束执行"
	case action.BreakLoop != nil:
		return "流程动作：跳出当前执行循环"
	case action.CustomizedAction != nil:
		if data, err := json.Marshal(action.CustomizedAction); err == nil {
			return "原始动作日志：\n" + string(data)
		}
		return fmt.Sprintf("原始动作日志：%v", action.CustomizedAction)
	default:
		return ""
	}
}
