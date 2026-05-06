# ThinkTank Visible Clarifier Acceptance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make ClarifierAgent and AcceptanceAgent user-visible in a concise way: structured demand clarification before answering, and a final validation summary after answering.

**Architecture:** Keep the existing Eino ADK ClarifierAgent and AcceptanceAgent. Extend their structured outputs, add deterministic backend formatters, wire formatted messages into non-stream and stream flows, and emit synthetic process steps for ClarifierAgent / AcceptanceAgent so the existing frontend process panel can show them.

**Tech Stack:** Go backend, CloudWeGo Eino ADK, existing ThinkTank orchestrator, existing SSE stream events, React frontend with Markdown rendering.

---

### Task 1: Extend Review Types And Formatters

**Files:**
- Modify: `backend/internal/service/chat/thinktank_intent_review.go`
- Modify: `backend/internal/service/chat/thinktank_intent_review_agents.go`
- Test: `backend/internal/service/chat/thinktank_intent_review_test.go`

- [ ] **Step 1: Write failing tests for visible ClarifierAgent fields and formatting**

Add tests to `backend/internal/service/chat/thinktank_intent_review_test.go`:

```go
func TestParseClarifierDecision_IncludesVisibleNeedProfile(t *testing.T) {
	raw := `{
		"normalized_question":"制定学习计划",
		"intent":"用户想制定一个学习计划",
		"answer_goal":"tutorial",
		"target_dimensions":["学习领域","当前基础","学习目标"],
		"constraints":{"depth":"入门"},
		"ambiguity_level":"high",
		"should_ask_user":true,
		"clarification_question":"你想学习哪个领域？",
		"reason":"缺少学习领域",
		"need_summary":"制定一个可执行的学习计划",
		"missing_dimensions":["学习领域","当前基础","学习目标"],
		"why_needed":"不同领域、基础和目标会决定学习路径、资料难度和练习方式。",
		"suggested_reply":"我想学 AI，目前零基础，目标是能做一个小项目。"
	}`

	got := parseClarifierDecision(raw, "我要学习知识")

	if got.NeedSummary != "制定一个可执行的学习计划" {
		t.Fatalf("expected visible need summary, got %q", got.NeedSummary)
	}
	if strings.Join(got.MissingDimensions, "、") != "学习领域、当前基础、学习目标" {
		t.Fatalf("expected missing dimensions, got %#v", got.MissingDimensions)
	}
	if !strings.Contains(got.WhyNeeded, "学习路径") {
		t.Fatalf("expected why_needed, got %q", got.WhyNeeded)
	}
	if !strings.Contains(got.SuggestedReply, "我想学 AI") {
		t.Fatalf("expected suggested reply, got %q", got.SuggestedReply)
	}
}

func TestFormatClarifierQuestion_ShowsNeedMissingReasonAndSuggestedReply(t *testing.T) {
	decision := ClarifierDecision{
		NormalizedQuestion:    "制定学习计划",
		Intent:                "制定一个学习计划",
		AnswerGoal:            "tutorial",
		TargetDimensions:      []string{"学习领域", "当前基础", "学习目标"},
		ShouldAskUser:         true,
		ClarificationQuestion: "你想学习哪个领域？",
		NeedSummary:           "制定一个可执行的学习计划",
		MissingDimensions:     []string{"学习领域", "当前基础", "学习目标"},
		WhyNeeded:             "不同领域、基础和目标会决定学习路径、资料难度和练习方式。",
		SuggestedReply:        "我想学 AI，目前零基础，目标是能做一个小项目。",
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
			t.Fatalf("expected formatted clarifier question to contain %q, got:\n%s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
cd backend
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'Test(ParseClarifierDecision_IncludesVisibleNeedProfile|FormatClarifierQuestion_ShowsNeedMissingReasonAndSuggestedReply)'
```

Expected: FAIL because `ClarifierDecision.NeedSummary`, `MissingDimensions`, `WhyNeeded`, `SuggestedReply`, and `formatClarifierQuestion` do not exist yet.

- [ ] **Step 3: Extend ClarifierDecision and add formatter**

Modify `backend/internal/service/chat/thinktank_intent_review.go`:

```go
type ClarifierDecision struct {
	NormalizedQuestion    string               `json:"normalized_question"`
	Intent                string               `json:"intent"`
	AnswerGoal            string               `json:"answer_goal"`
	TargetDimensions      []string             `json:"target_dimensions"`
	Constraints           ClarifierConstraints `json:"constraints"`
	AmbiguityLevel        string               `json:"ambiguity_level"`
	ShouldAskUser         bool                 `json:"should_ask_user"`
	ClarificationQuestion string               `json:"clarification_question"`
	Reason                string               `json:"reason"`
	NeedSummary           string               `json:"need_summary"`
	MissingDimensions     []string             `json:"missing_dimensions"`
	WhyNeeded             string               `json:"why_needed"`
	SuggestedReply        string               `json:"suggested_reply"`
}
```

In `parseClarifierDecision`, after trimming `ClarificationQuestion`, add:

```go
decision.NeedSummary = strings.TrimSpace(decision.NeedSummary)
if decision.NeedSummary == "" {
	decision.NeedSummary = decision.Intent
}
decision.WhyNeeded = strings.TrimSpace(decision.WhyNeeded)
decision.SuggestedReply = strings.TrimSpace(decision.SuggestedReply)
if decision.ShouldAskUser && len(decision.MissingDimensions) == 0 && decision.ClarificationQuestion != "" {
	decision.MissingDimensions = []string{decision.ClarificationQuestion}
}
```

Add helpers near `buildClarifiedAgentQuery`:

```go
func formatClarifierQuestion(decision ClarifierDecision) string {
	summary := strings.TrimSpace(decision.NeedSummary)
	if summary == "" {
		summary = strings.TrimSpace(decision.Intent)
	}
	if summary == "" {
		summary = strings.TrimSpace(decision.NormalizedQuestion)
	}
	if summary == "" {
		summary = "进一步明确你的需求"
	}

	missing := compactNonEmptyStrings(decision.MissingDimensions)
	if len(missing) == 0 && strings.TrimSpace(decision.ClarificationQuestion) != "" {
		missing = []string{strings.TrimSpace(decision.ClarificationQuestion)}
	}

	var b strings.Builder
	b.WriteString("我理解你是想：")
	b.WriteString(summary)
	b.WriteString("\n\n为了后续回答更精确，需要确认：")
	for i, item := range missing {
		b.WriteString("\n")
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(". ")
		b.WriteString(item)
	}
	if len(missing) == 0 {
		b.WriteString("\n1. ")
		b.WriteString(strings.TrimSpace(decision.ClarificationQuestion))
	}
	if strings.TrimSpace(decision.WhyNeeded) != "" {
		b.WriteString("\n\n为什么需要这些信息：\n")
		b.WriteString(strings.TrimSpace(decision.WhyNeeded))
	}
	if strings.TrimSpace(decision.SuggestedReply) != "" {
		b.WriteString("\n\n你可以这样回复：\n")
		b.WriteString(strings.TrimSpace(decision.SuggestedReply))
	}
	return strings.TrimSpace(b.String())
}

func formatClarifierStepDetail(decision ClarifierDecision) string {
	var b strings.Builder
	b.WriteString("实际需求：")
	if strings.TrimSpace(decision.NeedSummary) != "" {
		b.WriteString(strings.TrimSpace(decision.NeedSummary))
	} else {
		b.WriteString(strings.TrimSpace(decision.Intent))
	}
	if len(decision.TargetDimensions) > 0 {
		b.WriteString("\n回答维度：")
		b.WriteString(strings.Join(compactNonEmptyStrings(decision.TargetDimensions), "、"))
	}
	if decision.ShouldAskUser {
		b.WriteString("\n处理方式：需要用户补充关键维度")
	} else {
		b.WriteString("\n处理方式：无需追问，按推断维度继续调研")
	}
	if strings.TrimSpace(decision.Reason) != "" {
		b.WriteString("\n原因：")
		b.WriteString(strings.TrimSpace(decision.Reason))
	}
	return strings.TrimSpace(b.String())
}

func compactNonEmptyStrings(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}
```

Add `strconv` to imports in `thinktank_intent_review.go`.

- [ ] **Step 4: Update ClarifierAgent prompt schema**

Modify `thinkTankClarifierInstruction` in `backend/internal/service/chat/thinktank_intent_review_agents.go` so it includes the new keys:

```go
const thinkTankClarifierInstruction = `You are the ThinkTank Clarifier.
Your job is to understand the user's real need before downstream agents answer.
Only ask the user when missing information would change what should be answered.
Do not ask follow-up questions for broad but clear research, explanation, comparison, or writing requests.
Infer reasonable target_dimensions, answer_goal, and constraints from the original question and context.
When should_ask_user is true, include a concise need_summary, 2-4 missing_dimensions, why_needed, and a suggested_reply template.
Return valid JSON only with keys: normalized_question, intent, answer_goal, target_dimensions, constraints, ambiguity_level, should_ask_user, clarification_question, reason, need_summary, missing_dimensions, why_needed, suggested_reply.`
```

In `buildClarifierPrompt`, update the policy:

```go
"policy": map[string]any{
	"should_ask_user": "only when missing information would change what should be answered",
	"visible_clarification": "when asking the user, explain understood need, missing dimensions, why they matter, and a suggested reply",
	"output": "Return valid JSON.",
},
```

- [ ] **Step 5: Run tests and commit**

Run:

```bash
cd backend
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'Test(ParseClarifierDecision_IncludesVisibleNeedProfile|FormatClarifierQuestion_ShowsNeedMissingReasonAndSuggestedReply|ClarifierInstruction)'
```

Expected: PASS.

Commit:

```bash
git add backend/internal/service/chat/thinktank_intent_review.go backend/internal/service/chat/thinktank_intent_review_agents.go backend/internal/service/chat/thinktank_intent_review_test.go
git commit -m "feat: add visible clarifier profile"
```

### Task 2: Add Acceptance Summary Formatting

**Files:**
- Modify: `backend/internal/service/chat/thinktank_intent_review.go`
- Modify: `backend/internal/service/chat/thinktank_intent_review_agents.go`
- Test: `backend/internal/service/chat/thinktank_intent_review_test.go`

- [ ] **Step 1: Write failing tests for acceptance summary**

Add tests:

```go
func TestAppendAcceptanceSummary_PassShowsScoreAndCoveredDimensions(t *testing.T) {
	review := AcceptanceReview{
		Verdict:           acceptanceVerdictPass,
		Score:             88,
		MatchedDimensions: []string{"技术演进", "产品形态", "风险限制"},
		Reason:            "覆盖主要问题",
		Summary:           "回答覆盖了用户关注的主要维度。",
		Available:         true,
	}

	got := appendAcceptanceSummary("最终答案", review, false)

	for _, want := range []string{"最终答案", "验收摘要：通过，评分 88/100", "已覆盖：技术演进、产品形态、风险限制", "回答覆盖了用户关注的主要维度。"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected acceptance summary to contain %q, got:\n%s", want, got)
		}
	}
}

func TestAppendAcceptanceSummary_SkipsUnavailableReview(t *testing.T) {
	got := appendAcceptanceSummary("最终答案", defaultAcceptanceReview(), false)
	if got != "最终答案" {
		t.Fatalf("expected unavailable review to skip visible score, got %q", got)
	}
}

func TestFormatAcceptanceQuestion_ShowsReasonAndQuestion(t *testing.T) {
	review := AcceptanceReview{
		Verdict:      acceptanceVerdictAskUser,
		UserQuestion: "你每天大概能投入多长时间学习？",
		Reason:       "学习计划节奏依赖时间投入。",
		Available:    true,
	}

	got := formatAcceptanceQuestion(review)

	for _, want := range []string{"验收时发现还缺少一个关键信息：", "你每天大概能投入多长时间学习？", "为什么需要：", "学习计划节奏依赖时间投入。"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected acceptance question to contain %q, got:\n%s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
cd backend
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'Test(AppendAcceptanceSummary|FormatAcceptanceQuestion)'
```

Expected: FAIL because `Summary`, `Available`, `appendAcceptanceSummary`, and `formatAcceptanceQuestion` do not exist yet.

- [ ] **Step 3: Extend AcceptanceReview and parser**

Modify `AcceptanceReview`:

```go
type AcceptanceReview struct {
	Verdict             string   `json:"verdict"`
	Score               int      `json:"score"`
	MatchedDimensions   []string `json:"matched_dimensions"`
	MissingDimensions   []string `json:"missing_dimensions"`
	UnsupportedClaims   []string `json:"unsupported_claims"`
	FormatIssues        []string `json:"format_issues"`
	RevisionInstruction string   `json:"revision_instruction"`
	UserQuestion        string   `json:"user_question"`
	Reason              string   `json:"reason"`
	Summary             string   `json:"summary"`
	Available           bool     `json:"-"`
}
```

In `parseAcceptanceReview`, after successful JSON unmarshal:

```go
review.Available = true
review.Summary = strings.TrimSpace(review.Summary)
```

Update `defaultAcceptanceReview`:

```go
func defaultAcceptanceReview() AcceptanceReview {
	return AcceptanceReview{
		Verdict:   acceptanceVerdictPass,
		Score:     0,
		Reason:    "acceptance output unavailable; returning generated answer",
		Available: false,
	}
}
```

Update any tests that expected default score `100` to expect `0` and unavailable.

- [ ] **Step 4: Add acceptance formatters**

Add near `appendAcceptanceLimitations`:

```go
func appendAcceptanceSummary(answer string, review AcceptanceReview, revised bool) string {
	answer = strings.TrimSpace(answer)
	if answer == "" || !review.Available {
		return answer
	}

	verdict := "通过"
	if revised {
		verdict = "初稿需要修订，已自动补充关键缺失项"
	}
	score := review.Score
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	lines := []string{fmt.Sprintf("验收摘要：%s，评分 %d/100", verdict, score)}
	if len(compactNonEmptyStrings(review.MatchedDimensions)) > 0 {
		lines = append(lines, "已覆盖："+strings.Join(compactNonEmptyStrings(review.MatchedDimensions), "、"))
	}
	if revised && strings.TrimSpace(review.RevisionInstruction) != "" {
		lines = append(lines, "修订重点："+strings.TrimSpace(review.RevisionInstruction))
	}
	if strings.TrimSpace(review.Summary) != "" {
		lines = append(lines, "结论："+strings.TrimSpace(review.Summary))
	} else if strings.TrimSpace(review.Reason) != "" {
		lines = append(lines, "结论："+strings.TrimSpace(review.Reason))
	}
	if len(compactNonEmptyStrings(review.MissingDimensions)) > 0 {
		lines = append(lines, "仍需注意："+strings.Join(compactNonEmptyStrings(review.MissingDimensions), "、"))
	}
	if len(compactNonEmptyStrings(review.UnsupportedClaims)) > 0 {
		lines = append(lines, "证据限制："+strings.Join(compactNonEmptyStrings(review.UnsupportedClaims), "、"))
	}

	return answer + "\n\n" + strings.Join(lines, "\n")
}

func formatAcceptanceQuestion(review AcceptanceReview) string {
	question := strings.TrimSpace(review.UserQuestion)
	if question == "" {
		question = "还需要你补充一点信息，我才能继续。"
	}
	var b strings.Builder
	b.WriteString("验收时发现还缺少一个关键信息：\n")
	b.WriteString(question)
	if strings.TrimSpace(review.Reason) != "" {
		b.WriteString("\n\n为什么需要：\n")
		b.WriteString(strings.TrimSpace(review.Reason))
	}
	return strings.TrimSpace(b.String())
}

func formatAcceptanceStepDetail(review AcceptanceReview, revised bool) string {
	if !review.Available {
		return "AcceptanceAgent 未返回可用验收结果，已直接返回生成答案。"
	}
	var b strings.Builder
	b.WriteString("验收结论：")
	b.WriteString(normalizeAcceptanceVerdict(review.Verdict))
	b.WriteString("\n评分：")
	b.WriteString(strconv.Itoa(review.Score))
	b.WriteString("/100")
	if len(compactNonEmptyStrings(review.MatchedDimensions)) > 0 {
		b.WriteString("\n已覆盖：")
		b.WriteString(strings.Join(compactNonEmptyStrings(review.MatchedDimensions), "、"))
	}
	if len(compactNonEmptyStrings(review.MissingDimensions)) > 0 {
		b.WriteString("\n缺失维度：")
		b.WriteString(strings.Join(compactNonEmptyStrings(review.MissingDimensions), "、"))
	}
	if revised {
		b.WriteString("\n处理方式：已自动修订一次")
	}
	if strings.TrimSpace(review.Reason) != "" {
		b.WriteString("\n原因：")
		b.WriteString(strings.TrimSpace(review.Reason))
	}
	return strings.TrimSpace(b.String())
}
```

Add `fmt` and `strconv` imports if not already present.

- [ ] **Step 5: Update AcceptanceAgent prompt schema**

Modify `thinkTankAcceptanceInstruction`:

```go
const thinkTankAcceptanceInstruction = `You are the ThinkTank Acceptance Reviewer.
Review whether the answer satisfies the user's original question and the clarified target dimensions.
Return pass when the answer substantially satisfies the user's original question and clarified target dimensions.
Return revise only when important requested dimensions, evidence, or answer structure are missing and can be fixed without asking the user.
Return ask_user only when the answer cannot proceed because critical user intent or constraints are still unknown.
Provide a 0-100 score and a concise user-facing summary.
Return valid JSON only with keys: verdict, score, matched_dimensions, missing_dimensions, unsupported_claims, format_issues, revision_instruction, user_question, reason, summary.`
```

In `buildAcceptancePrompt`, update instruction:

```go
"instruction": fmt.Sprintf("Revision count: %d. Return a valid JSON object with verdict, score, and summary.", input.RevisionCount),
```

- [ ] **Step 6: Run tests and commit**

Run:

```bash
cd backend
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'Test(AppendAcceptanceSummary|FormatAcceptanceQuestion|ParseAcceptanceReview|AcceptanceInstruction)'
```

Expected: PASS.

Commit:

```bash
git add backend/internal/service/chat/thinktank_intent_review.go backend/internal/service/chat/thinktank_intent_review_agents.go backend/internal/service/chat/thinktank_intent_review_test.go
git commit -m "feat: add visible acceptance summary"
```

### Task 3: Wire Visible Messages Into Non-Stream Flow

**Files:**
- Modify: `backend/internal/service/chat/thinktank_orchestrator.go`
- Test: `backend/internal/service/chat/thinktank_test.go`

- [ ] **Step 1: Write failing non-stream tests**

Add tests to `thinktank_test.go`:

```go
func TestThinkTankServiceChat_ClarifierReturnsStructuredQuestion(t *testing.T) {
	clarifier := &stubClarifier{decision: ClarifierDecision{
		NormalizedQuestion:    "制定学习计划",
		Intent:                "制定学习计划",
		AnswerGoal:            "tutorial",
		ShouldAskUser:         true,
		ClarificationQuestion: "你想学习哪个领域？",
		NeedSummary:           "制定一个可执行的学习计划",
		MissingDimensions:     []string{"学习领域", "当前基础", "学习目标"},
		WhyNeeded:             "不同领域、基础和目标会决定学习路径、资料难度和练习方式。",
		SuggestedReply:        "我想学 AI，目前零基础，目标是能做一个小项目。",
	}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier).(*thinkTankService)
	svc.adkRunner = &thinkTankADKRunner{}
	svc.adkAnswerFetcher = func(ctx context.Context, question string) (string, error) {
		t.Fatalf("ADK should not run when clarifier asks user")
		return "", nil
	}

	resp, err := svc.Chat(context.Background(), "我要学习知识", nil, nil)
	if err != nil {
		t.Fatalf("expected clarification response, got %v", err)
	}
	for _, want := range []string{"我理解你是想：制定一个可执行的学习计划", "为了后续回答更精确，需要确认：", "学习领域", "为什么需要这些信息：", "你可以这样回复："} {
		if !strings.Contains(resp.Message, want) {
			t.Fatalf("expected structured clarification to contain %q, got:\n%s", want, resp.Message)
		}
	}
}

func TestThinkTankServiceChat_AppendsAcceptanceSummary(t *testing.T) {
	clarifier := &stubClarifier{decision: ClarifierDecision{
		NormalizedQuestion: "分析 AI Agent 的发展趋势",
		Intent:             "分析 AI Agent 的发展趋势",
		TargetDimensions:   []string{"技术演进", "产品形态", "风险限制"},
	}}
	reviewer := &stubAcceptanceReviewer{reviews: []AcceptanceReview{{
		Verdict:           acceptanceVerdictPass,
		Score:             88,
		MatchedDimensions: []string{"技术演进", "产品形态", "风险限制"},
		Summary:           "回答覆盖了用户关注的主要维度。",
		Available:         true,
	}}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier, reviewer).(*thinkTankService)
	svc.adkRunner = &thinkTankADKRunner{}
	svc.adkAnswerFetcher = func(ctx context.Context, question string) (string, error) {
		return "AI Agent 趋势答案", nil
	}

	resp, err := svc.Chat(context.Background(), "帮我分析一下 AI Agent 的发展趋势", nil, nil)
	if err != nil {
		t.Fatalf("expected chat success, got %v", err)
	}
	for _, want := range []string{"AI Agent 趋势答案", "验收摘要：通过，评分 88/100", "已覆盖：技术演进、产品形态、风险限制"} {
		if !strings.Contains(resp.Message, want) {
			t.Fatalf("expected final response to contain %q, got:\n%s", want, resp.Message)
		}
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
cd backend
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'TestThinkTankServiceChat_(ClarifierReturnsStructuredQuestion|AppendsAcceptanceSummary)'
```

Expected: FAIL because `chat` still returns raw `ClarificationQuestion` and does not append acceptance summary.

- [ ] **Step 3: Wire structured clarifier question**

In `clarifyAgentQuery`, replace:

```go
if decision.ShouldAskUser {
	return queryForAgents, decision, true, strings.TrimSpace(decision.ClarificationQuestion)
}
```

with:

```go
if decision.ShouldAskUser {
	return queryForAgents, decision, true, formatClarifierQuestion(decision)
}
```

- [ ] **Step 4: Wire acceptance summary into non-stream answers**

In `chat`, introduce `revised := false` before review handling in both ADK and manual branches.

For ADK branch, after successful revision set:

```go
revised = true
```

Before `persistFinalAnswer` and return, add:

```go
finalAnswer = appendAcceptanceSummary(finalAnswer, review, revised)
```

For manual branch, do the same:

```go
revised := false
...
if revisionErr != nil || strings.TrimSpace(revisedAnswer) == "" {
	answer = appendAcceptanceLimitations(answer, review)
} else {
	revised = true
	answer = revisedAnswer
	...
}
answer = appendAcceptanceSummary(answer, review, revised)
```

When `acceptanceVerdictAskUser`, replace `review.UserQuestion` with `formatAcceptanceQuestion(review)` when calling `acceptanceQuestionResponse`.

- [ ] **Step 5: Run tests and commit**

Run:

```bash
cd backend
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'TestThinkTankServiceChat_(ClarifierReturnsStructuredQuestion|AppendsAcceptanceSummary|AcceptanceRevisionRunsOnce|AcceptanceCanAskUser)'
```

Expected: PASS.

Commit:

```bash
git add backend/internal/service/chat/thinktank_orchestrator.go backend/internal/service/chat/thinktank_test.go
git commit -m "feat: show review results in nonstream chat"
```

### Task 4: Wire Visible Steps And Summaries Into Stream Flow

**Files:**
- Modify: `backend/internal/service/chat/thinktank_orchestrator.go`
- Test: `backend/internal/service/chat/thinktank_adk_stage_test.go`

- [ ] **Step 1: Write failing stream tests**

Add tests to `thinktank_adk_stage_test.go`:

```go
func TestThinkTankService_ChatStream_EmitsClarifierAndAcceptanceSteps(t *testing.T) {
	clarifier := &stubClarifier{decision: ClarifierDecision{
		NormalizedQuestion: "分析 AI Agent 的发展趋势",
		Intent:             "分析 AI Agent 的发展趋势",
		NeedSummary:        "分析 AI Agent 的发展趋势",
		TargetDimensions:   []string{"技术演进", "产品形态", "风险限制"},
	}}
	reviewer := &stubAcceptanceReviewer{reviews: []AcceptanceReview{{
		Verdict:           acceptanceVerdictPass,
		Score:             88,
		MatchedDimensions: []string{"技术演进", "产品形态", "风险限制"},
		Summary:           "回答覆盖主要维度。",
		Available:         true,
	}}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier, reviewer).(*thinkTankService)
	svc.adkRunner = &thinkTankADKRunner{runner: nil}
	svc.adkAnswerFetcher = func(ctx context.Context, question string) (string, error) {
		return "AI Agent 趋势答案", nil
	}

	eventCh, errCh := svc.ChatStream(context.Background(), "帮我分析一下 AI Agent 的发展趋势", nil, nil)
	var stepAgents []string
	var finalChunk string
	for event := range eventCh {
		if event.Type == StreamEventStep {
			stepAgents = append(stepAgents, event.AgentName)
		}
		if event.Type == StreamEventChunk {
			finalChunk += event.Message
		}
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("expected no stream error, got %v", err)
		}
	}
	if !containsStage(stepAgents, "ClarifierAgent") {
		t.Fatalf("expected ClarifierAgent step in %#v", stepAgents)
	}
	if !containsStage(stepAgents, "AcceptanceAgent") {
		t.Fatalf("expected AcceptanceAgent step in %#v", stepAgents)
	}
	if !strings.Contains(finalChunk, "验收摘要：通过，评分 88/100") {
		t.Fatalf("expected final chunk to include acceptance summary, got:\n%s", finalChunk)
	}
}

func TestThinkTankService_ChatStream_ClarifierQuestionIsStructured(t *testing.T) {
	clarifier := &stubClarifier{decision: ClarifierDecision{
		NormalizedQuestion:    "制定学习计划",
		Intent:                "制定学习计划",
		ShouldAskUser:         true,
		ClarificationQuestion: "你想学习哪个领域？",
		NeedSummary:           "制定一个可执行的学习计划",
		MissingDimensions:     []string{"学习领域", "当前基础", "学习目标"},
		WhyNeeded:             "不同领域、基础和目标会决定学习路径。",
		SuggestedReply:        "我想学 AI，目前零基础。",
	}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier)

	eventCh, errCh := svc.ChatStream(context.Background(), "我要学习知识", nil, nil)
	var question string
	for event := range eventCh {
		if event.Type == StreamEventQuestion {
			question = event.Message
		}
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("expected no stream error, got %v", err)
		}
	}
	for _, want := range []string{"我理解你是想：制定一个可执行的学习计划", "学习领域", "为什么需要这些信息：", "你可以这样回复："} {
		if !strings.Contains(question, want) {
			t.Fatalf("expected structured stream question to contain %q, got:\n%s", want, question)
		}
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
cd backend
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'TestThinkTankService_ChatStream_(EmitsClarifierAndAcceptanceSteps|ClarifierQuestionIsStructured)'
```

Expected: FAIL because stream does not emit synthetic ClarifierAgent / AcceptanceAgent steps and question formatting is not wired everywhere yet.

- [ ] **Step 3: Add synthetic step helper**

In `thinktank_orchestrator.go`, add helper near emit methods:

```go
func (o *thinkTankOrchestrator) emitSyntheticAgentStep(eventCh chan<- StreamEvent, conv *model.Conversation, runID int64, agentName string, summary string, detail string) {
	conversationID := int64(0)
	if conv != nil {
		conversationID = conv.ID
	}
	step := o.service.runs.newStepTracker(conversationID, runID, agentName, summary)
	if strings.TrimSpace(detail) != "" {
		step.appendDetail(detail)
	}
	step.complete()
	o.emitStep(eventCh, conv, runID, step.snapshot())
}
```

- [ ] **Step 4: Emit ClarifierAgent steps**

In `chatStream`, after `clarifyAgentQuery` returns `clarifiedDecision`, emit:

```go
o.emitSyntheticAgentStep(eventCh, conv, runID, "ClarifierAgent", "已识别用户需求", formatClarifierStepDetail(clarifiedDecision))
```

Do this before returning for `needsUser` and before continuing for `!needsUser`.

- [ ] **Step 5: Emit AcceptanceAgent steps and append summary in stream branches**

In direct injected stream branch, after `reviewAnswer`, emit:

```go
o.emitSyntheticAgentStep(eventCh, conv, runID, "AcceptanceAgent", "已验收回答质量", formatAcceptanceStepDetail(review, false))
```

Track:

```go
revised := false
```

Set `revised = true` only when a non-empty revised answer is used.

Before `persistFinalAnswer`:

```go
answer = appendAcceptanceSummary(answer, review, revised)
```

Repeat the same pattern in `streamADKFlow` and `streamManualFlow`.

When `ask_user`, call:

```go
o.emitAcceptanceQuestion(eventCh, conv, runID, derefUserID(userID), question, formatAcceptanceQuestion(review), decision)
```

- [ ] **Step 6: Run stream tests and commit**

Run:

```bash
cd backend
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'TestThinkTankService_ChatStream'
```

Expected: PASS.

Commit:

```bash
git add backend/internal/service/chat/thinktank_orchestrator.go backend/internal/service/chat/thinktank_adk_stage_test.go
git commit -m "feat: show review steps in thinktank stream"
```

### Task 5: Frontend Contract Check And Full Verification

**Files:**
- Test only: `frontend/src/types/index.ts`
- Test only: `frontend/src/pages/AIChat.tsx`
- Test only: `frontend/src/store/chatStore.ts`

- [ ] **Step 1: Confirm no frontend code is required**

Check that:

- `ChatStage` already includes `clarifying_intent`, `reviewing`, and `revising`.
- `AgentProcessPanel` renders `StreamEventStep` details.
- Assistant message content is rendered through Markdown, so structured clarification and acceptance summary display correctly.

Run:

```bash
rg -n "clarifying_intent|reviewing|revising" frontend/src/types/index.ts
rg -n "AgentProcessPanel|processSteps|ArticleContent" frontend/src/pages/AIChat.tsx
rg -n "onStage|onQuestion|onStep|StreamEventStep" frontend/src/store/chatStore.ts
```

Expected:

- Stage type contains all three stages.
- `AgentProcessPanel` is rendered before answer Markdown.
- SSE `step`, `question`, and `stage` handlers already update state.

- [ ] **Step 2: Run backend package tests**

Run:

```bash
cd backend
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat
```

Expected: PASS.

- [ ] **Step 3: Run backend full tests**

Run:

```bash
cd backend
env GOTOOLCHAIN=go1.25.3 go test ./...
```

Expected: PASS.

- [ ] **Step 4: Run frontend verification**

Run:

```bash
cd frontend
npm run lint
npm run build
```

Expected: both PASS.

- [ ] **Step 5: Commit verification-only frontend result if code changed**

If no frontend files changed, do not commit. If frontend type or display code changed, commit:

```bash
git add frontend/src/types/index.ts frontend/src/pages/AIChat.tsx frontend/src/store/chatStore.ts
git commit -m "feat: surface thinktank review summaries"
```

### Task 6: Final Review And Merge Preparation

**Files:**
- Review only: all changed files

- [ ] **Step 1: Inspect final diff**

Run:

```bash
git status --short
git diff --stat
git diff --check
```

Expected:

- Only intended files changed.
- `git diff --check` emits no whitespace errors.

- [ ] **Step 2: Review behavior against spec**

Manually verify these acceptance criteria in tests or diff:

- “我要学习知识” produces structured clarification.
- Broad-but-clear questions continue without extra user interruption.
- Final answer includes visible acceptance summary when AcceptanceAgent returns available review.
- No fake score is appended when AcceptanceAgent fails or returns default unavailable review.
- Stream emits ClarifierAgent and AcceptanceAgent steps.

- [ ] **Step 3: Commit plan completion marker if needed**

No marker commit is required. If the implementation plan itself was edited during execution, commit it:

```bash
git add docs/superpowers/plans/2026-05-06-thinktank-visible-clarifier-acceptance.md
git commit -m "docs: add visible thinktank review implementation plan"
```

---

## Self-Review

**Spec coverage:** Covered structured ClarifierAgent demand profile, visible clarification question, broad question non-interruption, AcceptanceAgent score summary, revised-answer summary, stream steps, non-fake score behavior, and verification.

**Marker scan:** This plan contains no unresolved markers. Every task has concrete files, tests, code snippets, commands, and expected results.

**Type consistency:** New fields are consistently named `NeedSummary`, `MissingDimensions`, `WhyNeeded`, `SuggestedReply`, `Summary`, and `Available`. New helpers are consistently named `formatClarifierQuestion`, `formatClarifierStepDetail`, `appendAcceptanceSummary`, `formatAcceptanceQuestion`, `formatAcceptanceStepDetail`, and `emitSyntheticAgentStep`.
