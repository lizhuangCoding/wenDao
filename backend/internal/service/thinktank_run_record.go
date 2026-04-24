package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"

	"wenDao/internal/model"
)

func (s *thinkTankService) persistADKClarification(conversationID int64, userID int64, runID int64, question string, clarification string, checkpointID string, decision PlannerDecision) {
	if s.runRepo == nil {
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
		LastPlan:         decision.PlanSummary,
		PendingContext:   marshalADKPendingContext(checkpointID),
	}
	if run.ID > 0 {
		_ = s.runRepo.Update(run)
		return
	}
	_ = s.runRepo.Create(run)
}

func (s *thinkTankService) persistCompletedRun(conversationID int64, userID int64, question string, answer string, decision PlannerDecision) {
	if s.runRepo == nil {
		return
	}
	now := time.Now()
	run := &model.ConversationRun{
		ConversationID:   conversationID,
		UserID:           userID,
		Status:           "completed",
		CurrentStage:     "completed",
		OriginalQuestion: question,
		LastPlan:         decision.PlanSummary,
		PendingContext:   answer,
		CompletedAt:      &now,
	}
	if existing, _ := s.runRepo.GetActiveByConversationID(conversationID); existing != nil {
		run.ID = existing.ID
		_ = s.runRepo.Update(run)
		return
	}
	_ = s.runRepo.Create(run)
}

func (s *thinkTankService) logStage(conv *model.Conversation, userID *int64, stage string, message string, detail string) {
	if s.logger == nil {
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
		s.logger.LogError(entry)
		return
	}
	s.logger.LogStage(entry)
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
