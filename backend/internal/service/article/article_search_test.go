package article

import (
	"strings"
	"testing"

	"wenDao/internal/model"
)

func TestBuildSearchSnippet_HighlightsSummaryMatch(t *testing.T) {
	article := &model.Article{
		Title:   "Go scheduler",
		Summary: "This article explains goroutine scheduling in Go.",
		Content: "Long content",
	}

	snippet := buildSearchSnippet(article, "goroutine")

	if !strings.Contains(snippet, "<mark>goroutine</mark>") {
		t.Fatalf("expected highlighted keyword in snippet, got %q", snippet)
	}
	if strings.Contains(snippet, "<script") {
		t.Fatalf("expected escaped snippet, got %q", snippet)
	}
}

func TestBuildSearchSnippet_FallsBackToContentAndEscapesHTML(t *testing.T) {
	article := &model.Article{
		Title:   "Security",
		Summary: "No relevant summary",
		Content: "Content includes <script>alert(1)</script> and privacy controls.",
	}

	snippet := buildSearchSnippet(article, "privacy")

	if !strings.Contains(snippet, "<mark>privacy</mark>") {
		t.Fatalf("expected highlighted content keyword, got %q", snippet)
	}
	if strings.Contains(snippet, "<script>") || !strings.Contains(snippet, "&lt;script") {
		t.Fatalf("expected escaped HTML content, got %q", snippet)
	}
}

func TestDetectArticleMatchedFields(t *testing.T) {
	article := &model.Article{
		Title:   "AI tools",
		Summary: "Writing workflow",
		Content: "Searchable body",
		Category: &model.Category{
			Name: "Product",
		},
		Tags: []*model.Tag{{Name: "Observability"}},
	}

	fields := detectArticleMatchedFields(article, "observability")

	if len(fields) != 1 || fields[0] != "tag" {
		t.Fatalf("expected tag match, got %#v", fields)
	}
}
