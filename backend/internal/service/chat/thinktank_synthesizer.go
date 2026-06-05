package chat

import (
	"context"
	"fmt"
	"strings"

	"wenDao/internal/pkg/eino"
)

// ThinkTankSynthesizer 汇总代理
type ThinkTankSynthesizer interface {
	Compose(ctx context.Context, question string, local LibrarianResult, web *JournalistResult) (string, []string, error)
}

type thinkTankSynthesizer struct {
	llm eino.LLMClient
}

const thinkTankSynthesizerInstruction = "你是问道博客的最终答复撰写助手。请输出给用户看的答案正文，用中文直接回答问题，聚焦结论、依据和建议。参考链接会由系统追加到末尾。"

// NewThinkTankSynthesizer 创建汇总代理
func NewThinkTankSynthesizer(llm eino.LLMClient) ThinkTankSynthesizer {
	return &thinkTankSynthesizer{llm: llm}
}

func (s *thinkTankSynthesizer) Compose(ctx context.Context, question string, local LibrarianResult, web *JournalistResult) (string, []string, error) {
	var answer string
	if s.llm == nil {
		if web != nil && web.Summary != "" {
			answer = web.Summary
		} else {
			answer = local.Summary
		}
	} else {
		var builder strings.Builder
		if local.Summary != "" {
			builder.WriteString("材料 A：\n")
			builder.WriteString(local.Summary)
			builder.WriteString("\n\n")
		}
		if web != nil && web.Summary != "" {
			builder.WriteString("材料 B：\n")
			builder.WriteString(web.Summary)
		}
		messages := []eino.ChatMessage{
			{Role: "system", Content: thinkTankSynthesizerInstruction},
			{Role: "user", Content: fmt.Sprintf("问题：%s\n\n以下材料用于组织答案，最终答案应聚焦用户问题本身：\n%s", question, builder.String())},
		}
		var err error
		answer, err = s.llm.Chat(ctx, messages)
		if err != nil {
			return "", nil, err
		}
	}

	answer = appendGroupedReferences(answer, local.Sources, webSources(web))
	return answer, collectSourceTitles(local, web), nil
}

func appendArticleReferences(answer string, sources []SourceRef) string {
	return appendGroupedReferences(answer, sources, nil)
}

func appendGroupedReferences(answer string, blogSources []SourceRef, externalSources []SourceRef) string {
	blogLines := referenceLines(answer, blogSources, "article")
	externalLines := referenceLines(answer, externalSources, "web")
	if len(blogLines) == 0 && len(externalLines) == 0 {
		return answer
	}
	trimmed := strings.TrimSpace(answer)
	if trimmed != "" {
		trimmed += "\n\n"
	}
	if len(blogLines) > 0 {
		trimmed += "参考博主文章\n" + strings.Join(blogLines, "\n")
	}
	if len(externalLines) > 0 {
		if len(blogLines) > 0 {
			trimmed += "\n\n"
		}
		trimmed += "参考外部文章\n" + strings.Join(externalLines, "\n")
	}
	return trimmed
}

func referenceLines(answer string, sources []SourceRef, kind string) []string {
	lines := make([]string, 0)
	seen := make(map[string]struct{})
	for _, source := range sources {
		if source.Kind != kind || source.Title == "" || source.URL == "" {
			continue
		}
		if strings.Contains(answer, source.URL) {
			continue
		}
		key := source.Title + "|" + source.URL
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		lines = append(lines, fmt.Sprintf("- [%s](%s)", source.Title, source.URL))
	}
	return lines
}

func webSources(web *JournalistResult) []SourceRef {
	if web == nil {
		return nil
	}
	return web.Sources
}

func collectSourceTitles(local LibrarianResult, web *JournalistResult) []string {
	result := make([]string, 0, len(local.Sources)+4)
	for _, source := range local.Sources {
		if source.Title != "" {
			result = append(result, source.Title)
		}
	}
	if web != nil {
		for _, source := range web.Sources {
			if source.Title != "" {
				result = append(result, source.Title)
			}
		}
	}
	return result
}

func collectSourceRefTitles(groups ...[]SourceRef) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, sources := range groups {
		for _, source := range sources {
			title := strings.TrimSpace(source.Title)
			if title == "" {
				continue
			}
			if _, ok := seen[title]; ok {
				continue
			}
			seen[title] = struct{}{}
			result = append(result, title)
		}
	}
	return result
}
