package service

import (
	"context"
	"strings"

	"wenDao/internal/model"
)

func (s *thinkTankService) loadConversationMemories(conversationID int64) []model.ConversationMemory {
	if s.memoryRepo == nil || conversationID <= 0 {
		return nil
	}
	memories, err := s.memoryRepo.GetByConversationID(conversationID)
	if err != nil {
		if s.logger != nil {
			s.logger.LogError(AILogEntry{ConversationID: conversationID, Stage: "memory", Message: "Failed to load conversation memory", Detail: err.Error()})
		}
		return nil
	}
	return memories
}

func (s *thinkTankService) updateConversationMemoryWithWarning(conversationID int64, userID int64, history []model.ChatMessage) {
	if s.memoryRepo == nil || conversationID <= 0 {
		return
	}
	olderEnd := len(history) - recentMemoryMessageCount
	if olderEnd <= 0 {
		return
	}
	older := history[:olderEnd]
	existing := s.loadConversationMemories(conversationID)
	drafts, err := s.summarizeConversationMemory(context.Background(), older, existing)
	if err != nil && s.logger != nil {
		s.logger.LogError(AILogEntry{ConversationID: conversationID, UserID: userID, Stage: "memory", Message: "Dynamic memory summarizer failed; using fallback", Detail: err.Error()})
	}
	if len(drafts) == 0 {
		return
	}
	for _, draft := range drafts {
		if strings.TrimSpace(draft.Scope) == "" || strings.TrimSpace(draft.Content) == "" {
			continue
		}
		importance := draft.Importance
		if importance <= 0 {
			importance = 1
		}
		memory := &model.ConversationMemory{
			ConversationID:       conversationID,
			UserID:               userID,
			Scope:                draft.Scope,
			Content:              strings.TrimSpace(draft.Content),
			SourceMessageIDStart: firstMessageID(older),
			SourceMessageIDEnd:   lastMessageID(older),
			Importance:           importance,
		}
		if err := s.memoryRepo.Upsert(memory); err != nil && s.logger != nil {
			s.logger.LogError(AILogEntry{ConversationID: conversationID, UserID: userID, Stage: "memory", Message: "Failed to update conversation memory", Detail: err.Error()})
		}
	}
}

func (s *thinkTankService) summarizeConversationMemory(ctx context.Context, older []model.ChatMessage, existing []model.ConversationMemory) ([]ConversationMemoryDraft, error) {
	if s.memorySummarizer != nil {
		drafts, err := s.memorySummarizer.Summarize(ctx, older, existing)
		if err == nil && len(drafts) > 0 {
			return drafts, nil
		}
		if err != nil {
			return fallbackMemoryDrafts(older), err
		}
	}
	return fallbackMemoryDrafts(older), nil
}

func fallbackMemoryDrafts(history []model.ChatMessage) []ConversationMemoryDraft {
	summary := summarizeOlderConversationMemory(history)
	if summary == "" {
		return nil
	}
	return []ConversationMemoryDraft{{Scope: ConversationMemoryScopeSummary, Content: summary, Importance: 1}}
}

func appendConversationTurn(history []model.ChatMessage, question string, answer string) []model.ChatMessage {
	next := make([]model.ChatMessage, 0, len(history)+2)
	next = append(next, history...)
	next = append(next, model.ChatMessage{Role: "user", Content: question})
	next = append(next, model.ChatMessage{Role: "assistant", Content: answer})
	return next
}

func firstMessageID(messages []model.ChatMessage) int64 {
	for _, msg := range messages {
		if msg.ID != 0 {
			return msg.ID
		}
	}
	return 0
}

func lastMessageID(messages []model.ChatMessage) int64 {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].ID != 0 {
			return messages[i].ID
		}
	}
	return 0
}

func buildConversationMemory(history []model.ChatMessage, memories []model.ConversationMemory) string {
	return buildConversationMemoryForQuestion("", history, memories)
}

func buildConversationMemoryForQuestion(question string, history []model.ChatMessage, memories []model.ConversationMemory) string {
	if len(history) == 0 && len(memories) == 0 {
		return ""
	}
	var builder strings.Builder

	if len(memories) > 0 {
		builder.WriteString("长期记忆：")
		for _, memory := range memories {
			content := strings.TrimSpace(memory.Content)
			if content == "" {
				continue
			}
			builder.WriteString("\n- ")
			builder.WriteString(content)
		}
	}

	start := selectRecentMemoryStart(question, history)
	older := history[:start]
	if len(older) > 0 {
		if summary := summarizeOlderConversationMemory(older); summary != "" {
			if builder.Len() > 0 {
				builder.WriteString("\n\n")
			}
			builder.WriteString("更早对话摘要：\n- ")
			builder.WriteString(summary)
		}
	}
	relevant := selectRelevantOlderMessages(question, older, 3)
	if len(relevant) > 0 {
		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString("相关历史片段：")
		for _, msg := range relevant {
			builder.WriteString("\n")
			builder.WriteString(msg.Role)
			builder.WriteString(": ")
			builder.WriteString(truncateRunes(strings.TrimSpace(msg.Content), 160))
		}
	}

	if builder.Len() > 0 && len(history[start:]) > 0 {
		builder.WriteString("\n\n")
	}
	if len(history[start:]) > 0 {
		builder.WriteString("最近对话：\n")
	}
	for _, msg := range history[start:] {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		builder.WriteString(msg.Role)
		builder.WriteString(": ")
		builder.WriteString(truncateRunes(content, 120))
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func compressConversationMemory(history []model.ChatMessage) string {
	return buildConversationMemoryForQuestion("", history, nil)
}

func selectRecentMemoryStart(question string, history []model.ChatMessage) int {
	if len(history) == 0 {
		return 0
	}
	if strings.TrimSpace(question) == "" && len(history) > recentMemoryMessageCount {
		return len(history) - recentMemoryMessageCount
	}
	budget := recentMemoryRuneBudget(question)
	used := 0
	start := len(history)
	for i := len(history) - 1; i >= 0; i-- {
		msgCost := len([]rune(strings.TrimSpace(history[i].Role))) + len([]rune(strings.TrimSpace(history[i].Content))) + 4
		if start < len(history) && used+msgCost > budget {
			break
		}
		used += msgCost
		start = i
	}
	if len(history)-start < 2 && len(history) >= 2 {
		start = len(history) - 2
	}
	return start
}

func recentMemoryRuneBudget(question string) int {
	question = strings.TrimSpace(question)
	runeCount := len([]rune(question))
	if runeCount <= 8 || isReferentialQuestion(question) {
		return 1200
	}
	if runeCount > 120 {
		return 480
	}
	return 720
}

func isReferentialQuestion(question string) bool {
	if question == "" {
		return false
	}
	for _, token := range []string{"继续", "这个", "那个", "上面", "刚才", "前面", "原文", "这篇", "它", "他们", "这种"} {
		if strings.Contains(question, token) {
			return true
		}
	}
	return false
}

func selectRelevantOlderMessages(question string, older []model.ChatMessage, limit int) []model.ChatMessage {
	if limit <= 0 || len(older) == 0 {
		return nil
	}
	keywords := memoryKeywords(question)
	if len(keywords) == 0 {
		return nil
	}
	result := make([]model.ChatMessage, 0, limit)
	for _, msg := range older {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		for _, keyword := range keywords {
			if strings.Contains(strings.ToLower(content), keyword) {
				result = append(result, msg)
				break
			}
		}
		if len(result) >= limit {
			break
		}
	}
	return result
}

func memoryKeywords(question string) []string {
	question = strings.ToLower(strings.TrimSpace(question))
	if question == "" {
		return nil
	}
	parts := strings.FieldsFunc(question, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t' || r == ',' || r == '，' || r == '。' || r == '?' || r == '？' || r == '!' || r == '！' || r == '；' || r == ';'
	})
	seen := make(map[string]struct{})
	keywords := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len([]rune(part)) < 2 || isMemoryStopword(part) {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		keywords = append(keywords, part)
	}
	return keywords
}

func isMemoryStopword(token string) bool {
	switch token {
	case "这个", "那个", "还有", "继续", "需要", "什么", "一下", "如何", "怎么", "我们", "你们", "他们", "the", "and", "for", "with":
		return true
	default:
		return false
	}
}

func summarizeOlderConversationMemory(history []model.ChatMessage) string {
	snippets := make([]string, 0, 3)
	for _, msg := range history {
		content := extractMemorySnippet(msg.Content)
		if content == "" {
			continue
		}
		if msg.Role == "user" {
			snippets = append(snippets, content)
		}
		if len(snippets) >= 3 {
			break
		}
	}
	if len(snippets) == 0 {
		for _, msg := range history {
			content := extractMemorySnippet(msg.Content)
			if content == "" {
				continue
			}
			snippets = append(snippets, content)
			if len(snippets) >= 3 {
				break
			}
		}
	}
	if len(snippets) == 0 {
		return ""
	}
	return "更早对话主要讨论：" + strings.Join(snippets, "；") + "。"
}

func extractMemorySnippet(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	content = strings.ReplaceAll(content, "\n", " ")
	cutAt := -1
	for _, sep := range []string{"。", "，", ",", "；", ";", "！", "？", "!", "?"} {
		if idx := strings.Index(content, sep); idx > 0 {
			if cutAt == -1 || idx < cutAt {
				cutAt = idx
			}
		}
	}
	if cutAt > 0 {
		content = content[:cutAt]
	}
	content = strings.TrimSpace(content)
	return strings.TrimRight(truncateRunes(content, maxMemorySnippetRunes), "。；，,;!?！？")
}

func truncateRunes(content string, limit int) string {
	content = strings.TrimSpace(content)
	if limit <= 0 {
		return ""
	}
	runes := []rune(content)
	if len(runes) <= limit {
		return content
	}
	return string(runes[:limit])
}

func buildAgentQuery(question string, memory string) string {
	memory = strings.TrimSpace(memory)
	if memory == "" {
		return question
	}
	return "历史上下文：\n" + memory + "\n\n当前问题：\n" + question
}
