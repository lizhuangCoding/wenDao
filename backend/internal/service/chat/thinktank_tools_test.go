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

func TestToolFailureHints_DoNotCarryFinalAnswerHardDenials(t *testing.T) {
	localSearchBase, err := newLocalSearchTool(nil)
	localSearchTool := mustInvokableTool(t, localSearchBase, err)
	webSearchBase, err := newWebSearchTool(ResearchConfig{})
	webSearchTool := mustInvokableTool(t, webSearchBase, err)
	webFetchBase, err := newWebFetchTool(ResearchConfig{TimeoutSeconds: 1})
	webFetchTool := mustInvokableTool(t, webFetchBase, err)

	tools := []struct {
		name  string
		tool  tool.InvokableTool
		input string
	}{
		{
			name:  "LocalSearch",
			tool:  localSearchTool,
			input: `{"query":"Redis"}`,
		},
		{
			name:  "WebSearch",
			tool:  webSearchTool,
			input: `{"query":"AI Agent"}`,
		},
		{
			name:  "WebFetch",
			tool:  webFetchTool,
			input: `{"url":"不是 URL"}`,
		},
	}

	for _, tt := range tools {
		result, err := tt.tool.InvokableRun(context.Background(), tt.input)
		if err != nil {
			t.Fatalf("%s should encode failures as tool output, got %v", tt.name, err)
		}
		if !strings.Contains(result, `"ok":false`) {
			t.Fatalf("%s expected structured failure, got %s", tt.name, result)
		}
		for _, forbidden := range []string{"不要", "不得", "禁止", "最终回答", "展示此工具失败"} {
			if strings.Contains(result, forbidden) {
				t.Fatalf("%s failure hint should be operational, not a hard final-answer guard %q: %s", tt.name, forbidden, result)
			}
		}
	}
}

func mustInvokableTool(t *testing.T, baseTool tool.BaseTool, err error) tool.InvokableTool {
	t.Helper()
	if err != nil {
		t.Fatalf("expected tool creation to succeed, got %v", err)
	}
	invokable, ok := baseTool.(tool.InvokableTool)
	if !ok {
		t.Fatal("expected tool to be invokable")
	}
	return invokable
}

func TestDocWriterTool_DoesNotReturnDraftMetadataToModel(t *testing.T) {
	knowledgeSvc := &stubKnowledgeDocumentService{}
	baseTool, err := newDocWriterTool(knowledgeSvc)
	if err != nil {
		t.Fatalf("expected tool creation to succeed, got %v", err)
	}
	invokable, ok := baseTool.(tool.InvokableTool)
	if !ok {
		t.Fatal("expected DocWriter to be invokable")
	}

	result, err := invokable.InvokableRun(context.Background(), `{"title":"李小龙调研报告","summary":"李小龙简介","content":"正文内容正文内容正文内容"}`)
	if err != nil {
		t.Fatalf("expected DocWriter result, got error %v", err)
	}
	if knowledgeSvc.created == nil || knowledgeSvc.created.Title != "李小龙调研报告" {
		t.Fatalf("expected draft to be persisted, got %#v", knowledgeSvc.created)
	}
	for _, forbidden := range []string{"ID=", "doc_id", "文档 ID", "文档ID", "李小龙调研报告", "知识文档草稿"} {
		if strings.Contains(result, forbidden) {
			t.Fatalf("DocWriter result must not expose internal metadata %q to model, got %q", forbidden, result)
		}
	}
}
