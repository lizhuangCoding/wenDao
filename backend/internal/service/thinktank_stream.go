package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"

	"wenDao/internal/model"
)

type thinkTankStreamEmitter struct{}

func newThinkTankStreamEmitter() *thinkTankStreamEmitter {
	return &thinkTankStreamEmitter{}
}

func (e *thinkTankStreamEmitter) emitStage(eventCh chan<- StreamEvent, stage string, label string) {
	if eventCh == nil {
		return
	}
	eventCh <- StreamEvent{Type: StreamEventStage, Stage: stage, Label: label}
}

func (e *thinkTankStreamEmitter) emitQuestion(eventCh chan<- StreamEvent, stage string, question string) {
	if eventCh == nil {
		return
	}
	eventCh <- StreamEvent{Type: StreamEventQuestion, Stage: stage, Message: question}
}

func (e *thinkTankStreamEmitter) emitChunk(eventCh chan<- StreamEvent, message string, sources []string) {
	if eventCh == nil {
		return
	}
	eventCh <- StreamEvent{Type: StreamEventChunk, Message: message, Sources: sources}
}

func (e *thinkTankStreamEmitter) emitStep(eventCh chan<- StreamEvent, step *model.ConversationRunStep) {
	if eventCh == nil || step == nil {
		return
	}
	eventCh <- StreamEvent{
		Type:      StreamEventStep,
		StepID:    step.ID,
		AgentName: step.AgentName,
		Status:    step.Status,
		Summary:   step.Summary,
		Detail:    step.Detail,
	}
}

func (e *thinkTankStreamEmitter) emitDone(eventCh chan<- StreamEvent, stage string, label string) {
	if eventCh == nil {
		return
	}
	if strings.TrimSpace(label) != "" {
		e.emitStage(eventCh, stage, label)
	}
	eventCh <- StreamEvent{Type: StreamEventDone, Stage: stage}
}

func formatADKMessageDetail(msg *schema.Message) string {
	if msg == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	if strings.TrimSpace(msg.ReasoningContent) != "" {
		parts = append(parts, "思考步骤：\n"+strings.TrimSpace(msg.ReasoningContent))
	}
	for _, toolCall := range msg.ToolCalls {
		name := strings.TrimSpace(toolCall.Function.Name)
		if name == "" {
			name = "unknown_tool"
		}
		args := strings.TrimSpace(toolCall.Function.Arguments)
		if args == "" {
			args = "{}"
		}
		parts = append(parts, fmt.Sprintf("工具调用：%s\n参数：%s", name, args))
	}
	if strings.TrimSpace(msg.ToolName) != "" {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			content = "(空结果)"
		}
		parts = append(parts, fmt.Sprintf("工具返回：%s\n结果：%s", strings.TrimSpace(msg.ToolName), content))
	} else if strings.TrimSpace(msg.Content) != "" {
		parts = append(parts, "Agent 输出：\n"+strings.TrimSpace(msg.Content))
	}
	return strings.Join(parts, "\n\n")
}

func extractLocalSearchArticleSources(content string) []SourceRef {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	var payload struct {
		Sources []SourceRef `json:"sources"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil
	}
	result := make([]SourceRef, 0, len(payload.Sources))
	for _, source := range payload.Sources {
		if source.Kind == "" {
			source.Kind = "article"
		}
		if source.Kind != "article" || strings.TrimSpace(source.Title) == "" || strings.TrimSpace(source.URL) == "" {
			continue
		}
		result = append(result, source)
	}
	return result
}

func extractLocalSearchSummary(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	var payload struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Summary)
}

func extractWebSearchSources(content string) []SourceRef {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	var payload struct {
		Organic []struct {
			Title string `json:"title"`
			Link  string `json:"link"`
		} `json:"organic"`
		Items []struct {
			Title string `json:"title"`
			URL   string `json:"url"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil
	}
	result := make([]SourceRef, 0, len(payload.Organic)+len(payload.Items))
	for _, item := range payload.Organic {
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.Link) == "" {
			continue
		}
		result = append(result, SourceRef{Kind: "web", Title: item.Title, URL: item.Link})
	}
	for _, item := range payload.Items {
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.URL) == "" {
			continue
		}
		result = append(result, SourceRef{Kind: "web", Title: item.Title, URL: item.URL})
	}
	return result
}

func summarizeWebSearchResult(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	var payload struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return ""
	}
	lines := make([]string, 0, 5)
	for _, item := range payload.Organic {
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}
		line := title
		if snippet := strings.TrimSpace(item.Snippet); snippet != "" {
			line += "：" + snippet
		}
		if link := strings.TrimSpace(item.Link); link != "" {
			line += " (" + link + ")"
		}
		lines = append(lines, line)
		if len(lines) >= 5 {
			break
		}
	}
	return strings.Join(lines, "\n")
}

func appendNonEmptyNote(notes []string, note string) []string {
	note = strings.TrimSpace(note)
	if note == "" || looksLikeRawHTML(note) {
		return notes
	}
	if len([]rune(note)) > 1200 {
		note = string([]rune(note)[:1200]) + "..."
	}
	for _, existing := range notes {
		if existing == note {
			return notes
		}
	}
	return append(notes, note)
}

func compactNotes(notes []string) string {
	compacted := make([]string, 0, len(notes))
	for _, note := range notes {
		note = strings.TrimSpace(note)
		if note == "" || looksLikeRawHTML(note) {
			continue
		}
		compacted = append(compacted, note)
		if len(compacted) >= 6 {
			break
		}
	}
	return strings.Join(compacted, "\n\n")
}

func looksLikeRawHTML(text string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(text))
	return strings.HasPrefix(trimmed, "<!doctype html") || strings.HasPrefix(trimmed, "<html") || strings.Contains(trimmed, "<script")
}

func mergeSourceRefs(existing []SourceRef, next []SourceRef) []SourceRef {
	if len(next) == 0 {
		return existing
	}
	seen := make(map[string]struct{}, len(existing)+len(next))
	result := make([]SourceRef, 0, len(existing)+len(next))
	for _, source := range existing {
		key := source.Title + "|" + source.URL
		if key == "|" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, source)
	}
	for _, source := range next {
		key := source.Title + "|" + source.URL
		if key == "|" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, source)
	}
	return result
}

func splitStreamChunks(message string) []string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return []string{""}
	}
	parts := strings.Split(trimmed, "\n\n")
	chunks := make([]string, 0, len(parts))
	var builder strings.Builder
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(part)
		chunks = append(chunks, builder.String())
	}
	if len(chunks) == 0 {
		return []string{trimmed}
	}
	return chunks
}
