package chat

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

func TestWithRunID_PropagatesRunIDThroughToolContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithRunID(ctx, 42)

	if got := getRunID(ctx); got != 42 {
		t.Fatalf("expected run_id 42, got %d", got)
	}
}

func TestToolResultEnvelope_ExtractsLocalSearchEvidence(t *testing.T) {
	content := `{"ok":true,"data":{"coverage_status":"sufficient","summary":"站内摘要","sources":[{"Kind":"article","Title":"文章","URL":"/article/a"}]}}`

	if got := extractLocalSearchSummary(content); got != "站内摘要" {
		t.Fatalf("expected local summary from envelope, got %q", got)
	}
	sources := extractLocalSearchArticleSources(content)
	if len(sources) != 1 || sources[0].Title != "文章" || sources[0].URL != "/article/a" {
		t.Fatalf("expected article source from envelope, got %#v", sources)
	}
}

func TestToolResultEnvelope_ExtractsWebSearchEvidence(t *testing.T) {
	content := `{"ok":true,"data":{"organic":[{"title":"网页","link":"https://example.com","snippet":"摘要"}],"items":[{"title":"条目","url":"https://example.org"}]}}`

	sources := extractWebSearchSources(content)
	if len(sources) != 2 {
		t.Fatalf("expected web sources from envelope, got %#v", sources)
	}
	if got := summarizeWebSearchResult(content); !strings.Contains(got, "网页：摘要 (https://example.com)") {
		t.Fatalf("expected web summary from envelope, got %q", got)
	}
}

func TestLocalSearchTool_ReturnsStructuredFailureWhenUnavailable(t *testing.T) {
	baseTool, err := newLocalSearchTool(nil)
	if err != nil {
		t.Fatalf("expected tool creation to succeed, got %v", err)
	}
	invokable, ok := baseTool.(tool.InvokableTool)
	if !ok {
		t.Fatal("expected LocalSearch to be invokable")
	}

	result, err := invokable.InvokableRun(context.Background(), `{"query":"Redis"}`)
	if err != nil {
		t.Fatalf("expected tool failure to be encoded in result, got error %v", err)
	}
	if !strings.Contains(result, `"ok":false`) || !strings.Contains(result, `"error"`) {
		t.Fatalf("expected structured failure envelope, got %s", result)
	}
}
