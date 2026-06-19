package article

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

// ArticleSearchResult 站内搜索返回项。
type ArticleSearchResult struct {
	Article       *model.Article `json:"article"`
	Snippet       string         `json:"snippet"`
	MatchedFields []string       `json:"matched_fields"`
}

func (s *articleService) SearchArticles(keyword string, categoryID, tagID int64, page, pageSize int) ([]ArticleSearchResult, int64, error) {
	keyword = strings.TrimSpace(keyword)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	results, total, err := s.articleRepo.Search(repository.ArticleSearchFilter{
		Keyword:    keyword,
		CategoryID: categoryID,
		TagID:      tagID,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search articles: %w", err)
	}

	payload := make([]ArticleSearchResult, 0, len(results))
	for _, result := range results {
		if result.Article == nil {
			continue
		}
		payload = append(payload, ArticleSearchResult{
			Article:       result.Article,
			Snippet:       buildSearchSnippet(result.Article, keyword),
			MatchedFields: detectArticleMatchedFields(result.Article, keyword),
		})
	}
	return payload, total, nil
}

func detectArticleMatchedFields(article *model.Article, keyword string) []string {
	if article == nil {
		return nil
	}
	keyword = strings.TrimSpace(strings.ToLower(keyword))
	if keyword == "" {
		return []string{"filter"}
	}

	matched := make([]string, 0, 5)
	if strings.Contains(strings.ToLower(article.Title), keyword) {
		matched = append(matched, "title")
	}
	if strings.Contains(strings.ToLower(article.Summary), keyword) {
		matched = append(matched, "summary")
	}
	if strings.Contains(strings.ToLower(article.Content), keyword) {
		matched = append(matched, "content")
	}
	if article.Category != nil && strings.Contains(strings.ToLower(article.Category.Name), keyword) {
		matched = append(matched, "category")
	}
	for _, tag := range article.Tags {
		if tag != nil && strings.Contains(strings.ToLower(tag.Name), keyword) {
			matched = append(matched, "tag")
			break
		}
	}
	if len(matched) == 0 {
		return []string{"filter"}
	}
	return matched
}

func buildSearchSnippet(article *model.Article, keyword string) string {
	if article == nil {
		return ""
	}
	text := article.Summary
	if strings.TrimSpace(text) == "" || !containsFold(text, keyword) {
		text = article.Content
	}
	text = compactWhitespace(stripMarkdownMarkup(text))
	if text == "" {
		text = article.Title
	}

	const maxRunes = 160
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return html.EscapeString(truncateRunes(text, maxRunes))
	}

	lowerText := strings.ToLower(text)
	lowerKeyword := strings.ToLower(keyword)
	index := strings.Index(lowerText, lowerKeyword)
	if index >= 0 {
		text = centeredSnippet(text, index, len(keyword), maxRunes)
	} else {
		text = truncateRunes(text, maxRunes)
	}
	return highlightKeywordHTML(text, keyword)
}

func containsFold(text, keyword string) bool {
	keyword = strings.TrimSpace(keyword)
	return keyword == "" || strings.Contains(strings.ToLower(text), strings.ToLower(keyword))
}

func compactWhitespace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func stripMarkdownMarkup(text string) string {
	replacer := strings.NewReplacer("#", " ", "*", " ", "`", " ", ">", " ", "[", " ", "]", " ", "(", " ", ")", " ")
	return replacer.Replace(text)
}

func truncateRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}

func centeredSnippet(text string, byteIndex, keywordBytes, limit int) string {
	runes := []rune(text)
	prefixRunes := len([]rune(text[:byteIndex]))
	keywordRunes := len([]rune(text[byteIndex : byteIndex+keywordBytes]))
	if len(runes) <= limit {
		return text
	}

	start := prefixRunes - limit/3
	if start < 0 {
		start = 0
	}
	end := start + limit
	if end < prefixRunes+keywordRunes {
		end = prefixRunes + keywordRunes
	}
	if end > len(runes) {
		end = len(runes)
		start = end - limit
		if start < 0 {
			start = 0
		}
	}

	snippet := string(runes[start:end])
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(runes) {
		snippet += "..."
	}
	return snippet
}

func highlightKeywordHTML(text, keyword string) string {
	escaped := html.EscapeString(text)
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return escaped
	}
	pattern := regexp.MustCompile("(?i)" + regexp.QuoteMeta(html.EscapeString(keyword)))
	return pattern.ReplaceAllStringFunc(escaped, func(match string) string {
		return "<mark>" + match + "</mark>"
	})
}
