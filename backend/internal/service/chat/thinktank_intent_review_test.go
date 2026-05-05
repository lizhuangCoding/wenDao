package chat

import (
	"strings"
	"testing"
)

func TestParseClarifierDecision_UsesModelJSON(t *testing.T) {
	raw := `模型输出:
{
  "normalized_question": "分析 AI Agent 的发展趋势",
  "intent": "了解 AI Agent 的发展趋势",
  "answer_goal": "research",
  "target_dimensions": ["技术演进", "商业落地"],
  "constraints": {"time_range": "未来三年", "audience": "创业者", "depth": "深入", "style": "结构化", "source_policy": "优先引用来源"},
  "ambiguity_level": "low",
  "should_ask_user": false,
  "clarification_question": "",
  "reason": "对象明确，维度可自动推断"
}`

	got := parseClarifierDecision(raw, "帮我分析一下 AI Agent 的发展趋势")
	if got.ShouldAskUser {
		t.Fatalf("expected broad but clear question to continue, got %#v", got)
	}
	if got.NormalizedQuestion != "分析 AI Agent 的发展趋势" {
		t.Fatalf("unexpected normalized question %q", got.NormalizedQuestion)
	}
	if len(got.TargetDimensions) != 2 || got.TargetDimensions[0] != "技术演进" {
		t.Fatalf("unexpected target dimensions %#v", got.TargetDimensions)
	}
	if got.Constraints.Audience != "创业者" {
		t.Fatalf("expected constraints to be parsed, got %#v", got.Constraints)
	}
}

func TestParseClarifierDecision_DefaultsWhenJSONInvalid(t *testing.T) {
	got := parseClarifierDecision("not json", "Redis 和 MySQL 有什么区别？")
	if got.ShouldAskUser {
		t.Fatalf("invalid clarifier output should fall back to continuing, got %#v", got)
	}
	if got.NormalizedQuestion != "Redis 和 MySQL 有什么区别？" {
		t.Fatalf("expected original question fallback, got %q", got.NormalizedQuestion)
	}
	if got.AnswerGoal != "explain" {
		t.Fatalf("expected explain fallback, got %q", got.AnswerGoal)
	}
}

func TestParseAcceptanceReview_NormalizesVerdict(t *testing.T) {
	raw := `{"verdict":"revise","score":62,"matched_dimensions":["技术演进"],"missing_dimensions":["商业落地"],"unsupported_claims":["缺少来源"],"format_issues":["没有结论"],"revision_instruction":"补充商业落地和明确趋势判断","user_question":"","reason":"缺少关键维度"}`
	got := parseAcceptanceReview(raw)
	if got.Verdict != acceptanceVerdictRevise {
		t.Fatalf("expected revise verdict, got %#v", got)
	}
	if got.Score != 62 {
		t.Fatalf("expected score 62, got %d", got.Score)
	}
	if got.RevisionInstruction != "补充商业落地和明确趋势判断" {
		t.Fatalf("unexpected revision instruction %q", got.RevisionInstruction)
	}
}

func TestParseAcceptanceReview_DefaultsToPassWhenInvalid(t *testing.T) {
	got := parseAcceptanceReview("not json")
	if got.Verdict != acceptanceVerdictPass {
		t.Fatalf("invalid acceptance output should not block answer, got %#v", got)
	}
	if got.Score != 100 {
		t.Fatalf("expected safe pass score, got %d", got.Score)
	}
}

func TestBuildClarifiedAgentQuery_IncludesIntentAndDimensions(t *testing.T) {
	decision := ClarifierDecision{
		NormalizedQuestion: "分析 AI Agent 的发展趋势",
		Intent:             "了解 AI Agent 的发展趋势",
		AnswerGoal:         "research",
		TargetDimensions:   []string{"技术演进", "商业落地"},
	}
	got := buildClarifiedAgentQuery("原始上下文", decision)
	for _, want := range []string{"原始上下文", "分析 AI Agent 的发展趋势", "技术演进", "商业落地"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected clarified query to contain %q, got %q", want, got)
		}
	}
}

func TestBuildRevisionAgentQuery_IncludesReviewInstruction(t *testing.T) {
	review := AcceptanceReview{
		Verdict:             acceptanceVerdictRevise,
		MissingDimensions:   []string{"风险限制"},
		RevisionInstruction: "补充风险限制和落地条件",
	}
	got := buildRevisionAgentQuery("原始上下文", "第一版答案", review)
	for _, want := range []string{"原始上下文", "第一版答案", "风险限制", "补充风险限制和落地条件"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected revision query to contain %q, got %q", want, got)
		}
	}
}
