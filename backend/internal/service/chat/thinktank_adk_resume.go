package chat

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"

	"wenDao/internal/model"
)

type adkPendingContext struct {
	Type             string `json:"type"`
	Checkpoint       string `json:"checkpoint_id,omitempty"`
	OriginalQuestion string `json:"original_question,omitempty"`
	SystemQuestion   string `json:"system_question,omitempty"`
}

func buildADKCheckpointID(conv *model.Conversation, question string) string {
	conversationID := int64(0)
	if conv != nil {
		conversationID = conv.ID
	}
	return fmt.Sprintf("thinktank-%d-%d-%d", conversationID, time.Now().UnixNano(), len([]rune(question)))
}

func marshalADKPendingContext(checkpointID string, contextParts ...string) string {
	payload := adkPendingContext{
		Type:       "adk_interrupt",
		Checkpoint: checkpointID,
	}
	if len(contextParts) > 0 {
		payload.OriginalQuestion = strings.TrimSpace(contextParts[0])
	}
	if len(contextParts) > 1 {
		payload.SystemQuestion = strings.TrimSpace(contextParts[1])
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(data)
}

func marshalAgentPendingContext(contextType string, originalQuestion string, systemQuestion string) string {
	payload, err := json.Marshal(adkPendingContext{
		Type:             contextType,
		OriginalQuestion: strings.TrimSpace(originalQuestion),
		SystemQuestion:   strings.TrimSpace(systemQuestion),
	})
	if err != nil {
		return ""
	}
	return string(payload)
}

func parseADKPendingContext(run *model.ConversationRun) (adkPendingContext, bool) {
	if run == nil || run.Status != "waiting_user" || strings.TrimSpace(run.PendingContext) == "" {
		return adkPendingContext{}, false
	}
	var payload adkPendingContext
	if err := json.Unmarshal([]byte(run.PendingContext), &payload); err != nil {
		return adkPendingContext{}, false
	}
	if payload.Type != "adk_interrupt" || strings.TrimSpace(payload.Checkpoint) == "" {
		return adkPendingContext{}, false
	}
	payload.OriginalQuestion = strings.TrimSpace(payload.OriginalQuestion)
	if payload.OriginalQuestion == "" {
		payload.OriginalQuestion = strings.TrimSpace(run.OriginalQuestion)
	}
	payload.SystemQuestion = strings.TrimSpace(payload.SystemQuestion)
	if payload.SystemQuestion == "" && run.PendingQuestion != nil {
		payload.SystemQuestion = strings.TrimSpace(*run.PendingQuestion)
	}
	return payload, true
}

func parseAgentPendingContext(run *model.ConversationRun) (adkPendingContext, bool) {
	if run == nil || run.Status != "waiting_user" || strings.TrimSpace(run.PendingContext) == "" {
		return adkPendingContext{}, false
	}
	var payload adkPendingContext
	if err := json.Unmarshal([]byte(run.PendingContext), &payload); err != nil {
		return adkPendingContext{}, false
	}
	if payload.Type != "clarifier_interrupt" && payload.Type != "acceptance_interrupt" {
		return adkPendingContext{}, false
	}
	payload.OriginalQuestion = strings.TrimSpace(payload.OriginalQuestion)
	if payload.OriginalQuestion == "" {
		payload.OriginalQuestion = strings.TrimSpace(run.OriginalQuestion)
	}
	payload.SystemQuestion = strings.TrimSpace(payload.SystemQuestion)
	if payload.SystemQuestion == "" && run.PendingQuestion != nil {
		payload.SystemQuestion = strings.TrimSpace(*run.PendingQuestion)
	}
	if payload.OriginalQuestion == "" || payload.SystemQuestion == "" {
		return adkPendingContext{}, false
	}
	return payload, true
}

func buildInterruptedFollowUpQuestion(originalQuestion string, systemQuestion string, userSupplement string) string {
	parts := []string{
		"原始问题：\n" + strings.TrimSpace(originalQuestion),
		"系统追问：\n" + strings.TrimSpace(systemQuestion),
		"用户补充：\n" + strings.TrimSpace(userSupplement),
	}
	return strings.Join(parts, "\n\n")
}

func extractADKClarificationQuestion(info *adk.InterruptInfo) string {
	if info == nil {
		return ""
	}
	// Eino 的中断 Data 可能包含完整 checkpoint、OrigInput、嵌套子图状态等内部结构；
	// ask_for_clarification 真正要展示给用户的问题在 root-cause InterruptContext.Info 中。
	for _, ctx := range info.InterruptContexts {
		if ctx == nil {
			continue
		}
		if ctx.IsRootCause {
			if text := interruptInfoToUserText(ctx.Info); text != "" {
				return text
			}
		}
	}
	for _, ctx := range info.InterruptContexts {
		if ctx == nil {
			continue
		}
		if text := interruptInfoToUserText(ctx.Info); text != "" {
			return text
		}
	}
	return interruptInfoToUserText(info.Data)
}

func interruptInfoToUserText(info any) string {
	switch v := info.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return ""
	}
}

func extractPlanExecuteFinalResponse(content string) (string, bool) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", false
	}
	var payload struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return "", false
	}
	response := strings.TrimSpace(payload.Response)
	if response == "" {
		return "", false
	}
	return response, true
}

func isNonFinalToolLimitationAnswer(response string) bool {
	response = strings.TrimSpace(response)
	if response == "" {
		return false
	}
	markers := []string{
		"DocParser",
		"当前工具列表中无",
		"工具列表中无",
		"无法完成解析 HTML",
		"无法完成解析HTML",
		"请提供其他可行的工具",
		"missing tool",
		"tool is missing",
		"unavailable tool",
	}
	for _, marker := range markers {
		if strings.Contains(response, marker) {
			return true
		}
	}
	return false
}
