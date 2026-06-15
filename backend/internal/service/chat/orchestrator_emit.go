package chat

import (
	"strings"

	"wenDao/internal/model"
)

func (o *thinkTankOrchestrator) emitAcceptanceQuestion(eventCh chan<- StreamEvent, conv *model.Conversation, runID int64, userID int64, question string, review AcceptanceReview, decision PlannerDecision) {
	userQuestion := strings.TrimSpace(formatAcceptanceQuestion(review))
	if userQuestion == "" {
		userQuestion = "还需要你补充一点信息，我才能继续。"
	}
	if conv != nil {
		pendingContext := marshalAgentPendingContext("acceptance_interrupt", question, userQuestion)
		o.service.runs.persistAgentClarification(conv.ID, userID, runID, question, userQuestion, "clarifying", pendingContext, decision)
		o.service.conversations.saveMessageWithWarning(conv.ID, "assistant", userQuestion, "Failed to save acceptance clarification message", runID)
	}
	o.emitStage(eventCh, conv, runID, "clarifying", "需要补充一点信息")
	o.emitQuestion(eventCh, conv, runID, "clarifying", userQuestion)
}

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
