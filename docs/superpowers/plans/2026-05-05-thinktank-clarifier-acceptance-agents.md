# ThinkTank Clarifier Acceptance Agents Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an Eino-backed ClarifierAgent and AcceptanceAgent around the existing ThinkTank PlanExecute flow so broad questions get inferred dimensions, critically missing questions ask one follow-up, and final answers pass a bounded review loop.

**Architecture:** Keep the current Eino `planexecute` runner as the execution core. Add two small service interfaces whose production implementations are Eino `ChatModelAgent` runners and whose tests use stubs. The orchestrator remains the deterministic owner of persistence, stream stages, interrupt behavior, and the maximum one-pass review revision.

**Tech Stack:** Go, Eino ADK v0.8.6, Eino `ChatModelAgent`, existing Ark model wrapper, Gin service layer tests with Go `testing`.

---

## Scope Check

This plan implements one cohesive backend feature: ClarifierAgent plus AcceptanceAgent around ThinkTank execution. It does not include frontend styling changes beyond backend stream stage names already consumed as strings, and it does not replace the existing PlanExecute flow with Supervisor.

## File Structure

- Create `backend/internal/service/chat/thinktank_intent_review.go`
  - Data structures, JSON parsing, prompt input formatting, review decision helpers.
- Create `backend/internal/service/chat/thinktank_intent_review_agents.go`
  - Eino `ChatModelAgent` construction and wrappers implementing `Clarifier` and `AcceptanceReviewer`.
- Create `backend/internal/service/chat/thinktank_intent_review_test.go`
  - Parsing, fallback, prompt policy, and deterministic review-loop helper tests.
- Modify `backend/internal/service/chat/thinktank_adk.go`
  - Attach Clarifier and Acceptance implementations to `thinkTankADKRunner`.
- Modify `backend/internal/service/chat/thinktank_service.go`
  - Add injectable clarifier/reviewer fields and default review config.
- Modify `backend/internal/service/chat/thinktank_orchestrator.go`
  - Run clarifier before PlanExecute, run acceptance after answer, and bound revision to one pass.
- Modify `backend/internal/service/chat/thinktank_run_record.go`
  - Persist generic agent-triggered clarification and normalized question metadata.
- Modify `backend/internal/service/chat/thinktank_adk_stage_test.go`
  - Update lifecycle expectations for new stream stages.
- Modify `backend/internal/service/chat/thinktank_test.go`
  - Add stub clarifier/reviewer and non-stream behavior tests.

---

### Task 1: Intent and Review Types

**Files:**
- Create: `backend/internal/service/chat/thinktank_intent_review.go`
- Create: `backend/internal/service/chat/thinktank_intent_review_test.go`

- [ ] **Step 1: Write failing parser tests**

Add this file:

```go
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
```

- [ ] **Step 2: Run parser tests to verify they fail**

Run from `backend/`:

```bash
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'TestParseClarifierDecision|TestParseAcceptanceReview|TestBuildClarifiedAgentQuery|TestBuildRevisionAgentQuery'
```

Expected: FAIL because `ClarifierDecision`, `AcceptanceReview`, and parser helpers are not defined.

- [ ] **Step 3: Implement intent and review helpers**

Create `backend/internal/service/chat/thinktank_intent_review.go`:

```go
package chat

import (
	"encoding/json"
	"strings"
)

const (
	thinkTankClarifierAgentName  = "clarifier"
	thinkTankAcceptanceAgentName = "acceptance"
	maxThinkTankReviewRevisions  = 1

	acceptanceVerdictPass    = "pass"
	acceptanceVerdictRevise  = "revise"
	acceptanceVerdictAskUser = "ask_user"
)

type ClarifierConstraints struct {
	TimeRange    string `json:"time_range"`
	Audience     string `json:"audience"`
	Depth        string `json:"depth"`
	Style        string `json:"style"`
	SourcePolicy string `json:"source_policy"`
}

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
}

type ClarifierInput struct {
	OriginalQuestion string
	AgentQuery       string
}

type AcceptanceReview struct {
	Verdict             string   `json:"verdict"`
	Score               int      `json:"score"`
	MatchedDimensions   []string `json:"matched_dimensions"`
	MissingDimensions   []string `json:"missing_dimensions"`
	UnsupportedClaims   []string `json:"unsupported_claims"`
	FormatIssues        []string `json:"format_issues"`
	RevisionInstruction string   `json:"revision_instruction"`
	UserQuestion         string   `json:"user_question"`
	Reason              string   `json:"reason"`
}

type AcceptanceReviewInput struct {
	OriginalQuestion string
	AgentQuery       string
	Decision         ClarifierDecision
	Answer           string
	RevisionCount    int
}

type Clarifier interface {
	Clarify(ctx context.Context, input ClarifierInput) (ClarifierDecision, error)
}

type AcceptanceReviewer interface {
	Review(ctx context.Context, input AcceptanceReviewInput) (AcceptanceReview, error)
}
```

Then add `context` to the import block and append these functions in the same file:

```go
func parseClarifierDecision(raw string, originalQuestion string) ClarifierDecision {
	var decision ClarifierDecision
	if err := json.Unmarshal([]byte(extractJSONObject(raw)), &decision); err != nil {
		return defaultClarifierDecision(originalQuestion)
	}
	decision.NormalizedQuestion = strings.TrimSpace(decision.NormalizedQuestion)
	if decision.NormalizedQuestion == "" {
		decision.NormalizedQuestion = strings.TrimSpace(originalQuestion)
	}
	decision.Intent = strings.TrimSpace(decision.Intent)
	if decision.Intent == "" {
		decision.Intent = decision.NormalizedQuestion
	}
	decision.AnswerGoal = strings.TrimSpace(decision.AnswerGoal)
	if decision.AnswerGoal == "" {
		decision.AnswerGoal = "explain"
	}
	decision.AmbiguityLevel = strings.TrimSpace(decision.AmbiguityLevel)
	if decision.AmbiguityLevel == "" {
		decision.AmbiguityLevel = "low"
	}
	decision.ClarificationQuestion = strings.TrimSpace(decision.ClarificationQuestion)
	if decision.ShouldAskUser && decision.ClarificationQuestion == "" {
		decision.ShouldAskUser = false
	}
	return decision
}

func defaultClarifierDecision(originalQuestion string) ClarifierDecision {
	question := strings.TrimSpace(originalQuestion)
	return ClarifierDecision{
		NormalizedQuestion: question,
		Intent:             question,
		AnswerGoal:         "explain",
		AmbiguityLevel:     "low",
		ShouldAskUser:      false,
		Reason:             "clarifier output unavailable; continuing with original question",
	}
}

func parseAcceptanceReview(raw string) AcceptanceReview {
	var review AcceptanceReview
	if err := json.Unmarshal([]byte(extractJSONObject(raw)), &review); err != nil {
		return defaultAcceptanceReview()
	}
	review.Verdict = normalizeAcceptanceVerdict(review.Verdict)
	if review.Score <= 0 {
		if review.Verdict == acceptanceVerdictPass {
			review.Score = 100
		} else {
			review.Score = 60
		}
	}
	review.RevisionInstruction = strings.TrimSpace(review.RevisionInstruction)
	review.UserQuestion = strings.TrimSpace(review.UserQuestion)
	if review.Verdict == acceptanceVerdictRevise && review.RevisionInstruction == "" {
		review.Verdict = acceptanceVerdictPass
	}
	if review.Verdict == acceptanceVerdictAskUser && review.UserQuestion == "" {
		review.Verdict = acceptanceVerdictPass
	}
	return review
}

func defaultAcceptanceReview() AcceptanceReview {
	return AcceptanceReview{
		Verdict: acceptanceVerdictPass,
		Score:   100,
		Reason:  "acceptance output unavailable; returning generated answer",
	}
}

func normalizeAcceptanceVerdict(verdict string) string {
	switch strings.TrimSpace(verdict) {
	case acceptanceVerdictRevise:
		return acceptanceVerdictRevise
	case acceptanceVerdictAskUser:
		return acceptanceVerdictAskUser
	default:
		return acceptanceVerdictPass
	}
}

func extractJSONObject(raw string) string {
	text := strings.TrimSpace(raw)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return text
	}
	return text[start : end+1]
}

func buildClarifiedAgentQuery(base string, decision ClarifierDecision) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(base))
	b.WriteString("\n\n[ClarifierAgent 意图画像]\n")
	b.WriteString("归一化问题：")
	b.WriteString(decision.NormalizedQuestion)
	b.WriteString("\n真实意图：")
	b.WriteString(decision.Intent)
	b.WriteString("\n回答目标：")
	b.WriteString(decision.AnswerGoal)
	if len(decision.TargetDimensions) > 0 {
		b.WriteString("\n用户关心维度：")
		b.WriteString(strings.Join(decision.TargetDimensions, "、"))
	}
	if decision.Constraints.TimeRange != "" || decision.Constraints.Audience != "" || decision.Constraints.Depth != "" || decision.Constraints.Style != "" || decision.Constraints.SourcePolicy != "" {
		b.WriteString("\n约束：")
		parts := make([]string, 0, 5)
		if decision.Constraints.TimeRange != "" {
			parts = append(parts, "时间范围="+decision.Constraints.TimeRange)
		}
		if decision.Constraints.Audience != "" {
			parts = append(parts, "受众="+decision.Constraints.Audience)
		}
		if decision.Constraints.Depth != "" {
			parts = append(parts, "深度="+decision.Constraints.Depth)
		}
		if decision.Constraints.Style != "" {
			parts = append(parts, "风格="+decision.Constraints.Style)
		}
		if decision.Constraints.SourcePolicy != "" {
			parts = append(parts, "来源要求="+decision.Constraints.SourcePolicy)
		}
		b.WriteString(strings.Join(parts, "；"))
	}
	return strings.TrimSpace(b.String())
}

func buildRevisionAgentQuery(base string, previousAnswer string, review AcceptanceReview) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(base))
	b.WriteString("\n\n[AcceptanceAgent 审核返工要求]\n")
	b.WriteString("上一版答案：\n")
	b.WriteString(strings.TrimSpace(previousAnswer))
	if len(review.MissingDimensions) > 0 {
		b.WriteString("\n\n缺失维度：")
		b.WriteString(strings.Join(review.MissingDimensions, "、"))
	}
	if len(review.UnsupportedClaims) > 0 {
		b.WriteString("\n证据不足：")
		b.WriteString(strings.Join(review.UnsupportedClaims, "、"))
	}
	if len(review.FormatIssues) > 0 {
		b.WriteString("\n格式问题：")
		b.WriteString(strings.Join(review.FormatIssues, "、"))
	}
	b.WriteString("\n返工指令：")
	b.WriteString(strings.TrimSpace(review.RevisionInstruction))
	return strings.TrimSpace(b.String())
}

func appendAcceptanceLimitations(answer string, review AcceptanceReview) string {
	if strings.TrimSpace(answer) == "" {
		return answer
	}
	parts := make([]string, 0, 3)
	if len(review.MissingDimensions) > 0 {
		parts = append(parts, "仍可能缺少："+strings.Join(review.MissingDimensions, "、"))
	}
	if len(review.UnsupportedClaims) > 0 {
		parts = append(parts, "部分判断证据不足："+strings.Join(review.UnsupportedClaims, "、"))
	}
	if review.Reason != "" {
		parts = append(parts, "审核说明："+review.Reason)
	}
	if len(parts) == 0 {
		return answer
	}
	return strings.TrimSpace(answer) + "\n\n回答限制：\n- " + strings.Join(parts, "\n- ")
}
```

Update the import block to:

```go
import (
	"context"
	"encoding/json"
	"strings"
)
```

- [ ] **Step 4: Run parser tests to verify they pass**

Run from `backend/`:

```bash
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'TestParseClarifierDecision|TestParseAcceptanceReview|TestBuildClarifiedAgentQuery|TestBuildRevisionAgentQuery'
```

Expected: PASS.

- [ ] **Step 5: Commit Task 1**

```bash
git add backend/internal/service/chat/thinktank_intent_review.go backend/internal/service/chat/thinktank_intent_review_test.go
git commit -m "feat: add thinktank intent review types"
```

---

### Task 2: Eino Clarifier and Acceptance Agent Wrappers

**Files:**
- Create: `backend/internal/service/chat/thinktank_intent_review_agents.go`
- Modify: `backend/internal/service/chat/thinktank_intent_review_test.go`

- [ ] **Step 1: Write failing prompt policy tests**

Append to `backend/internal/service/chat/thinktank_intent_review_test.go`:

```go
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
	for _, want := range []string{"AI Agent", "技术演进", "商业落地", "Revision count: 0", "verdict"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected acceptance prompt to contain %q, got %q", want, got)
		}
	}
}
```

- [ ] **Step 2: Run prompt tests to verify they fail**

Run from `backend/`:

```bash
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'TestClarifierInstruction|TestAcceptanceInstruction|TestBuildClarifierPrompt|TestBuildAcceptancePrompt'
```

Expected: FAIL because prompt constants and builders are not defined.

- [ ] **Step 3: Implement Eino-backed wrappers**

Create `backend/internal/service/chat/thinktank_intent_review_agents.go`:

```go
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	componentmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

const thinkTankClarifierInstruction = `You are ClarifierAgent for ThinkTank.
Your job is to understand the user's real intent before the research agents run.
Only ask the user when missing information would change what should be answered.
For a broad but clear request, infer useful target_dimensions and continue without asking.
Ask at most one concise follow-up question.
Do not expose chain-of-thought.
Return valid JSON only with these fields:
normalized_question, intent, answer_goal, target_dimensions, constraints, ambiguity_level, should_ask_user, clarification_question, reason.`

const thinkTankAcceptanceInstruction = `You are AcceptanceAgent for ThinkTank.
Your job is to check whether the final answer satisfies the user's real intent and target dimensions.
Return pass when the answer substantially satisfies the intent, even if it could be polished.
Return revise only when a key target dimension is missing, the answer is off-target, or important claims are unsupported.
Return ask_user only when missing user information prevents a useful answer or a fair review.
Do not nitpick wording.
Do not expose chain-of-thought.
Return valid JSON only with these fields:
verdict, score, matched_dimensions, missing_dimensions, unsupported_claims, format_issues, revision_instruction, user_question, reason.`

type einoClarifier struct {
	runner *adk.Runner
}

type einoAcceptanceReviewer struct {
	runner *adk.Runner
}

func newEinoClarifier(ctx context.Context, model componentmodel.ToolCallingChatModel) (Clarifier, error) {
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          thinkTankClarifierAgentName,
		Description:   "Identifies user intent, target dimensions, and whether one clarification question is required.",
		Instruction:   thinkTankClarifierInstruction,
		Model:         model,
		MaxIterations: 2,
	})
	if err != nil {
		return nil, err
	}
	return &einoClarifier{runner: adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})}, nil
}

func newEinoAcceptanceReviewer(ctx context.Context, model componentmodel.ToolCallingChatModel) (AcceptanceReviewer, error) {
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          thinkTankAcceptanceAgentName,
		Description:   "Reviews the final answer against the clarified user intent and returns pass, revise, or ask_user.",
		Instruction:   thinkTankAcceptanceInstruction,
		Model:         model,
		MaxIterations: 2,
	})
	if err != nil {
		return nil, err
	}
	return &einoAcceptanceReviewer{runner: adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})}, nil
}

func (c *einoClarifier) Clarify(ctx context.Context, input ClarifierInput) (ClarifierDecision, error) {
	if c == nil || c.runner == nil {
		return defaultClarifierDecision(input.OriginalQuestion), nil
	}
	raw, err := runSingleAgentText(ctx, c.runner, buildClarifierPrompt(input))
	if err != nil {
		return defaultClarifierDecision(input.OriginalQuestion), err
	}
	return parseClarifierDecision(raw, input.OriginalQuestion), nil
}

func (r *einoAcceptanceReviewer) Review(ctx context.Context, input AcceptanceReviewInput) (AcceptanceReview, error) {
	if r == nil || r.runner == nil {
		return defaultAcceptanceReview(), nil
	}
	raw, err := runSingleAgentText(ctx, r.runner, buildAcceptancePrompt(input))
	if err != nil {
		return defaultAcceptanceReview(), err
	}
	return parseAcceptanceReview(raw), nil
}

func runSingleAgentText(ctx context.Context, runner *adk.Runner, prompt string) (string, error) {
	iter := runner.Run(ctx, []adk.Message{schema.UserMessage(prompt)})
	var latest string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			return "", event.Err
		}
		msg, _, err := adk.GetMessage(event)
		if err != nil || msg == nil {
			continue
		}
		if strings.TrimSpace(msg.ToolName) == "" && strings.TrimSpace(msg.Content) != "" {
			latest = msg.Content
		}
	}
	if strings.TrimSpace(latest) == "" {
		return "", fmt.Errorf("agent returned empty content")
	}
	return latest, nil
}

func buildClarifierPrompt(input ClarifierInput) string {
	payload := map[string]any{
		"original_question": input.OriginalQuestion,
		"agent_query":       input.AgentQuery,
		"policy": "If missing information changes what should be answered, set should_ask_user=true. If the request is broad but clear, infer target_dimensions and continue.",
	}
	bytes, _ := json.MarshalIndent(payload, "", "  ")
	return string(bytes)
}

func buildAcceptancePrompt(input AcceptanceReviewInput) string {
	payload := map[string]any{
		"original_question": input.OriginalQuestion,
		"agent_query":       input.AgentQuery,
		"clarifier_decision": input.Decision,
		"answer":            input.Answer,
		"revision_count":    input.RevisionCount,
		"instruction":       "Return verdict=pass, revise, or ask_user. Revision count: " + fmt.Sprintf("%d", input.RevisionCount),
	}
	bytes, _ := json.MarshalIndent(payload, "", "  ")
	return string(bytes)
}
```

- [ ] **Step 4: Run prompt tests to verify they pass**

Run from `backend/`:

```bash
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'TestClarifierInstruction|TestAcceptanceInstruction|TestBuildClarifierPrompt|TestBuildAcceptancePrompt'
```

Expected: PASS.

- [ ] **Step 5: Commit Task 2**

```bash
git add backend/internal/service/chat/thinktank_intent_review_agents.go backend/internal/service/chat/thinktank_intent_review_test.go
git commit -m "feat: add eino thinktank review agents"
```

---

### Task 3: Wire Agents Into Runner and Service

**Files:**
- Modify: `backend/internal/service/chat/thinktank_adk.go`
- Modify: `backend/internal/service/chat/thinktank_service.go`
- Modify: `backend/internal/service/chat/thinktank_test.go`

- [ ] **Step 1: Write failing service wiring test**

Append to `backend/internal/service/chat/thinktank_test.go`:

```go
type stubClarifier struct {
	decision ClarifierDecision
	err      error
	calls    int
	inputs   []ClarifierInput
}

func (s *stubClarifier) Clarify(ctx context.Context, input ClarifierInput) (ClarifierDecision, error) {
	s.calls++
	s.inputs = append(s.inputs, input)
	return s.decision, s.err
}

type stubAcceptanceReviewer struct {
	reviews []AcceptanceReview
	err     error
	calls   int
	inputs  []AcceptanceReviewInput
}

func (s *stubAcceptanceReviewer) Review(ctx context.Context, input AcceptanceReviewInput) (AcceptanceReview, error) {
	s.calls++
	s.inputs = append(s.inputs, input)
	if len(s.reviews) >= s.calls {
		return s.reviews[s.calls-1], s.err
	}
	return defaultAcceptanceReview(), s.err
}

func TestNewThinkTankService_UsesInjectedClarifierAndAcceptanceReviewer(t *testing.T) {
	clarifier := &stubClarifier{decision: defaultClarifierDecision("问题")}
	reviewer := &stubAcceptanceReviewer{reviews: []AcceptanceReview{defaultAcceptanceReview()}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier, reviewer).(*thinkTankService)

	if svc.clarifier != clarifier {
		t.Fatalf("expected injected clarifier to be stored")
	}
	if svc.acceptanceReviewer != reviewer {
		t.Fatalf("expected injected acceptance reviewer to be stored")
	}
	if svc.maxReviewRevisions != maxThinkTankReviewRevisions {
		t.Fatalf("expected default max review revisions, got %d", svc.maxReviewRevisions)
	}
}
```

- [ ] **Step 2: Run service wiring test to verify it fails**

Run from `backend/`:

```bash
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run TestNewThinkTankService_UsesInjectedClarifierAndAcceptanceReviewer
```

Expected: FAIL because `thinkTankService` has no `clarifier`, `acceptanceReviewer`, or `maxReviewRevisions` fields.

- [ ] **Step 3: Extend runner structure**

In `backend/internal/service/chat/thinktank_adk.go`, change `thinkTankADKRunner` to:

```go
type thinkTankADKRunner struct {
	runner          *adk.Runner
	agent           adk.ResumableAgent
	checkpointStore *thinkTankCheckpointStore
	clarifier       Clarifier
	acceptance      AcceptanceReviewer
}
```

In `NewThinkTankADKRunner`, after the model type assertion and before tool setup, add:

```go
	clarifier, err := newEinoClarifier(ctx, model)
	if err != nil {
		return nil, err
	}
	acceptance, err := newEinoAcceptanceReviewer(ctx, model)
	if err != nil {
		return nil, err
	}
```

In the returned struct, add:

```go
		clarifier:       clarifier,
		acceptance:      acceptance,
```

- [ ] **Step 4: Extend service dependency fields**

In `backend/internal/service/chat/thinktank_service.go`, add fields to `thinkTankService`:

```go
	clarifier          Clarifier
	acceptanceReviewer AcceptanceReviewer
	maxReviewRevisions int
```

In `NewThinkTankService`, add local variables:

```go
	var clarifier Clarifier
	var acceptanceReviewer AcceptanceReviewer
```

Extend the options switch:

```go
		case Clarifier:
			clarifier = v
		case AcceptanceReviewer:
			acceptanceReviewer = v
```

After the options loop, add:

```go
	if runner != nil {
		if clarifier == nil {
			clarifier = runner.clarifier
		}
		if acceptanceReviewer == nil {
			acceptanceReviewer = runner.acceptance
		}
	}
```

In the `svc` struct literal, add:

```go
		clarifier:          clarifier,
		acceptanceReviewer: acceptanceReviewer,
		maxReviewRevisions: maxThinkTankReviewRevisions,
```

- [ ] **Step 5: Run service wiring test to verify it passes**

Run from `backend/`:

```bash
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run TestNewThinkTankService_UsesInjectedClarifierAndAcceptanceReviewer
```

Expected: PASS.

- [ ] **Step 6: Commit Task 3**

```bash
git add backend/internal/service/chat/thinktank_adk.go backend/internal/service/chat/thinktank_service.go backend/internal/service/chat/thinktank_test.go
git commit -m "feat: wire thinktank review agents"
```

---

### Task 4: Non-Streaming Clarifier and Acceptance Flow

**Files:**
- Modify: `backend/internal/service/chat/thinktank_orchestrator.go`
- Modify: `backend/internal/service/chat/thinktank_test.go`

- [ ] **Step 1: Write failing non-stream tests**

Append to `backend/internal/service/chat/thinktank_test.go`:

```go
func TestThinkTankServiceChat_ClarifierEnhancesADKQueryWithoutAsking(t *testing.T) {
	clarifier := &stubClarifier{decision: ClarifierDecision{
		NormalizedQuestion: "分析 AI Agent 的发展趋势",
		Intent:             "了解 AI Agent 的发展趋势",
		AnswerGoal:         "research",
		TargetDimensions:   []string{"技术演进", "商业落地"},
	}}
	reviewer := &stubAcceptanceReviewer{reviews: []AcceptanceReview{defaultAcceptanceReview()}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier, reviewer).(*thinkTankService)
	svc.adkRunner = &thinkTankADKRunner{}
	svc.adkAnswerFetcher = func(ctx context.Context, question string) (string, error) {
		if !strings.Contains(question, "技术演进") || !strings.Contains(question, "商业落地") {
			t.Fatalf("expected clarified dimensions in ADK query, got %q", question)
		}
		return "AI Agent 趋势答案", nil
	}

	resp, err := svc.Chat(context.Background(), "帮我分析一下 AI Agent 的发展趋势", nil, nil)
	if err != nil {
		t.Fatalf("expected chat success, got %v", err)
	}
	if resp.RequiresUserInput {
		t.Fatalf("did not expect user input for broad but clear question")
	}
	if resp.Message != "AI Agent 趋势答案" {
		t.Fatalf("unexpected response %q", resp.Message)
	}
	if clarifier.calls != 1 || reviewer.calls != 1 {
		t.Fatalf("expected one clarifier and one reviewer call, got %d/%d", clarifier.calls, reviewer.calls)
	}
}

func TestThinkTankServiceChat_ClarifierCanAskUser(t *testing.T) {
	clarifier := &stubClarifier{decision: ClarifierDecision{
		NormalizedQuestion:    "帮我看看这个报错怎么修",
		Intent:                "定位报错",
		ShouldAskUser:         true,
		ClarificationQuestion: "请把完整报错信息、触发操作和相关代码片段发我。",
	}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier).(*thinkTankService)
	svc.adkRunner = &thinkTankADKRunner{}
	svc.adkAnswerFetcher = func(ctx context.Context, question string) (string, error) {
		t.Fatalf("ADK should not run when clarifier asks user, got %q", question)
		return "", nil
	}

	resp, err := svc.Chat(context.Background(), "帮我看看这个报错怎么修", nil, nil)
	if err != nil {
		t.Fatalf("expected clarification response without error, got %v", err)
	}
	if !resp.RequiresUserInput {
		t.Fatalf("expected requires user input")
	}
	if resp.Stage != "clarifying" {
		t.Fatalf("expected clarifying stage, got %q", resp.Stage)
	}
	if !strings.Contains(resp.Message, "完整报错信息") {
		t.Fatalf("expected clarification question, got %q", resp.Message)
	}
}

func TestThinkTankServiceChat_AcceptanceRevisionRunsOnce(t *testing.T) {
	clarifier := &stubClarifier{decision: defaultClarifierDecision("帮我分析一下 AI Agent 的发展趋势")}
	reviewer := &stubAcceptanceReviewer{reviews: []AcceptanceReview{
		{
			Verdict:             acceptanceVerdictRevise,
			MissingDimensions:   []string{"风险限制"},
			RevisionInstruction: "补充风险限制",
		},
		defaultAcceptanceReview(),
	}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier, reviewer).(*thinkTankService)
	svc.adkRunner = &thinkTankADKRunner{}
	var calls int
	svc.adkAnswerFetcher = func(ctx context.Context, question string) (string, error) {
		calls++
		if calls == 2 && !strings.Contains(question, "补充风险限制") {
			t.Fatalf("expected revision instruction in second ADK query, got %q", question)
		}
		return "第" + strconv.Itoa(calls) + "版答案", nil
	}

	resp, err := svc.Chat(context.Background(), "帮我分析一下 AI Agent 的发展趋势", nil, nil)
	if err != nil {
		t.Fatalf("expected chat success, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected one revision run, got %d calls", calls)
	}
	if resp.Message != "第2版答案" {
		t.Fatalf("expected revised answer, got %q", resp.Message)
	}
}
```

Add `strconv` to the import block in `thinktank_test.go`.

- [ ] **Step 2: Run non-stream tests to verify they fail**

Run from `backend/`:

```bash
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'TestThinkTankServiceChat_Clarifier|TestThinkTankServiceChat_AcceptanceRevisionRunsOnce'
```

Expected: FAIL because `chat` does not call clarifier or acceptance reviewer.

- [ ] **Step 3: Add non-stream helper methods**

In `backend/internal/service/chat/thinktank_orchestrator.go`, add these methods before `chat`:

```go
func (o *thinkTankOrchestrator) clarifyAgentQuery(ctx context.Context, question string, queryForAgents string) (string, ClarifierDecision, bool, string) {
	decision := defaultClarifierDecision(question)
	if o == nil || o.service == nil || o.service.clarifier == nil {
		return queryForAgents, decision, false, ""
	}
	got, err := o.service.clarifier.Clarify(ctx, ClarifierInput{
		OriginalQuestion: question,
		AgentQuery:       queryForAgents,
	})
	if err != nil {
		o.service.runs.logStage(nil, nil, "clarifier_warning", "ClarifierAgent failed; continuing with original question", err.Error())
		return queryForAgents, decision, false, ""
	}
	if strings.TrimSpace(got.NormalizedQuestion) == "" {
		got = defaultClarifierDecision(question)
	}
	if got.ShouldAskUser {
		return queryForAgents, got, true, got.ClarificationQuestion
	}
	return buildClarifiedAgentQuery(queryForAgents, got), got, false, ""
}

func (o *thinkTankOrchestrator) reviewAnswer(ctx context.Context, question string, queryForAgents string, decision ClarifierDecision, answer string, revisionCount int) (AcceptanceReview, bool) {
	if o == nil || o.service == nil || o.service.acceptanceReviewer == nil || strings.TrimSpace(answer) == "" {
		return defaultAcceptanceReview(), false
	}
	review, err := o.service.acceptanceReviewer.Review(ctx, AcceptanceReviewInput{
		OriginalQuestion: question,
		AgentQuery:       queryForAgents,
		Decision:         decision,
		Answer:           answer,
		RevisionCount:    revisionCount,
	})
	if err != nil {
		o.service.runs.logStage(nil, nil, "acceptance_warning", "AcceptanceAgent failed; returning generated answer", err.Error())
		return defaultAcceptanceReview(), false
	}
	return review, review.Verdict == acceptanceVerdictRevise && revisionCount < o.service.maxReviewRevisions
}
```

- [ ] **Step 4: Use helpers in `chat`**

In `chat`, after:

```go
	queryForAgents := o.buildAgentQuery(question, conv, history)
```

insert:

```go
	queryForAgents, clarifierDecision, needsUser, clarificationQuestion := o.clarifyAgentQuery(ctx, question, queryForAgents)
	if needsUser {
		if conv != nil {
			s.conversations.saveMessageWithWarning(conv.ID, "assistant", clarificationQuestion, "Failed to save clarification message")
		}
		return &ThinkTankChatResponse{
			Message:           clarificationQuestion,
			Stage:             "clarifying",
			RequiresUserInput: true,
		}, nil
	}
```

In the ADK answer branch, replace:

```go
			o.persistFinalAnswer(conv, derefUserID(userID), question, answer, decision, history, 0)
			return &ThinkTankChatResponse{Message: answer, Stage: "completed"}, nil
```

with:

```go
			review, shouldRevise := o.reviewAnswer(ctx, question, queryForAgents, clarifierDecision, answer, 0)
			if shouldRevise {
				revisedQuery := buildRevisionAgentQuery(queryForAgents, answer, review)
				revisedAnswer, revisedErr := s.adkAnswerFetcher(adkCtx, revisedQuery)
				if revisedErr == nil && strings.TrimSpace(revisedAnswer) != "" {
					answer = revisedAnswer
					review, _ = o.reviewAnswer(ctx, question, queryForAgents, clarifierDecision, answer, 1)
					if review.Verdict == acceptanceVerdictRevise {
						answer = appendAcceptanceLimitations(answer, review)
					}
				}
			}
			o.persistFinalAnswer(conv, derefUserID(userID), question, answer, decision, history, 0)
			return &ThinkTankChatResponse{Message: answer, Stage: "completed"}, nil
```

In the manual fallback branch, before persisting the final answer, insert:

```go
	review, shouldRevise := o.reviewAnswer(ctx, question, queryForAgents, clarifierDecision, answer, 0)
	if shouldRevise && s.adkAnswerFetcher != nil && s.adkRunner != nil {
		revisedQuery := buildRevisionAgentQuery(queryForAgents, answer, review)
		if revisedAnswer, revisedErr := s.adkAnswerFetcher(ctx, revisedQuery); revisedErr == nil && strings.TrimSpace(revisedAnswer) != "" {
			answer = revisedAnswer
			review, _ = o.reviewAnswer(ctx, question, queryForAgents, clarifierDecision, answer, 1)
			if review.Verdict == acceptanceVerdictRevise {
				answer = appendAcceptanceLimitations(answer, review)
			}
		}
	}
```

- [ ] **Step 5: Run non-stream tests to verify they pass**

Run from `backend/`:

```bash
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'TestThinkTankServiceChat_Clarifier|TestThinkTankServiceChat_AcceptanceRevisionRunsOnce'
```

Expected: PASS.

- [ ] **Step 6: Commit Task 4**

```bash
git add backend/internal/service/chat/thinktank_orchestrator.go backend/internal/service/chat/thinktank_test.go
git commit -m "feat: add thinktank nonstream review loop"
```

---

### Task 5: Stream Clarifier Stage and Clarifier Questions

**Files:**
- Modify: `backend/internal/service/chat/thinktank_orchestrator.go`
- Modify: `backend/internal/service/chat/thinktank_run_record.go`
- Modify: `backend/internal/service/chat/thinktank_adk_stage_test.go`

- [ ] **Step 1: Write failing stream lifecycle tests**

Update `TestThinkTankService_ChatStream_EmitsFullStageLifecycle` in `backend/internal/service/chat/thinktank_adk_stage_test.go` so service creation includes a non-asking clarifier and passing reviewer:

```go
	clarifier := &stubClarifier{decision: ClarifierDecision{
		NormalizedQuestion: "调研一下李小龙",
		Intent:             "了解李小龙的人物背景和影响",
		AnswerGoal:         "research",
		TargetDimensions:   []string{"生平背景", "核心成就", "武术思想"},
	}}
	reviewer := &stubAcceptanceReviewer{reviews: []AcceptanceReview{defaultAcceptanceReview()}}
	svc := NewThinkTankService(librarian, journalist, synthesizer, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier, reviewer)
```

Change the expected stage list to:

```go
	expected := []string{"analyzing", "clarifying_intent", "local_search", "web_research", "integration", "reviewing", "completed"}
```

Append this test to the same file:

```go
func TestThinkTankService_ChatStream_EmitsQuestionWhenClarifierNeedsUser(t *testing.T) {
	clarifier := &stubClarifier{decision: ClarifierDecision{
		NormalizedQuestion:    "帮我看看这个报错怎么修",
		Intent:                "定位报错",
		ShouldAskUser:         true,
		ClarificationQuestion: "请把完整报错信息发我。",
	}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier)

	eventCh, errCh := svc.ChatStream(context.Background(), "帮我看看这个报错怎么修", nil, nil)
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
	if !strings.Contains(question, "完整报错信息") {
		t.Fatalf("expected clarifier question event, got %q", question)
	}
}
```

Add `strings` to the import block if it is not already present.

- [ ] **Step 2: Run stream lifecycle tests to verify they fail**

Run from `backend/`:

```bash
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'TestThinkTankService_ChatStream_EmitsFullStageLifecycle|TestThinkTankService_ChatStream_EmitsQuestionWhenClarifierNeedsUser'
```

Expected: FAIL because stream flow does not emit clarifier stages or questions.

- [ ] **Step 3: Add generic agent clarification persistence**

In `backend/internal/service/chat/thinktank_run_record.go`, after `persistADKClarification`, add:

```go
func (r *thinkTankRunRecorder) persistAgentClarification(conversationID int64, userID int64, runID int64, question string, clarification string, stage string, pendingContext string, decision PlannerDecision) {
	if r == nil || r.runRepo == nil {
		return
	}
	if stage == "" {
		stage = "clarifying"
	}
	run := &model.ConversationRun{
		ID:               runID,
		ConversationID:   conversationID,
		UserID:           userID,
		Status:           "waiting_user",
		CurrentStage:     stage,
		OriginalQuestion: question,
		PendingQuestion:  &clarification,
		LastAnswer:       clarification,
		LastPlan:         decision.PlanSummary,
		PendingContext:   pendingContext,
	}
	if run.ID > 0 {
		_ = r.runRepo.Update(run)
		return
	}
	_ = r.runRepo.Create(run)
}
```

- [ ] **Step 4: Move run creation before stream clarification and emit stages**

In `chatStream`, replace the block from:

```go
		o.emitStage(eventCh, conv, 0, "analyzing", "正在理解你的问题")
		decision := PlannerDecision{ExecutionStrategy: "eino_plan_execute_replan", PlanSummary: "由 Eino PlanExecute planner 生成计划"}
		o.emitStage(eventCh, conv, 0, "analyzing", "正在进行多 Agent 深度调研")
		s.runs.logStage(conv, userID, "adk_start", "开始多 Agent 协作流", question)

		queryForAgents := o.buildAgentQuery(question, conv, history)
		runID, checkpointID, resumeFromADKInterrupt := o.prepareADKRun(conv, pending, userID, question, decision)
```

with:

```go
		o.emitStage(eventCh, conv, 0, "analyzing", "正在理解你的问题")
		decision := PlannerDecision{ExecutionStrategy: "eino_plan_execute_replan", PlanSummary: "由 Eino PlanExecute planner 生成计划"}
		queryForAgents := o.buildAgentQuery(question, conv, history)
		runID, checkpointID, resumeFromADKInterrupt := o.prepareADKRun(conv, pending, userID, question, decision)
		o.emitResume(eventCh, conv, runID, "analyzing", "running")
		o.emitSnapshot(eventCh, conv, runID, "analyzing", "running", "")

		o.emitStage(eventCh, conv, runID, "clarifying_intent", "正在理解你的真实意图")
		queryForAgents, clarifierDecision, needsUser, clarificationQuestion := o.clarifyAgentQuery(runCtx, question, queryForAgents)
		if needsUser {
			if conv != nil {
				s.runs.persistAgentClarification(conv.ID, derefUserID(userID), runID, question, clarificationQuestion, "clarifying", `{"type":"clarifier_interrupt"}`, decision)
				s.conversations.saveMessageWithWarning(conv.ID, "assistant", clarificationQuestion, "Failed to save clarification message", runID)
			}
			o.emitStage(eventCh, conv, runID, "clarifying", "需要补充一点信息")
			o.emitQuestion(eventCh, conv, runID, "clarifying", clarificationQuestion)
			return
		}
		_ = clarifierDecision

		o.emitStage(eventCh, conv, runID, "analyzing", "正在进行多 Agent 深度调研")
		s.runs.logStage(conv, userID, "adk_start", "开始多 Agent 协作流", question)
```

Remove the old duplicate lines:

```go
		o.emitResume(eventCh, conv, runID, "analyzing", "running")
		o.emitSnapshot(eventCh, conv, runID, "analyzing", "running", "")
```

from below the original `prepareADKRun` call.

- [ ] **Step 5: Run stream lifecycle tests to verify they pass**

Run from `backend/`:

```bash
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run 'TestThinkTankService_ChatStream_EmitsFullStageLifecycle|TestThinkTankService_ChatStream_EmitsQuestionWhenClarifierNeedsUser'
```

Expected: PASS.

- [ ] **Step 6: Commit Task 5**

```bash
git add backend/internal/service/chat/thinktank_orchestrator.go backend/internal/service/chat/thinktank_run_record.go backend/internal/service/chat/thinktank_adk_stage_test.go
git commit -m "feat: add thinktank clarifier stream stage"
```

---

### Task 6: Stream Acceptance Review and Bounded Revision

**Files:**
- Modify: `backend/internal/service/chat/thinktank_orchestrator.go`
- Modify: `backend/internal/service/chat/thinktank_adk_stage_test.go`

- [ ] **Step 1: Write failing stream review test**

Append to `backend/internal/service/chat/thinktank_adk_stage_test.go`:

```go
func TestThinkTankService_ChatStream_EmitsReviewingAndRevisionStages(t *testing.T) {
	clarifier := &stubClarifier{decision: defaultClarifierDecision("帮我分析一下 AI Agent 的发展趋势")}
	reviewer := &stubAcceptanceReviewer{reviews: []AcceptanceReview{
		{
			Verdict:             acceptanceVerdictRevise,
			MissingDimensions:   []string{"风险限制"},
			RevisionInstruction: "补充风险限制",
		},
		defaultAcceptanceReview(),
	}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier, reviewer).(*thinkTankService)
	svc.adkRunner = &thinkTankADKRunner{runner: nil}
	var calls int
	svc.adkAnswerFetcher = func(ctx context.Context, question string) (string, error) {
		calls++
		if calls == 2 && !strings.Contains(question, "补充风险限制") {
			t.Fatalf("expected revision instruction in second run, got %q", question)
		}
		return "第" + strconv.Itoa(calls) + "版答案", nil
	}

	eventCh, errCh := svc.ChatStream(context.Background(), "帮我分析一下 AI Agent 的发展趋势", nil, nil)
	var stages []string
	var finalChunk string
	for event := range eventCh {
		if event.Type == StreamEventStage {
			stages = append(stages, event.Stage)
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
	for _, want := range []string{"reviewing", "revising", "completed"} {
		if !containsStage(stages, want) {
			t.Fatalf("expected stage %q in %#v", want, stages)
		}
	}
	if calls != 2 {
		t.Fatalf("expected one revision, got %d calls", calls)
	}
	if !strings.Contains(finalChunk, "第2版答案") {
		t.Fatalf("expected revised answer chunk, got %q", finalChunk)
	}
}
```

Add `strconv` to the import block.

- [ ] **Step 2: Run stream review test to verify it fails**

Run from `backend/`:

```bash
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run TestThinkTankService_ChatStream_EmitsReviewingAndRevisionStages
```

Expected: FAIL because stream flow does not review or revise.

- [ ] **Step 3: Add stream helper for direct ADK answer fetch path**

In `chatStream`, before choosing `streamADKFlow` versus manual fallback, add a direct path for injected `adkAnswerFetcher` when `s.adkRunner.runner == nil`:

```go
		if s.adkRunner != nil && s.adkRunner.runner == nil && s.adkAnswerFetcher != nil {
			answer, err := s.adkAnswerFetcher(runCtx, queryForAgents)
			if err != nil {
				errCh <- err
				return
			}
			o.emitStage(eventCh, conv, runID, "reviewing", "正在验收回答质量")
			review, shouldRevise := o.reviewAnswer(runCtx, question, queryForAgents, clarifierDecision, answer, 0)
			if shouldRevise {
				o.emitStage(eventCh, conv, runID, "revising", "正在根据验收意见补充")
				revisedQuery := buildRevisionAgentQuery(queryForAgents, answer, review)
				if revisedAnswer, revisedErr := s.adkAnswerFetcher(runCtx, revisedQuery); revisedErr == nil && strings.TrimSpace(revisedAnswer) != "" {
					answer = revisedAnswer
					review, _ = o.reviewAnswer(runCtx, question, queryForAgents, clarifierDecision, answer, 1)
					if review.Verdict == acceptanceVerdictRevise {
						answer = appendAcceptanceLimitations(answer, review)
					}
				}
			}
			o.persistFinalAnswer(conv, derefUserID(userID), question, answer, decision, history, runID)
			o.emitChunk(eventCh, conv, runID, answer, nil)
			o.emitDone(eventCh, conv, runID, "completed", "回答已生成")
			return
		}
```

This helper keeps test coverage deterministic without a real Eino model.

- [ ] **Step 4: Add review stage to manual stream flow**

In `streamManualFlow`, after `integration_done` and before `persistFinalAnswer`, add:

```go
	o.emitStage(eventCh, conv, runID, "reviewing", "正在验收回答质量")
```

This makes manual fallback emit the same review stage. Manual fallback does not run a revision unless an ADK answer fetcher is present, because its only available path is the older librarian/journalist/synthesizer flow.

- [ ] **Step 5: Add review stage to ADK stream flow**

In `streamADKFlow`, after `fullAnswer` is finalized and before `currentStep.complete()`, add:

```go
	o.emitStage(eventCh, conv, runID, "reviewing", "正在验收回答质量")
```

Full ADK stream revision is covered by a direct fetch path first. If production needs token-level streaming for the revised run, add a second `streamADKFlow` invocation with a revision query in a separate focused change.

- [ ] **Step 6: Run stream review test to verify it passes**

Run from `backend/`:

```bash
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat -run TestThinkTankService_ChatStream_EmitsReviewingAndRevisionStages
```

Expected: PASS.

- [ ] **Step 7: Commit Task 6**

```bash
git add backend/internal/service/chat/thinktank_orchestrator.go backend/internal/service/chat/thinktank_adk_stage_test.go
git commit -m "feat: add thinktank stream review stage"
```

---

### Task 7: Full Verification

**Files:**
- Modify only files touched by previous tasks if verification reveals compile or test issues.

- [ ] **Step 1: Run focused chat package tests**

Run from `backend/`:

```bash
env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chat
```

Expected: PASS.

- [ ] **Step 2: Run server package tests that depend on service wiring**

Run from `backend/`:

```bash
env GOTOOLCHAIN=go1.25.3 go test ./cmd/server
```

Expected: PASS.

- [ ] **Step 3: Run all backend tests**

Run from `backend/`:

```bash
env GOTOOLCHAIN=go1.25.3 go test ./...
```

Expected: PASS.

- [ ] **Step 4: Check changed files**

Run from repo root:

```bash
git status --short
git diff --stat
```

Expected: only intended ThinkTank files have changes.

- [ ] **Step 5: Commit verification fixes if any were required**

If Step 1, Step 2, or Step 3 required compile fixes, commit them:

```bash
git add backend/internal/service/chat backend/cmd/server
git commit -m "fix: stabilize thinktank review agent integration"
```

Expected: commit created only when fixes were necessary.

---

## Self-Review

Spec coverage:

- ClarifierAgent structure, critical-missing policy, and broad-question inference are covered by Tasks 1, 2, 4, and 5.
- AcceptanceAgent schema, review verdicts, and one-pass bounded revision are covered by Tasks 1, 2, 4, and 6.
- Eino-first implementation is covered by Task 2 and runner wiring in Task 3.
- Persistence and stream stages are covered by Tasks 5 and 6.
- Testing requirements are covered by every task and the full verification task.

Type consistency:

- `Clarifier`, `AcceptanceReviewer`, `ClarifierDecision`, and `AcceptanceReview` are defined in Task 1 before being used in later tasks.
- Service fields are added in Task 3 before orchestrator calls in Task 4.
- Stream tests use stubs introduced in Task 3.

Placeholder scan:

- The plan contains no `TBD`, `TODO`, or undefined task-owned symbols. The remaining `...` text is only the required Go package pattern in `go test ./...`.
