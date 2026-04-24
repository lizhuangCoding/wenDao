package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"wenDao/internal/model"
	"wenDao/internal/pkg/eino"
)

// ConversationMemoryDraft is a normalized memory item ready for persistence.
type ConversationMemoryDraft struct {
	Scope      string
	Content    string
	Importance int
}

// ConversationMemorySummarizer extracts structured memories from older chat history.
type ConversationMemorySummarizer interface {
	Summarize(ctx context.Context, history []model.ChatMessage, existing []model.ConversationMemory) ([]ConversationMemoryDraft, error)
}

type llmConversationMemorySummarizer struct {
	llm eino.LLMClient
}

// NewConversationMemorySummarizer creates an LLM-backed memory summarizer.
func NewConversationMemorySummarizer(llm eino.LLMClient) ConversationMemorySummarizer {
	if llm == nil {
		return nil
	}
	return &llmConversationMemorySummarizer{llm: llm}
}

func (s *llmConversationMemorySummarizer) Summarize(ctx context.Context, history []model.ChatMessage, existing []model.ConversationMemory) ([]ConversationMemoryDraft, error) {
	if s == nil || s.llm == nil {
		return nil, nil
	}
	source := formatMemorySource(history)
	if strings.TrimSpace(source) == "" {
		return nil, nil
	}
	existingText := formatExistingMemories(existing)
	prompt := fmt.Sprintf(`请从以下对话中提取值得长期保留的记忆。

要求：
1. 只保留对后续回答有帮助的稳定信息，不要记录寒暄、临时状态或无关流水账。
2. 输出 JSON，不要 Markdown，不要解释。
3. JSON 字段必须为：
{
  "conversation_summary": "一句话概括会话主线",
  "user_preferences": ["用户稳定偏好"],
  "project_facts": ["项目事实"],
  "decisions": ["已经确认的技术/产品决策"],
  "open_threads": ["仍未完成或后续要处理的问题"]
}
4. 每个数组最多 3 条，每条不超过 80 个中文字符。

已有记忆：
%s

新增较早对话：
%s`, existingText, source)

	output, err := s.llm.Chat([]eino.ChatMessage{
		{Role: "system", Content: "你是对话记忆压缩器，负责把多轮对话整理成结构化长期记忆。"},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, err
	}
	return parseMemorySummaryOutput(output)
}

type memorySummaryOutput struct {
	ConversationSummary string   `json:"conversation_summary"`
	UserPreferences     []string `json:"user_preferences"`
	ProjectFacts        []string `json:"project_facts"`
	Decisions           []string `json:"decisions"`
	OpenThreads         []string `json:"open_threads"`
}

func parseMemorySummaryOutput(output string) ([]ConversationMemoryDraft, error) {
	output = strings.TrimSpace(output)
	output = strings.TrimPrefix(output, "```json")
	output = strings.TrimPrefix(output, "```")
	output = strings.TrimSuffix(output, "```")
	output = strings.TrimSpace(output)

	var parsed memorySummaryOutput
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		return nil, err
	}

	drafts := make([]ConversationMemoryDraft, 0, 5)
	appendDraft := func(scope, content string, importance int) {
		content = strings.TrimSpace(content)
		if content == "" {
			return
		}
		drafts = append(drafts, ConversationMemoryDraft{Scope: scope, Content: content, Importance: importance})
	}

	appendDraft(ConversationMemoryScopeSummary, parsed.ConversationSummary, 2)
	appendDraft(ConversationMemoryScopePreference, strings.Join(filterNonEmpty(parsed.UserPreferences), "\n"), 3)
	appendDraft(ConversationMemoryScopeProjectFact, strings.Join(filterNonEmpty(parsed.ProjectFacts), "\n"), 2)
	appendDraft(ConversationMemoryScopeDecision, strings.Join(filterNonEmpty(parsed.Decisions), "\n"), 3)
	appendDraft(ConversationMemoryScopeOpenThread, strings.Join(filterNonEmpty(parsed.OpenThreads), "\n"), 2)

	return drafts, nil
}

func filterNonEmpty(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func formatMemorySource(history []model.ChatMessage) string {
	var builder strings.Builder
	for _, msg := range history {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		builder.WriteString(msg.Role)
		builder.WriteString(": ")
		builder.WriteString(truncateRunes(content, 240))
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func formatExistingMemories(memories []model.ConversationMemory) string {
	if len(memories) == 0 {
		return "无"
	}
	var builder strings.Builder
	for _, memory := range memories {
		content := strings.TrimSpace(memory.Content)
		if content == "" {
			continue
		}
		builder.WriteString("- ")
		builder.WriteString(memory.Scope)
		builder.WriteString(": ")
		builder.WriteString(content)
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}
