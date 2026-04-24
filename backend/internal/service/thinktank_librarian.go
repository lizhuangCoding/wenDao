package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
	"wenDao/internal/pkg/eino"
)

// SourceRef 结果来源引用
type SourceRef struct {
	Kind  string
	ID    int64
	Title string
	URL   string
}

// LibrarianResult 本地知识检索结果
type LibrarianResult struct {
	CoverageStatus string
	Summary        string
	Sources        []SourceRef
	FollowupHint   string
}

// Librarian 本地知识代理
type Librarian interface {
	Search(ctx context.Context, question string) (LibrarianResult, error)
}

type librarianService struct {
	chain *eino.RAGChain
}

// NewLibrarianService 创建 Librarian 服务
func NewLibrarianService(chain *eino.RAGChain) Librarian {
	return &librarianService{chain: chain}
}

func (s *librarianService) Search(ctx context.Context, question string) (LibrarianResult, error) {
	if s.chain == nil {
		return LibrarianResult{CoverageStatus: "insufficient"}, nil
	}
	summary, docs, err := s.chain.SummarizeLocalFindings(ctx, question)
	if err != nil {
		return LibrarianResult{}, err
	}
	result := buildLibrarianResult(summary, docs)

	fallbackQuery := fallbackLocalSearchQuery(question)
	if result.CoverageStatus == "insufficient" && fallbackQuery != "" && fallbackQuery != strings.TrimSpace(question) {
		fallbackSummary, fallbackDocs, fallbackErr := s.chain.SummarizeLocalFindings(ctx, fallbackQuery)
		if fallbackErr != nil {
			return result, nil
		}
		fallbackResult := buildLibrarianResult(fallbackSummary, fallbackDocs)
		if fallbackResult.Summary != "" || len(fallbackResult.Sources) > 0 {
			fallbackResult.FollowupHint = fmt.Sprintf("已将宽泛检索改为核心主题检索：%s", fallbackQuery)
			return fallbackResult, nil
		}
	}
	return result, nil
}

func buildLibrarianResult(summary string, docs []*schema.Document) LibrarianResult {
	result := LibrarianResult{CoverageStatus: "insufficient"}
	if strings.TrimSpace(summary) != "" {
		result.Summary = summary
		if isNoLocalCoverageSummary(summary) {
			result.CoverageStatus = "insufficient"
		} else if len(docs) >= 2 {
			result.CoverageStatus = "sufficient"
		} else {
			result.CoverageStatus = "partial"
		}
	}
	for _, doc := range docs {
		result.Sources = append(result.Sources, buildSourceRefFromMetadata(doc.MetaData))
	}
	return result
}

func isNoLocalCoverageSummary(summary string) bool {
	text := strings.TrimSpace(summary)
	if text == "" {
		return true
	}
	noCoverageMarkers := []string{
		"未涵盖",
		"未涉及",
		"未覆盖",
		"没有覆盖",
		"未找到",
		"没有找到",
		"没有返回可用结果",
		"要点列表：无",
		"要点列表:无",
		"要点列表: 无",
	}
	for _, marker := range noCoverageMarkers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func fallbackLocalSearchQuery(query string) string {
	text := strings.TrimSpace(query)
	if text == "" {
		return ""
	}
	cutAt := len(text)
	for _, marker := range []string{"的", "、", "，", ",", "。", "；", ";", "以及", "等", "方面"} {
		if idx := strings.Index(text, marker); idx > 0 && idx < cutAt {
			cutAt = idx
		}
	}
	candidate := strings.TrimSpace(text[:cutAt])
	if candidate == "" || utf8.RuneCountInString(candidate) > 24 {
		return text
	}
	return candidate
}

func buildSourceRefFromMetadata(meta map[string]any) SourceRef {
	ref := SourceRef{Title: fmt.Sprintf("%v", meta["title"])}
	if kind, ok := meta["source_kind"].(string); ok {
		ref.Kind = kind
	} else {
		ref.Kind = "article"
	}
	ref.ID = int64FromMetadata(meta, "source_id")
	if ref.ID == 0 {
		ref.ID = int64FromMetadata(meta, "article_id")
	}
	if ref.Kind == "article" {
		if slug, ok := meta["slug"].(string); ok && slug != "" {
			ref.URL = "/article/" + slug
		}
	}
	return ref
}

func int64FromMetadata(meta map[string]any, key string) int64 {
	switch v := meta[key].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err == nil {
			return n
		}
	}
	return 0
}
