package chat

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestBuildSourceRefFromMetadata_BuildsArticleDetailURLFromSlug(t *testing.T) {
	ref := buildSourceRefFromMetadata(map[string]any{
		"source_kind": "article",
		"source_id":   int64(12),
		"title":       "李小龙的功夫哲学",
		"slug":        "lee-philosophy",
	})
	if ref.Kind != "article" {
		t.Fatalf("expected article source kind, got %q", ref.Kind)
	}
	if ref.URL != "/article/lee-philosophy" {
		t.Fatalf("expected article detail url, got %q", ref.URL)
	}
}

func TestBuildSourceRefFromMetadata_UsesArticleIDFallback(t *testing.T) {
	ref := buildSourceRefFromMetadata(map[string]any{
		"article_id": "42",
		"title":      "李小龙",
		"slug":       "bruce-lee",
	})
	if ref.Kind != "article" {
		t.Fatalf("expected article kind, got %q", ref.Kind)
	}
	if ref.ID != 42 {
		t.Fatalf("expected article id fallback, got %d", ref.ID)
	}
	if ref.URL != "/article/bruce-lee" {
		t.Fatalf("expected article detail url, got %q", ref.URL)
	}
}

func TestBuildLibrarianResult_DoesNotMarkNoCoverageSummarySufficient(t *testing.T) {
	docs := []*schema.Document{
		{MetaData: map[string]any{"title": "李小龙", "slug": "bruce-lee"}},
		{MetaData: map[string]any{"title": "李小龙", "slug": "bruce-lee"}},
		{MetaData: map[string]any{"title": "李小龙", "slug": "bruce-lee"}},
	}
	result := buildLibrarianResult("结论：文章片段中未涵盖李小龙的生平经历。\n要点列表：无", docs)
	if result.CoverageStatus != "insufficient" {
		t.Fatalf("expected insufficient coverage, got %q", result.CoverageStatus)
	}
}

func TestBuildLibrarianResult_TreatsNoCoveragePhraseAsInsufficient(t *testing.T) {
	docs := []*schema.Document{
		{MetaData: map[string]any{"title": "k8s学习笔记", "slug": "d4735e3a26"}},
		{MetaData: map[string]any{"title": "k8s学习笔记", "slug": "d4735e3a26"}},
		{MetaData: map[string]any{"title": "k8s学习笔记", "slug": "d4735e3a26"}},
	}
	result := buildLibrarianResult("本站没有覆盖关于K8s中节点、Pod、Deployment、Service等基础概念的定义和作用的内容。", docs)
	if result.CoverageStatus != "insufficient" {
		t.Fatalf("expected insufficient coverage, got %q", result.CoverageStatus)
	}
}

func TestBuildLibrarianResult_TreatsNoAboutPhraseAsInsufficient(t *testing.T) {
	docs := []*schema.Document{
		{MetaData: map[string]any{"title": "fjjjjj", "slug": "9400f1b21c"}},
		{MetaData: map[string]any{"title": "wfoijejfioew", "slug": "f5ca38f748"}},
	}
	result := buildLibrarianResult("本站没有关于马斯克的相关内容。", docs)
	if result.CoverageStatus != "insufficient" {
		t.Fatalf("expected insufficient coverage, got %q", result.CoverageStatus)
	}
}

func TestFallbackLocalSearchQuery_ExtractsCoreSubject(t *testing.T) {
	query := fallbackLocalSearchQuery("李小龙的生平经历、武术成就、电影作品、对武术和电影界的影响等方面的信息")
	if query != "李小龙" {
		t.Fatalf("expected core subject query, got %q", query)
	}
}
