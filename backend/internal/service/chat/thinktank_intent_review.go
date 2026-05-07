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

type rawAcceptanceReview struct {
	Verdict             string   `json:"verdict"`
	Score               *int     `json:"score"`
	MatchedDimensions   []string `json:"matched_dimensions"`
	MissingDimensions   []string `json:"missing_dimensions"`
	UnsupportedClaims   []string `json:"unsupported_claims"`
	FormatIssues        []string `json:"format_issues"`
	RevisionInstruction string   `json:"revision_instruction"`
	UserQuestion        string   `json:"user_question"`
	FollowUpQuestion    string   `json:"follow_up_question"`
	Reason              string   `json:"reason"`
	Summary             string   `json:"summary"`
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
	decision.TargetDimensions = compactNonEmptyStrings(decision.TargetDimensions)
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
	decision = applyResearchClarifierProfile(decision, originalQuestion)
	return decision
}

func defaultClarifierDecision(originalQuestion string) ClarifierDecision {
	question := strings.TrimSpace(originalQuestion)
	decision := ClarifierDecision{
		NormalizedQuestion: question,
		Intent:             question,
		AnswerGoal:         "explain",
		AmbiguityLevel:     "low",
		ShouldAskUser:      false,
		Reason:             "ClarifierAgent 未返回结构化结果，已按原问题继续回答。",
	}
	if isResearchReportQuestion(question) {
		decision.AnswerGoal = "research_report"
		decision.TargetDimensions = defaultResearchTargetDimensions(question)
		decision.Constraints.Depth = "深度调研"
		decision.Constraints.Style = "结构化调研报告"
		decision.Constraints.SourcePolicy = "优先使用可追溯来源，并标注证据限制"
		decision.Reason = "ClarifierAgent 未返回结构化结果，已按调研任务自动推断回答维度继续。"
	}
	return decision
}

func applyResearchClarifierProfile(decision ClarifierDecision, originalQuestion string) ClarifierDecision {
	if !isResearchReportQuestion(originalQuestion) && !isResearchReportQuestion(decision.NormalizedQuestion) {
		return decision
	}
	decision.AnswerGoal = "research_report"
	decision.TargetDimensions = mergeUniqueStrings(decision.TargetDimensions, defaultResearchTargetDimensions(originalQuestion))
	if decision.Constraints.Depth == "" {
		decision.Constraints.Depth = "深度调研"
	}
	if decision.Constraints.Style == "" {
		decision.Constraints.Style = "结构化调研报告"
	}
	if decision.Constraints.SourcePolicy == "" {
		decision.Constraints.SourcePolicy = "优先使用可追溯来源，并标注证据限制"
	}
	if strings.TrimSpace(decision.Reason) == "" {
		decision.Reason = "已按调研任务自动推断回答维度继续。"
	}
	return decision
}

func parseAcceptanceReview(raw string) AcceptanceReview {
	var parsed rawAcceptanceReview
	if err := json.Unmarshal([]byte(extractJSONObject(raw)), &parsed); err != nil {
		return defaultAcceptanceReview()
	}
	review := AcceptanceReview{
		Verdict:             strings.TrimSpace(parsed.Verdict),
		MatchedDimensions:   compactNonEmptyStrings(parsed.MatchedDimensions),
		MissingDimensions:   compactNonEmptyStrings(parsed.MissingDimensions),
		UnsupportedClaims:   compactNonEmptyStrings(parsed.UnsupportedClaims),
		FormatIssues:        compactNonEmptyStrings(parsed.FormatIssues),
		RevisionInstruction: strings.TrimSpace(parsed.RevisionInstruction),
		UserQuestion:        firstNonEmptyString(parsed.UserQuestion, parsed.FollowUpQuestion),
		Reason:              strings.TrimSpace(parsed.Reason),
		Summary:             strings.TrimSpace(parsed.Summary),
	}
	normalizedVerdict, validVerdict := parseAcceptanceVerdict(review.Verdict)
	if !hasMeaningfulAcceptanceReview(parsed, review, validVerdict) {
		return defaultAcceptanceReview()
	}
	review.Available = true
	review.Verdict = normalizedVerdict
	review.Score = clampAcceptanceScore(*parsed.Score)
	return review
}

func hasMeaningfulAcceptanceReview(raw rawAcceptanceReview, review AcceptanceReview, validVerdict bool) bool {
	if !validVerdict || raw.Score == nil || review.Summary == "" {
		return false
	}
	switch normalizeAcceptanceVerdict(review.Verdict) {
	case acceptanceVerdictRevise:
		return review.RevisionInstruction != ""
	case acceptanceVerdictAskUser:
		return review.UserQuestion != ""
	default:
		return true
	}
}

func clampAcceptanceScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
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
	normalized, ok := parseAcceptanceVerdict(verdict)
	if ok {
		return normalized
	}
	return acceptanceVerdictPass
}

func parseAcceptanceVerdict(verdict string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(verdict))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	switch normalized {
	case acceptanceVerdictPass:
		return acceptanceVerdictPass, true
	case acceptanceVerdictRevise:
		return acceptanceVerdictRevise, true
	case acceptanceVerdictAskUser:
		return acceptanceVerdictAskUser, true
	case "needs_revision", "needs_revise", "revise_answer":
		return acceptanceVerdictRevise, true
	default:
		return acceptanceVerdictPass, false
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
	if whyNeeded == "" {
		whyNeeded = defaultClarifierWhyNeeded(needSummary, missingDimensions)
	}
	suggestedReply := strings.TrimSpace(decision.SuggestedReply)
	if suggestedReply == "" {
		suggestedReply = defaultClarifierSuggestedReply(missingDimensions)
	}
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

func defaultClarifierWhyNeeded(needSummary string, missingDimensions []string) string {
	if len(missingDimensions) == 0 && strings.TrimSpace(needSummary) == "" {
		return ""
	}
	return "这些信息会直接影响回答范围、深度和可执行建议，补充后我可以按你的真实目标组织答案。"
}

func defaultClarifierSuggestedReply(missingDimensions []string) string {
	missingDimensions = compactNonEmptyStrings(missingDimensions)
	if len(missingDimensions) == 0 {
		return ""
	}
	return "请按上述维度逐项补充，例如：" + strings.Join(missingDimensions, "、") + "。"
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
	} else {
		parts = append(parts, "已覆盖：AcceptanceAgent 未返回具体覆盖维度")
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
	} else {
		parts = append(parts, "仍需注意：未发现明确缺失维度")
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
	} else {
		parts = append(parts, "已覆盖：AcceptanceAgent 未返回具体覆盖维度")
	}
	if missingDimensions := compactNonEmptyStrings(review.MissingDimensions); len(missingDimensions) > 0 {
		parts = append(parts, "缺失维度："+strings.Join(missingDimensions, "、"))
	} else {
		parts = append(parts, "缺失维度：未发现明确缺失维度")
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

func firstNonEmptyString(items ...string) string {
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			return trimmed
		}
	}
	return ""
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

func enforceAcceptanceQuality(review AcceptanceReview, input AcceptanceReviewInput) AcceptanceReview {
	if !review.Available || !isResearchReportQuestion(input.OriginalQuestion) {
		return review
	}
	missingDimensions := missingRequiredResearchDimensions(input.OriginalQuestion, input.Answer)
	if len(missingDimensions) == 0 {
		return review
	}

	review.Verdict = acceptanceVerdictRevise
	if review.Score == 0 || review.Score > 65 {
		review.Score = 65
	}
	review.MissingDimensions = mergeUniqueStrings(review.MissingDimensions, missingDimensions)
	if strings.TrimSpace(review.Summary) == "" || strings.Contains(review.Summary, "判定通过") || strings.Contains(review.Summary, "满足") {
		review.Summary = "调研报告仍缺少关键维度或深度不足，不能判定为充分完成用户调研需求。"
	}
	reason := "调研类问题需要覆盖核心事实、时间线、争议/法律风险、当前状态、影响分析和来源边界；当前答案缺少：" + strings.Join(missingDimensions, "、") + "。"
	if strings.TrimSpace(review.Reason) == "" {
		review.Reason = reason
	} else if !strings.Contains(review.Reason, reason) {
		review.Reason = strings.TrimSpace(review.Reason) + "\n" + reason
	}
	review.RevisionInstruction = "请重写为深度调研报告，补充：" + strings.Join(missingDimensions, "、") + "；每个关键结论尽量给出来源依据，并避免只用概述性表述。"
	return review
}

func isResearchReportQuestion(question string) bool {
	question = strings.TrimSpace(question)
	return containsAny(question, "调研", "研究一下", "研究报告", "调研报告", "深度研究")
}

type researchDimensionRule struct {
	name     string
	keywords []string
}

func defaultResearchTargetDimensions(question string) []string {
	if isTrumpQuestion(question) {
		return []string{
			"个人背景与商业经历",
			"政治生涯时间线",
			"政策主张与举措",
			"法律案件与争议",
			"当前身份与最新动态",
			"国内外影响与多方评价",
			"来源依据与证据限制",
		}
	}
	return []string{
		"背景与基本事实",
		"关键事件时间线",
		"核心观点或贡献",
		"争议与证据限制",
		"影响分析",
		"参考来源",
	}
}

func missingRequiredResearchDimensions(question string, answer string) []string {
	if !isResearchReportQuestion(question) {
		return nil
	}
	answer = strings.TrimSpace(answer)
	rules := defaultResearchDimensionRules(question)
	missing := make([]string, 0, len(rules))
	for _, rule := range rules {
		if !containsAny(answer, rule.keywords...) {
			missing = append(missing, rule.name)
		}
	}
	return missing
}

func defaultResearchDimensionRules(question string) []researchDimensionRule {
	if isTrumpQuestion(question) {
		return []researchDimensionRule{
			{name: "个人背景与商业经历", keywords: []string{"商业经历", "房地产", "特朗普集团", "媒体人物", "电视节目", "品牌授权"}},
			{name: "政治生涯时间线", keywords: []string{"政治生涯", "时间线", "2016", "2020", "2024", "第47任", "第二任期", "竞选历程"}},
			{name: "政策主张与举措", keywords: []string{"移民政策", "贸易政策", "关税", "税改", "外交政策", "能源政策", "司法任命"}},
			{name: "法律案件与争议", keywords: []string{"法律案件", "刑事", "民事", "诉讼", "弹劾", "定罪", "争议"}},
			{name: "当前身份与最新动态", keywords: []string{"第47任", "现任", "当前", "第二任期", "2025", "2026", "JD Vance", "万斯"}},
			{name: "国内外影响与多方评价", keywords: []string{"支持者", "批评者", "两极化", "国内影响", "国际影响", "多方评价", "反对者"}},
			{name: "来源依据与证据限制", keywords: []string{"参考", "来源", "证据", "链接", "资料限制", "检索限制"}},
		}
	}
	return []researchDimensionRule{
		{name: "背景与基本事实", keywords: []string{"背景", "基本信息", "概述", "简介"}},
		{name: "关键事件时间线", keywords: []string{"时间线", "阶段", "历程", "关键事件"}},
		{name: "核心观点或贡献", keywords: []string{"核心", "贡献", "主张", "关键事实"}},
		{name: "争议与证据限制", keywords: []string{"争议", "限制", "不足", "风险", "证据"}},
		{name: "影响分析", keywords: []string{"影响", "分析", "意义", "评价"}},
		{name: "参考来源", keywords: []string{"参考", "来源", "链接"}},
	}
}

func isTrumpQuestion(question string) bool {
	return containsAny(question, "特朗普", "川普", "Donald Trump", "Trump")
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func mergeUniqueStrings(base []string, extra []string) []string {
	result := compactNonEmptyStrings(base)
	seen := make(map[string]struct{}, len(result)+len(extra))
	for _, item := range result {
		seen[item] = struct{}{}
	}
	for _, item := range compactNonEmptyStrings(extra) {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}
