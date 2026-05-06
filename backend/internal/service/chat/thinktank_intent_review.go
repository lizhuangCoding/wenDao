package chat

import (
	"context"
	"encoding/json"
	"strconv"
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
	NeedSummary           string               `json:"need_summary"`
	MissingDimensions     []string             `json:"missing_dimensions"`
	WhyNeeded             string               `json:"why_needed"`
	SuggestedReply        string               `json:"suggested_reply"`
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
	UserQuestion        string   `json:"user_question"`
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
	decision.NeedSummary = strings.TrimSpace(decision.NeedSummary)
	if decision.NeedSummary == "" {
		decision.NeedSummary = decision.Intent
	}
	decision.MissingDimensions = compactNonEmptyStrings(decision.MissingDimensions)
	if decision.ShouldAskUser && len(decision.MissingDimensions) == 0 && decision.ClarificationQuestion != "" {
		decision.MissingDimensions = []string{decision.ClarificationQuestion}
	}
	decision.WhyNeeded = strings.TrimSpace(decision.WhyNeeded)
	decision.SuggestedReply = strings.TrimSpace(decision.SuggestedReply)
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

func formatClarifierQuestion(decision ClarifierDecision) string {
	needSummary := strings.TrimSpace(decision.NeedSummary)
	if needSummary == "" {
		needSummary = strings.TrimSpace(decision.Intent)
	}
	if needSummary == "" {
		needSummary = strings.TrimSpace(decision.NormalizedQuestion)
	}

	missingDimensions := compactNonEmptyStrings(decision.MissingDimensions)
	if len(missingDimensions) == 0 && strings.TrimSpace(decision.ClarificationQuestion) != "" {
		missingDimensions = []string{strings.TrimSpace(decision.ClarificationQuestion)}
	}

	var b strings.Builder
	if needSummary != "" {
		b.WriteString("我理解你是想：")
		b.WriteString(needSummary)
	}
	if len(missingDimensions) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("为了后续回答更精确，需要确认：")
		for i, dimension := range missingDimensions {
			b.WriteString("\n")
			b.WriteString(strconv.Itoa(i + 1))
			b.WriteString(". ")
			b.WriteString(dimension)
		}
	}
	if whyNeeded := strings.TrimSpace(decision.WhyNeeded); whyNeeded != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("为什么需要这些信息：\n")
		b.WriteString(whyNeeded)
	}
	if suggestedReply := strings.TrimSpace(decision.SuggestedReply); suggestedReply != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("你可以这样回复：\n")
		b.WriteString(suggestedReply)
	}
	if b.Len() == 0 {
		return strings.TrimSpace(decision.ClarificationQuestion)
	}
	return strings.TrimSpace(b.String())
}

func formatClarifierStepDetail(decision ClarifierDecision) string {
	parts := make([]string, 0, 5)
	if question := formatClarifierQuestion(decision); question != "" {
		parts = append(parts, question)
	}
	if normalizedQuestion := strings.TrimSpace(decision.NormalizedQuestion); normalizedQuestion != "" {
		parts = append(parts, "归一化问题："+normalizedQuestion)
	}
	if answerGoal := strings.TrimSpace(decision.AnswerGoal); answerGoal != "" {
		parts = append(parts, "回答目标："+answerGoal)
	}
	if ambiguityLevel := strings.TrimSpace(decision.AmbiguityLevel); ambiguityLevel != "" {
		parts = append(parts, "歧义程度："+ambiguityLevel)
	}
	if reason := strings.TrimSpace(decision.Reason); reason != "" {
		parts = append(parts, "判断依据："+reason)
	}
	return strings.Join(parts, "\n\n")
}

func compactNonEmptyStrings(items []string) []string {
	compacted := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		compacted = append(compacted, item)
	}
	return compacted
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
