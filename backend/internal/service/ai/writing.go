package ai

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"wenDao/internal/pkg/eino"
)

type WritingAction string

const (
	WritingActionPolish   WritingAction = "polish"
	WritingActionExpand   WritingAction = "expand"
	WritingActionShorten  WritingAction = "shorten"
	WritingActionSEOTitle WritingAction = "seo-title"
)

var (
	ErrUnsupportedWritingAction = errors.New("unsupported writing action")
	ErrWritingContentEmpty      = errors.New("writing content is empty")
)

type WritingRequest struct {
	Action  WritingAction
	Content string
	Title   string
	Summary string
}

type WritingResult struct {
	Result      string
	Suggestions []string
}

type writingPromptSpec struct {
	system string
	task   string
}

var writingPromptSpecs = map[WritingAction]writingPromptSpec{
	WritingActionPolish: {
		system: "你是一个专业的中文技术博客编辑，擅长在不改变作者观点的前提下提升表达质量。",
		task: `请润色下面这段 Markdown 内容。
要求：
1. 保持原意和事实不变，并保持 Markdown 结构。
2. 优化语气、逻辑衔接和可读性。
3. 不要补充作者没有表达的新观点。
4. 直接返回润色后的 Markdown，不要解释过程。`,
	},
	WritingActionExpand: {
		system: "你是一个专业的中文技术博客编辑，擅长把简略段落扩写成信息更完整的内容。",
		task: `请扩写下面这段 Markdown 内容。
要求：
1. 保持原主题和 Markdown 结构。
2. 补充必要的背景、解释、例子或过渡句，让内容更完整。
3. 不要编造具体数据、引用或不存在的事实。
4. 直接返回扩写后的 Markdown，不要解释过程。`,
	},
	WritingActionShorten: {
		system: "你是一个专业的中文技术博客编辑，擅长压缩冗余表达并保留关键信息。",
		task: `请缩写下面这段 Markdown 内容。
要求：
1. 保留核心观点、关键术语和 Markdown 结构。
2. 删除重复、松散和不必要的表达。
3. 让内容更简洁，但不要变成摘要标题。
4. 直接返回缩写后的 Markdown，不要解释过程。`,
	},
	WritingActionSEOTitle: {
		system: "你是一个专业的 SEO 博客标题顾问，擅长生成自然、不夸张、适合搜索展示的中文标题。",
		task: `请根据文章信息生成 5 个 SEO 友好的中文标题候选。
要求：
1. 每个标题 15-35 个中文字符左右。
2. 标题要清晰体现主题和读者收益。
3. 不要标题党，不要使用夸张承诺。
4. 每行只输出一个标题，可以使用编号。`,
	},
}

var titleListPrefixPattern = regexp.MustCompile(`^\s*(?:[-*]\s+|\d+[.)、]\s*)`)

func (s *aiService) GenerateWriting(ctx context.Context, req WritingRequest) (WritingResult, error) {
	spec, ok := writingPromptSpecs[req.Action]
	if !ok {
		return WritingResult{}, fmt.Errorf("%w: %q", ErrUnsupportedWritingAction, req.Action)
	}

	content := trimRunes(strings.TrimSpace(req.Content), 4000)
	if content == "" {
		return WritingResult{}, ErrWritingContentEmpty
	}
	if ctx == nil {
		ctx = context.Background()
	}

	messages := []eino.ChatMessage{
		{Role: "system", Content: spec.system},
		{Role: "user", Content: buildWritingPrompt(spec, req, content)},
	}

	raw, err := s.llmClient.Chat(ctx, messages)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("Failed to generate writing assistance", zap.String("action", string(req.Action)), zap.Error(err))
		}
		return WritingResult{}, fmt.Errorf("failed to generate writing assistance: %w", err)
	}

	result := strings.TrimSpace(raw)
	if req.Action == WritingActionSEOTitle {
		suggestions := parseSEOTitleSuggestions(result)
		if len(suggestions) > 0 {
			return WritingResult{Result: suggestions[0], Suggestions: suggestions}, nil
		}
	}

	return WritingResult{Result: result}, nil
}

func buildWritingPrompt(spec writingPromptSpec, req WritingRequest, content string) string {
	var builder strings.Builder
	builder.WriteString(spec.task)
	builder.WriteString("\n\n")

	if title := strings.TrimSpace(req.Title); title != "" {
		builder.WriteString("当前文章标题：")
		builder.WriteString(trimRunes(title, 120))
		builder.WriteString("\n")
	}
	if summary := strings.TrimSpace(req.Summary); summary != "" {
		builder.WriteString("当前文章摘要：")
		builder.WriteString(trimRunes(summary, 300))
		builder.WriteString("\n")
	}

	builder.WriteString("Markdown 内容：\n")
	builder.WriteString(content)
	return builder.String()
}

func parseSEOTitleSuggestions(raw string) []string {
	lines := strings.Split(raw, "\n")
	suggestions := make([]string, 0, 5)
	seen := make(map[string]struct{})

	for _, line := range lines {
		title := strings.TrimSpace(titleListPrefixPattern.ReplaceAllString(line, ""))
		title = strings.Trim(title, `"'“”‘’`)
		if title == "" {
			continue
		}
		if _, exists := seen[title]; exists {
			continue
		}
		seen[title] = struct{}{}
		suggestions = append(suggestions, title)
		if len(suggestions) == 5 {
			break
		}
	}

	if len(suggestions) == 0 {
		fallback := strings.TrimSpace(raw)
		if fallback != "" {
			suggestions = append(suggestions, fallback)
		}
	}

	return suggestions
}

func trimRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
