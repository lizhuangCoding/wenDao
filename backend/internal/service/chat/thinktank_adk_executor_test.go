package chat

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
)

func TestAskForClarificationTool_InterruptsThenUsesResumeInput(t *testing.T) {
	tool, err := newAskForClarificationTool()
	if err != nil {
		t.Fatalf("expected tool to be created, got %v", err)
	}

	_, err = tool.InvokableRun(context.Background(), `{"question":"请补充时间范围"}`)
	if err == nil {
		t.Fatalf("expected ask_for_clarification to interrupt without resume input")
	}
	if !strings.Contains(err.Error(), "interrupt signal") || !strings.Contains(err.Error(), "请补充时间范围") {
		t.Fatalf("expected interrupt-and-rerun error, got %v", err)
	}

	answer, err := tool.InvokableRun(context.Background(), `{"question":"请补充时间范围"}`, WithNewInput("最近三个月"))
	if err != nil {
		t.Fatalf("expected resume input to satisfy clarification, got %v", err)
	}
	if answer != "最近三个月" {
		t.Fatalf("expected resume answer, got %q", answer)
	}
}

func TestThinkTankExecutorInstruction_UsesDirectToolsWithoutSupervisorTransfer(t *testing.T) {
	if !strings.Contains(thinkTankExecutorInstruction, "ask_for_clarification") {
		t.Fatalf("executor instruction must expose ask_for_clarification")
	}
	if !strings.Contains(thinkTankExecutorInstruction, "检索 Redis 知识库") || !strings.Contains(thinkTankExecutorInstruction, "LocalSearch") {
		t.Fatalf("executor instruction must map Redis knowledge-base retrieval to LocalSearch")
	}
	if !strings.Contains(thinkTankExecutorInstruction, "Do not call transfer_to_agent") {
		t.Fatalf("executor instruction must forbid supervisor transfer calls")
	}
	if !strings.Contains(thinkTankExecutorInstruction, "Plan-Execute-Replan") {
		t.Fatalf("executor instruction must describe the plan-execute-replan loop")
	}
}

func TestThinkTankExecutorInstruction_ForbidsDocParserAndInvalidWebFetchURL(t *testing.T) {
	required := []string{
		"LocalSearch, WebSearch, WebFetch, DocWriter, ask_for_clarification",
		"Do not request or mention unavailable tools such as DocParser",
		"valid absolute http:// or https:// URL",
		"do not call WebFetch",
		"raw HTML",
	}
	for _, text := range required {
		if !strings.Contains(thinkTankExecutorInstruction, text) {
			t.Fatalf("executor instruction must contain %q", text)
		}
	}
}

func TestThinkTankPlannerInstruction_RequiresRedisKnowledgeBaseRetrieval(t *testing.T) {
	if !strings.Contains(thinkTankPlannerInstruction, "检索 Redis 知识库") {
		t.Fatalf("planner instruction must ask plans to include Redis knowledge-base retrieval")
	}
	if !strings.Contains(thinkTankPlannerInstruction, "LocalSearch") {
		t.Fatalf("planner instruction must bind Redis knowledge-base retrieval to LocalSearch")
	}
}

func TestThinkTankReplannerInstruction_RequiresRedisBeforeWebResearch(t *testing.T) {
	if !strings.Contains(thinkTankReplannerInstruction, "LocalSearch has not been executed yet") {
		t.Fatalf("replanner instruction must require LocalSearch before external research")
	}
	if !strings.Contains(thinkTankReplannerInstruction, "检索 Redis 知识库") {
		t.Fatalf("replanner instruction must preserve Redis knowledge-base retrieval wording")
	}
}

func TestThinkTankReplannerInstruction_DeliversFinalArtifactInsteadOfProcessSummary(t *testing.T) {
	required := []string{
		"deliver the requested artifact directly",
		"Do not make the final response a process summary",
		"调研报告",
		"已完成",
		"DocWriter",
	}
	for _, text := range required {
		if !strings.Contains(thinkTankReplannerInstruction, text) {
			t.Fatalf("replanner instruction must contain %q", text)
		}
	}
}

func TestThinkTankReplannerInstruction_UsesEvidenceInsteadOfMissingToolComplaint(t *testing.T) {
	required := []string{
		"Do not answer by saying a tool is missing",
		"DocParser",
		"RespondTool",
		"available evidence",
	}
	for _, text := range required {
		if !strings.Contains(thinkTankReplannerInstruction, text) {
			t.Fatalf("replanner instruction must contain %q", text)
		}
	}
}

func TestWebFetchToolRejectsNonHTTPURLWithoutNetworkAttempt(t *testing.T) {
	fetchTool, err := newWebFetchTool(ResearchConfig{TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("expected WebFetch tool to be created, got %v", err)
	}
	invokable, ok := fetchTool.(tool.InvokableTool)
	if !ok {
		t.Fatalf("expected WebFetch tool to be invokable")
	}

	output, err := invokable.InvokableRun(context.Background(), `{"url":"由于上一步未明确给出Reddit、Scribd等平台具体相关网页的URL，此处无法准确填写"}`)
	if err != nil {
		t.Fatalf("expected invalid URL to be returned as tool output, got error %v", err)
	}
	if !strings.Contains(output, "有效的 http(s) URL") || !strings.Contains(output, "不是 URL") {
		t.Fatalf("expected invalid URL guidance, got %q", output)
	}
}

func TestExtractADKClarificationQuestion_PrefersRootCauseInfo(t *testing.T) {
	info := &adk.InterruptInfo{
		Data: map[string]any{
			"OrigInput": map[string]any{
				"Messages": []any{map[string]any{"content": "我要学习"}},
			},
			"Data": "checkpoint-internal-state",
		},
		InterruptContexts: []*adk.InterruptCtx{
			{
				Info:        map[string]any{"ToolCalls": []any{"internal"}},
				IsRootCause: false,
			},
			{
				Info:        "请问您想学习什么内容？",
				IsRootCause: true,
			},
		},
	}

	got := extractADKClarificationQuestion(info)
	if got != "请问您想学习什么内容？" {
		t.Fatalf("expected root-cause clarification question, got %q", got)
	}
	if strings.Contains(got, "OrigInput") || strings.Contains(got, "checkpoint") {
		t.Fatalf("expected user-facing question only, got internal payload %q", got)
	}
}

func TestExtractPlanExecuteFinalResponse_ReturnsRespondContent(t *testing.T) {
	got, ok := extractPlanExecuteFinalResponse(`{"response":"这是最终学习计划"}`)
	if !ok {
		t.Fatalf("expected respond payload to be treated as final answer")
	}
	if got != "这是最终学习计划" {
		t.Fatalf("expected response text, got %q", got)
	}
}

func TestExtractPlanExecuteFinalResponse_IgnoresPlanPayload(t *testing.T) {
	_, ok := extractPlanExecuteFinalResponse(`{"steps":["继续搜索资源","制定学习计划"]}`)
	if ok {
		t.Fatalf("expected replanner plan payload to be ignored as non-final output")
	}
}

func TestIsNonFinalToolLimitationAnswer_DetectsMissingDocParserComplaint(t *testing.T) {
	if !isNonFinalToolLimitationAnswer("当前工具列表中无 DocParser 工具，无法完成解析 HTML 内容的操作，请提供其他可行的工具或解决方案。") {
		t.Fatalf("expected missing DocParser complaint to be treated as non-final")
	}
	if isNonFinalToolLimitationAnswer("### 李小龙调研报告\n\n李小龙是截拳道创始人。") {
		t.Fatalf("expected normal report to be treated as final")
	}
}

func TestComposeADKFallbackAnswer_UsesCollectedEvidence(t *testing.T) {
	svc := &thinkTankService{}

	answer, err := svc.composeADKFallbackAnswer(
		context.Background(),
		"我想要学习 AI",
		[]string{"站内资料说明了 2026 年 AI 技术趋势。"},
		[]string{"NVIDIA AI 基础学习：提供生成式 AI、RAG 等课程。"},
		nil,
		[]SourceRef{{Kind: "web", Title: "NVIDIA AI 基础学习", URL: "https://www.nvidia.cn/learn/ai-learning-essentials/"}},
	)
	if err != nil {
		t.Fatalf("expected fallback answer, got error %v", err)
	}
	if !strings.Contains(answer, "基于当前已完成的检索结果") {
		t.Fatalf("expected generated fallback answer, got %q", answer)
	}
	if !strings.Contains(answer, "NVIDIA AI 基础学习") {
		t.Fatalf("expected fallback answer to keep references, got %q", answer)
	}
}

func TestAppendNonEmptyNote_SkipsRawHTML(t *testing.T) {
	notes := appendNonEmptyNote(nil, "<!doctype html><html><script>alert(1)</script></html>")
	if len(notes) != 0 {
		t.Fatalf("expected raw HTML note to be skipped, got %#v", notes)
	}
}
