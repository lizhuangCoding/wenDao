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
	Summary             string   `json:"summary"`
	Available           bool     `json:"-"`
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
	if decision.ShouldAskUser && len(decision.MissingDimensions) == 0 && decision.ClarificationQuestion == "" {
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
	review.Available = true
	review.Verdict = normalizeAcceptanceVerdict(review.Verdict)
	review.MatchedDimensions = compactNonEmptyStrings(review.MatchedDimensions)
	review.MissingDimensions = compactNonEmptyStrings(review.MissingDimensions)
	review.UnsupportedClaims = compactNonEmptyStrings(review.UnsupportedClaims)
	review.FormatIssues = compactNonEmptyStrings(review.FormatIssues)
	review.Reason = strings.TrimSpace(review.Reason)
	review.Summary = strings.TrimSpace(review.Summary)
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
		Verdict:   acceptanceVerdictPass,
		Score:     0,
		Reason:    "acceptance output unavailable; returning generated answer",
		Available: false,
	}
}

func normalizeAcceptanceVerdict(verdict string) string {
	normalized := strings.ToLower(strings.TrimSpace(verdict))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	switch normalized {
	case acceptanceVerdictRevise:
		return acceptanceVerdictRevise
	case acceptanceVerdictAskUser:
		return acceptanceVerdictAskUser
	case "needs_revision", "needs_revise", "revise_answer":
		return acceptanceVerdictRevise
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
	whyNeeded := strings.TrimSpace(decision.WhyNeeded)
	suggestedReply := strings.TrimSpace(decision.SuggestedReply)
	if len(missingDimensions) == 0 && whyNeeded == "" && suggestedReply == "" {
		return ""
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
	if whyNeeded != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("为什么需要这些信息：\n")
		b.WriteString(whyNeeded)
	}
	if suggestedReply != "" {
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
	parts := make([]string, 0, 4)
	if needSummary := strings.TrimSpace(decision.NeedSummary); needSummary != "" {
		parts = append(parts, "实际需求："+needSummary)
	} else if intent := strings.TrimSpace(decision.Intent); intent != "" {
		parts = append(parts, "实际需求："+intent)
	} else if normalizedQuestion := strings.TrimSpace(decision.NormalizedQuestion); normalizedQuestion != "" {
		parts = append(parts, "实际需求："+normalizedQuestion)
	}
	if targetDimensions := compactNonEmptyStrings(decision.TargetDimensions); len(targetDimensions) > 0 {
		parts = append(parts, "回答维度："+strings.Join(targetDimensions, "、"))
	} else if missingDimensions := compactNonEmptyStrings(decision.MissingDimensions); len(missingDimensions) > 0 {
		parts = append(parts, "回答维度："+strings.Join(missingDimensions, "、"))
	}
	if decision.ShouldAskUser {
		parts = append(parts, "处理方式：需要用户补充关键维度")
	} else {
		parts = append(parts, "处理方式：无需追问，按推断维度继续调研")
	}
	if reason := strings.TrimSpace(decision.Reason); reason != "" {
		parts = append(parts, "原因："+reason)
	}
	return strings.Join(parts, "\n\n")
}

func appendAcceptanceSummary(answer string, review AcceptanceReview, revised bool) string {
	answer = strings.TrimSpace(answer)
	if answer == "" || !review.Available {
		return answer
	}

	score := review.Score
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	verdictText := "通过"
	if revised {
		verdictText = "初稿需要修订，已自动补充关键缺失项"
	} else {
		switch normalizeAcceptanceVerdict(review.Verdict) {
		case acceptanceVerdictRevise:
			verdictText = "需要修订"
		case acceptanceVerdictAskUser:
			verdictText = "需要补充信息"
		}
	}

	parts := []string{"验收摘要：" + verdictText + "，评分 " + strconv.Itoa(score) + "/100"}
	if matchedDimensions := compactNonEmptyStrings(review.MatchedDimensions); len(matchedDimensions) > 0 {
		parts = append(parts, "已覆盖："+strings.Join(matchedDimensions, "、"))
	}
	if revised {
		if revisionInstruction := strings.TrimSpace(review.RevisionInstruction); revisionInstruction != "" {
			parts = append(parts, "修订重点："+revisionInstruction)
		}
	}
	summary := strings.TrimSpace(review.Summary)
	reason := strings.TrimSpace(review.Reason)
	switch {
	case summary != "" && reason != "":
		parts = append(parts, "结论："+summary+"\n"+reason)
	case summary != "":
		parts = append(parts, "结论："+summary)
	case reason != "":
		parts = append(parts, "结论："+reason)
	}
	if missingDimensions := compactNonEmptyStrings(review.MissingDimensions); len(missingDimensions) > 0 {
		parts = append(parts, "仍需注意："+strings.Join(missingDimensions, "、"))
	}
	if unsupportedClaims := compactNonEmptyStrings(review.UnsupportedClaims); len(unsupportedClaims) > 0 {
		parts = append(parts, "证据限制："+strings.Join(unsupportedClaims, "、"))
	}

	return answer + "\n\n" + strings.Join(parts, "\n")
}

func formatAcceptanceQuestion(review AcceptanceReview) string {
	question := strings.TrimSpace(review.UserQuestion)
	if question == "" {
		question = "请补充缺失的关键信息后，我再继续完善答案。"
	}

	parts := []string{"验收时发现还缺少一个关键信息：", question}
	if reason := strings.TrimSpace(review.Reason); reason != "" {
		parts = append(parts, "为什么需要：", reason)
	}
	return strings.Join(parts, "\n\n")
}

func formatAcceptanceStepDetail(review AcceptanceReview, revised bool) string {
	if !review.Available {
		return "AcceptanceAgent 未返回可用验收结果，已直接返回生成答案。"
	}

	score := review.Score
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	parts := []string{
		"验收结论：" + acceptanceVerdictText(review.Verdict),
		"评分：" + strconv.Itoa(score) + "/100",
	}
	if matchedDimensions := compactNonEmptyStrings(review.MatchedDimensions); len(matchedDimensions) > 0 {
		parts = append(parts, "已覆盖："+strings.Join(matchedDimensions, "、"))
	}
	if missingDimensions := compactNonEmptyStrings(review.MissingDimensions); len(missingDimensions) > 0 {
		parts = append(parts, "缺失维度："+strings.Join(missingDimensions, "、"))
	}
	if revised {
		parts = append(parts, "处理方式：初稿需要修订，已自动补充关键缺失项")
	} else if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictRevise {
		parts = append(parts, "处理方式：需要返工修订")
	} else if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictAskUser {
		parts = append(parts, "处理方式：需要用户补充关键信息")
	} else {
		parts = append(parts, "处理方式：验收通过")
	}
	if reason := strings.TrimSpace(review.Reason); reason != "" {
		parts = append(parts, "原因："+reason)
	}
	return strings.Join(parts, "\n\n")
}

func acceptanceVerdictText(verdict string) string {
	switch normalizeAcceptanceVerdict(verdict) {
	case acceptanceVerdictRevise:
		return "需要修订"
	case acceptanceVerdictAskUser:
		return "需要补充信息"
	default:
		return "通过"
	}
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
