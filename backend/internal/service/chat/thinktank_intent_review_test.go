package chat

import (
	"encoding/json"
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

func TestParseClarifierDecision_IncludesVisibleNeedProfile(t *testing.T) {
	raw := `{
  "normalized_question": "制定 AI 学习计划",
  "intent": "学习 AI",
  "answer_goal": "learning_plan",
  "target_dimensions": ["学习路径", "练习项目"],
  "constraints": {"depth": "入门"},
  "ambiguity_level": "medium",
  "should_ask_user": true,
  "clarification_question": "你想学哪个领域？",
  "reason": "学习计划需要明确基础和目标",
  "need_summary": "制定一个可执行的学习计划",
  "missing_dimensions": ["学习领域", "当前基础", "学习目标"],
  "why_needed": "不同领域、基础和目标会影响学习路径与练习项目。",
  "suggested_reply": "我想学 AI，目前零基础，希望三个月能做一个小项目。"
}`

	got := parseClarifierDecision(raw, "我要学习知识")
	if got.NeedSummary != "制定一个可执行的学习计划" {
		t.Fatalf("expected need summary to be parsed, got %q", got.NeedSummary)
	}
	if len(got.MissingDimensions) != 3 || got.MissingDimensions[0] != "学习领域" {
		t.Fatalf("expected missing dimensions to be parsed, got %#v", got.MissingDimensions)
	}
	if got.WhyNeeded != "不同领域、基础和目标会影响学习路径与练习项目。" {
		t.Fatalf("expected why_needed to be parsed, got %q", got.WhyNeeded)
	}
	if got.SuggestedReply != "我想学 AI，目前零基础，希望三个月能做一个小项目。" {
		t.Fatalf("expected suggested reply to be parsed, got %q", got.SuggestedReply)
	}
}

func TestParseClarifierDecision_KeepsAskUserWhenVisibleProfileCanRenderQuestion(t *testing.T) {
	raw := `{
  "normalized_question": "制定 AI 学习计划",
  "intent": "学习 AI",
  "answer_goal": "learning_plan",
  "target_dimensions": ["学习路径"],
  "constraints": {"depth": "入门"},
  "ambiguity_level": "medium",
  "should_ask_user": true,
  "clarification_question": "",
  "reason": "学习计划需要明确基础和目标",
  "need_summary": "制定一个可执行的学习计划",
  "missing_dimensions": ["学习领域", "当前基础", "学习目标"],
  "why_needed": "不同领域、基础和目标会影响学习路径与练习项目。",
  "suggested_reply": "我想学 AI，目前零基础，希望三个月能做一个小项目。"
}`

	got := parseClarifierDecision(raw, "我要学习知识")
	if !got.ShouldAskUser {
		t.Fatalf("expected visible profile to preserve ask-user path, got %#v", got)
	}
	formatted := formatClarifierQuestion(got)
	for _, want := range []string{
		"我理解你是想：制定一个可执行的学习计划",
		"1. 学习领域",
		"为什么需要这些信息：",
		"你可以这样回复：",
		"我想学 AI，目前零基础",
	} {
		if !strings.Contains(formatted, want) {
			t.Fatalf("expected formatted clarification to contain %q, got %q", want, formatted)
		}
	}
}

func TestParseClarifierDecision_DisablesAskUserWithoutConcreteMissingDimension(t *testing.T) {
	raw := `{
  "normalized_question": "制定 AI 学习计划",
  "intent": "学习 AI",
  "answer_goal": "learning_plan",
  "target_dimensions": ["学习路径"],
  "constraints": {"depth": "入门"},
  "ambiguity_level": "medium",
  "should_ask_user": true,
  "clarification_question": "",
  "reason": "学习计划需要明确基础和目标",
  "need_summary": "制定一个可执行的学习计划",
  "missing_dimensions": [],
  "why_needed": "不同领域、基础和目标会影响学习路径与练习项目。",
  "suggested_reply": "我想学 AI，目前零基础，希望三个月能做一个小项目。"
}`

	got := parseClarifierDecision(raw, "我要学习知识")
	if got.ShouldAskUser {
		t.Fatalf("expected ask-user path to be disabled without concrete missing dimension, got %#v", got)
	}
}

func TestFormatClarifierQuestion_ShowsNeedMissingReasonAndSuggestedReply(t *testing.T) {
	decision := ClarifierDecision{
		NormalizedQuestion: "我要学习知识",
		Intent:             "学习知识",
		AnswerGoal:         "learning_plan",
		ShouldAskUser:      true,
		NeedSummary:        "制定一个可执行的学习计划",
		MissingDimensions:  []string{"学习领域", "当前基础", "学习目标"},
		WhyNeeded:          "不同领域、基础和目标会影响学习路径与练习项目。",
		SuggestedReply:     "我想学 AI，目前零基础，希望三个月能做一个小项目。",
	}

	got := formatClarifierQuestion(decision)
	for _, want := range []string{
		"我理解你是想：制定一个可执行的学习计划",
		"为了后续回答更精确，需要确认：",
		"1. 学习领域",
		"2. 当前基础",
		"3. 学习目标",
		"为什么需要这些信息：",
		"你可以这样回复：",
		"我想学 AI，目前零基础",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected formatted clarification to contain %q, got %q", want, got)
		}
	}
}

func TestFormatClarifierQuestion_FillsMissingVisibleProfile(t *testing.T) {
	decision := ClarifierDecision{
		NormalizedQuestion: "我要学习知识",
		Intent:             "学习知识",
		ShouldAskUser:      true,
		NeedSummary:        "制定一个可执行的学习计划",
		MissingDimensions:  []string{"学习领域", "当前基础"},
	}

	got := formatClarifierQuestion(decision)
	for _, want := range []string{
		"我理解你是想：制定一个可执行的学习计划",
		"1. 学习领域",
		"2. 当前基础",
		"为什么需要这些信息：",
		"这些信息会直接影响回答范围、深度和可执行建议",
		"你可以这样回复：",
		"请按上述维度逐项补充，例如：学习领域、当前基础。",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected fallback clarification to contain %q, got %q", want, got)
		}
	}
}

func TestFormatClarifierQuestion_StepDetailUsesConciseProcessPanel(t *testing.T) {
	decision := ClarifierDecision{
		NormalizedQuestion: "我要学习知识",
		Intent:             "学习知识",
		AnswerGoal:         "learning_plan",
		TargetDimensions:   []string{"学习路径", "练习项目"},
		ShouldAskUser:      true,
		NeedSummary:        "制定一个可执行的学习计划",
		MissingDimensions:  []string{"学习领域", "当前基础"},
		WhyNeeded:          "不同领域、基础和目标会影响学习路径与练习项目。",
		SuggestedReply:     "我想学 AI，目前零基础。",
		Reason:             "学习计划需要明确基础和目标",
	}

	got := formatClarifierStepDetail(decision)
	for _, want := range []string{
		"实际需求：制定一个可执行的学习计划",
		"回答维度：学习路径、练习项目",
		"处理方式：需要用户补充关键维度",
		"原因：学习计划需要明确基础和目标",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected step detail to contain %q, got %q", want, got)
		}
	}
	for _, notWant := range []string{
		"我理解你是想：",
		"为了后续回答更精确，需要确认：",
		"你可以这样回复：",
	} {
		if strings.Contains(got, notWant) {
			t.Fatalf("expected step detail to omit user-facing text %q, got %q", notWant, got)
		}
	}
}

func TestParseAcceptanceReview_NormalizesVerdict(t *testing.T) {
	raw := `{"verdict":"revise","score":62,"matched_dimensions":["技术演进"],"missing_dimensions":["商业落地"],"unsupported_claims":["缺少来源"],"format_issues":["没有结论"],"revision_instruction":"补充商业落地和明确趋势判断","user_question":"","reason":"缺少关键维度","summary":"  需要补充商业化分析  "}`
	got := parseAcceptanceReview(raw)
	if got.Verdict != acceptanceVerdictRevise {
		t.Fatalf("expected revise verdict, got %#v", got)
	}
	if !got.Available {
		t.Fatalf("expected parsed acceptance review to be available, got %#v", got)
	}
	if got.Score != 62 {
		t.Fatalf("expected score 62, got %d", got.Score)
	}
	if got.RevisionInstruction != "补充商业落地和明确趋势判断" {
		t.Fatalf("unexpected revision instruction %q", got.RevisionInstruction)
	}
	if got.Summary != "需要补充商业化分析" {
		t.Fatalf("expected trimmed summary, got %q", got.Summary)
	}
}

func TestNormalizeAcceptanceVerdict_AcceptsCaseAndHyphenVariants(t *testing.T) {
	tests := map[string]string{
		"Revise":   acceptanceVerdictRevise,
		"ASK_USER": acceptanceVerdictAskUser,
		"ask-user": acceptanceVerdictAskUser,
	}
	for verdict, want := range tests {
		if got := normalizeAcceptanceVerdict(verdict); got != want {
			t.Fatalf("expected %q to normalize to %q, got %q", verdict, want, got)
		}
	}
}

func TestParseAcceptanceReview_DefaultsToPassWhenInvalid(t *testing.T) {
	got := parseAcceptanceReview("not json")
	if got.Verdict != acceptanceVerdictPass {
		t.Fatalf("invalid acceptance output should not block answer, got %#v", got)
	}
	if got.Score != 0 {
		t.Fatalf("expected unavailable pass score, got %d", got.Score)
	}
	if got.Available {
		t.Fatalf("expected invalid acceptance output to be unavailable, got %#v", got)
	}
}

func TestParseAcceptanceReview_EmptyObjectIsUnavailable(t *testing.T) {
	got := parseAcceptanceReview(`{}`)
	if got.Available {
		t.Fatalf("expected empty acceptance object to be unavailable, got %#v", got)
	}
	if got.Score != 0 {
		t.Fatalf("expected unavailable empty object score 0, got %d", got.Score)
	}
}

func TestParseAcceptanceReview_SummaryOnlyIsUnavailable(t *testing.T) {
	got := parseAcceptanceReview(`{"summary":"ok"}`)
	if got.Available {
		t.Fatalf("expected summary-only acceptance review to be unavailable, got %#v", got)
	}
	if got.Verdict != acceptanceVerdictPass || got.Score != 0 {
		t.Fatalf("expected unavailable default review, got %#v", got)
	}
}

func TestParseAcceptanceReview_AskUserWithoutQuestionIsUnavailable(t *testing.T) {
	got := parseAcceptanceReview(`{"verdict":"ask_user","summary":"need scope","score":60}`)
	if got.Available {
		t.Fatalf("expected ask_user without follow-up question to be unavailable, got %#v", got)
	}
	if got.Verdict != acceptanceVerdictPass || got.Score != 0 {
		t.Fatalf("expected unavailable default review, got %#v", got)
	}
}

func TestParseAcceptanceReview_AskUserAcceptsFollowUpQuestion(t *testing.T) {
	got := parseAcceptanceReview(`{"verdict":"ask_user","summary":"need scope","score":60,"follow_up_question":"请确认市场范围？"}`)
	if !got.Available {
		t.Fatalf("expected ask_user with follow-up question to be available, got %#v", got)
	}
	if got.UserQuestion != "请确认市场范围？" {
		t.Fatalf("expected follow-up question to populate user question, got %#v", got)
	}
}

func TestParseAcceptanceReview_ClampsScoreAndPreservesExplicitZero(t *testing.T) {
	tooHigh := parseAcceptanceReview(`{"verdict":"pass","score":120,"summary":"覆盖核心问题"}`)
	if !tooHigh.Available {
		t.Fatalf("expected meaningful acceptance output to be available, got %#v", tooHigh)
	}
	if tooHigh.Score != 100 {
		t.Fatalf("expected score to be clamped to 100, got %d", tooHigh.Score)
	}

	explicitZero := parseAcceptanceReview(`{"verdict":"pass","score":0,"summary":"无法确认质量","reason":"模型明确给出零分"}`)
	if !explicitZero.Available {
		t.Fatalf("expected explicit zero with summary/reason to be available, got %#v", explicitZero)
	}
	if explicitZero.Score != 0 {
		t.Fatalf("expected explicit score 0 to be preserved, got %d", explicitZero.Score)
	}
}

func TestAppendAcceptanceSummary_PassShowsScoreAndCoveredDimensions(t *testing.T) {
	review := AcceptanceReview{
		Verdict:           acceptanceVerdictPass,
		Score:             88,
		MatchedDimensions: []string{"技术演进", "产品形态", "风险限制"},
		Reason:            "回答覆盖主要维度，证据边界清楚。",
		Summary:           "初稿已经满足用户的核心分析需求。",
		Available:         true,
	}

	got := appendAcceptanceSummary("最终答案", review, false)
	for _, want := range []string{
		"最终答案",
		"验收摘要：通过，评分 88/100",
		"已覆盖：技术演进、产品形态、风险限制",
		"初稿已经满足用户的核心分析需求。",
		"回答覆盖主要维度，证据边界清楚。",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected acceptance summary to contain %q, got %q", want, got)
		}
	}
}

func TestAppendAcceptanceSummary_ShowsDimensionFallbacks(t *testing.T) {
	review := AcceptanceReview{
		Verdict:   acceptanceVerdictPass,
		Score:     90,
		Summary:   "答案整体满足问题。",
		Available: true,
	}

	got := appendAcceptanceSummary("最终答案", review, false)
	for _, want := range []string{
		"验收摘要：通过，评分 90/100",
		"已覆盖：AcceptanceAgent 未返回具体覆盖维度",
		"仍需注意：未发现明确缺失维度",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected acceptance summary fallback to contain %q, got %q", want, got)
		}
	}
}

func TestAppendAcceptanceSummary_RevisedShowsAutomaticRevision(t *testing.T) {
	review := AcceptanceReview{
		Verdict:             acceptanceVerdictRevise,
		Score:               72,
		RevisionInstruction: "补充风险限制和落地条件",
		Summary:             "初稿缺少关键风险维度。",
		Available:           true,
	}

	got := appendAcceptanceSummary("修订后答案", review, true)
	for _, want := range []string{
		"验收摘要：初稿需要修订，已自动补充关键缺失项，评分 72/100",
		"修订重点：补充风险限制和落地条件",
		"修订后答案",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected revised acceptance summary to contain %q, got %q", want, got)
		}
	}
}

func TestAppendAcceptanceSummary_SkipsUnavailableReview(t *testing.T) {
	got := appendAcceptanceSummary("最终答案", defaultAcceptanceReview(), false)
	if got != "最终答案" {
		t.Fatalf("expected unavailable review to leave answer unchanged, got %q", got)
	}
}

func TestFormatAcceptanceQuestion_ShowsReasonAndQuestion(t *testing.T) {
	review := AcceptanceReview{
		Verdict:      acceptanceVerdictAskUser,
		UserQuestion: "你希望重点分析国内市场还是全球市场？",
		Reason:       "市场范围会改变案例、风险和趋势判断。",
		Available:    true,
	}

	got := formatAcceptanceQuestion(review)
	for _, want := range []string{
		"验收时发现还缺少一个关键信息：",
		"你希望重点分析国内市场还是全球市场？",
		"为什么需要：",
		"市场范围会改变案例、风险和趋势判断。",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected acceptance question to contain %q, got %q", want, got)
		}
	}
}

func TestFormatAcceptanceStepDetail_ShowsVerdictScoreAndHandling(t *testing.T) {
	passReview := AcceptanceReview{
		Verdict:           acceptanceVerdictPass,
		Score:             91,
		MatchedDimensions: []string{"技术演进", "产品形态"},
		Reason:            "覆盖核心维度。",
		Available:         true,
	}
	passDetail := formatAcceptanceStepDetail(passReview, false)
	for _, want := range []string{
		"验收结论：通过",
		"评分：91/100",
		"已覆盖：技术演进、产品形态",
		"处理方式：验收通过",
		"原因：覆盖核心维度。",
	} {
		if !strings.Contains(passDetail, want) {
			t.Fatalf("expected pass step detail to contain %q, got %q", want, passDetail)
		}
	}

	reviseReview := AcceptanceReview{
		Verdict:           acceptanceVerdictRevise,
		Score:             64,
		MissingDimensions: []string{"风险限制"},
		Available:         true,
	}
	reviseDetail := formatAcceptanceStepDetail(reviseReview, true)
	for _, want := range []string{
		"验收结论：需要修订",
		"评分：64/100",
		"缺失维度：风险限制",
		"处理方式：初稿需要修订，已自动补充关键缺失项",
	} {
		if !strings.Contains(reviseDetail, want) {
			t.Fatalf("expected revised step detail to contain %q, got %q", want, reviseDetail)
		}
	}
}

func TestFormatAcceptanceStepDetail_ShowsDimensionFallbacks(t *testing.T) {
	review := AcceptanceReview{
		Verdict:   acceptanceVerdictPass,
		Score:     90,
		Summary:   "答案整体满足问题。",
		Available: true,
	}

	got := formatAcceptanceStepDetail(review, false)
	for _, want := range []string{
		"已覆盖：AcceptanceAgent 未返回具体覆盖维度",
		"缺失维度：未发现明确缺失维度",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected acceptance step fallback to contain %q, got %q", want, got)
		}
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

func TestClarifierInstruction_OnlyAsksForCriticalMissingInfo(t *testing.T) {
	required := []string{
		"Only ask the user when missing information would change what should be answered",
		"broad but clear",
		"target_dimensions",
		"valid JSON",
	}
	for _, text := range required {
		if !strings.Contains(thinkTankClarifierInstruction, text) {
			t.Fatalf("clarifier instruction must contain %q", text)
		}
	}
}

func TestAcceptanceInstruction_BoundsReviewStrictness(t *testing.T) {
	required := []string{
		"Return pass when the answer substantially satisfies",
		"revise only when",
		"ask_user only when",
		"score",
		"summary",
		"valid JSON",
	}
	for _, text := range required {
		if !strings.Contains(thinkTankAcceptanceInstruction, text) {
			t.Fatalf("acceptance instruction must contain %q", text)
		}
	}
}

func TestBuildClarifierPrompt_IncludesOriginalQuestionAndAgentQuery(t *testing.T) {
	got := buildClarifierPrompt(ClarifierInput{
		OriginalQuestion: "帮我调研一下李小龙",
		AgentQuery:       "会话记忆：用户喜欢结构化回答",
	})
	for _, want := range []string{"帮我调研一下李小龙", "会话记忆", "should_ask_user"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected clarifier prompt to contain %q, got %q", want, got)
		}
	}
}

func TestBuildAcceptancePrompt_IncludesReviewContext(t *testing.T) {
	got := buildAcceptancePrompt(AcceptanceReviewInput{
		OriginalQuestion: "帮我分析一下 AI Agent 的发展趋势",
		AgentQuery:       "用户关心维度：技术演进、商业落地",
		Decision: ClarifierDecision{
			TargetDimensions: []string{"技术演进", "商业落地"},
		},
		Answer:        "AI Agent 正在发展。",
		RevisionCount: 0,
	})
	for _, want := range []string{"AI Agent", "技术演进", "商业落地", "Revision count: 0", "verdict", "score", "summary"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected acceptance prompt to contain %q, got %q", want, got)
		}
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(got), &payload); err != nil {
		t.Fatalf("expected acceptance prompt to be valid JSON, got error %v and prompt %q", err, got)
	}
	instruction, ok := payload["instruction"].(string)
	if !ok {
		t.Fatalf("expected acceptance prompt to include instruction field, got %#v", payload)
	}
	for _, want := range []string{"Revision count: 0", "verdict", "score", "summary"} {
		if !strings.Contains(instruction, want) {
			t.Fatalf("expected instruction to contain %q, got %q", want, instruction)
		}
	}
}
