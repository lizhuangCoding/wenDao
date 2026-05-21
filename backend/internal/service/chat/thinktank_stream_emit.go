package chat

import "wenDao/internal/model"

func adkAgentStepMetadata(agentName string) (string, string) {
	switch agentName {
	case "planner":
		return "正在生成完整任务计划", "Eino Planner 正在规划"
	case "executor":
		return "正在执行当前计划步骤", "Eino Executor 正在执行"
	case "replanner":
		return "正在评估结果并重规划", "Eino Replanner 正在评估"
	default:
		return "Agent " + agentName + " 正在协作", "切换为 " + agentName + " Agent"
	}
}

func (o *thinkTankOrchestrator) emitResume(eventCh chan<- StreamEvent, conv *model.Conversation, runID int64, stage string, status string) {
	o.service.streams.emitResume(eventCh, runID, stage, status)
	if conv != nil && runID > 0 {
		o.service.runHub.publish(runID, conv.ID, StreamEvent{Type: StreamEventResume, RunID: runID, Stage: stage, Status: status})
		o.service.runs.updateProgress(runID, stage, "")
	}
}

func (o *thinkTankOrchestrator) emitSnapshot(eventCh chan<- StreamEvent, conv *model.Conversation, runID int64, stage string, status string, message string) {
	o.service.streams.emitSnapshot(eventCh, runID, stage, status, message)
	if conv != nil && runID > 0 {
		o.service.runHub.publish(runID, conv.ID, StreamEvent{Type: StreamEventSnapshot, RunID: runID, Stage: stage, Status: status, Message: message})
		o.service.runs.updateProgress(runID, stage, message)
	}
}

func (o *thinkTankOrchestrator) emitStage(eventCh chan<- StreamEvent, conv *model.Conversation, runID int64, stage string, label string) {
	o.service.streams.emitStage(eventCh, stage, label)
	if conv != nil && runID > 0 {
		o.service.runHub.publish(runID, conv.ID, StreamEvent{Type: StreamEventStage, RunID: runID, Stage: stage, Label: label, Status: "running"})
		o.service.runs.updateProgress(runID, stage, "")
	}
}

func (o *thinkTankOrchestrator) emitQuestion(eventCh chan<- StreamEvent, conv *model.Conversation, runID int64, stage string, question string) {
	o.service.streams.emitQuestion(eventCh, stage, question)
	if conv != nil && runID > 0 {
		o.service.runHub.publish(runID, conv.ID, StreamEvent{Type: StreamEventQuestion, RunID: runID, Stage: stage, Message: question, Status: "waiting_user"})
		o.service.runs.updateProgress(runID, stage, question)
	}
}

func (o *thinkTankOrchestrator) emitChunk(eventCh chan<- StreamEvent, conv *model.Conversation, runID int64, message string, sources []string) {
	o.service.streams.emitChunk(eventCh, message, sources)
	if conv != nil && runID > 0 {
		o.service.runHub.publish(runID, conv.ID, StreamEvent{Type: StreamEventChunk, RunID: runID, Stage: "streaming", Message: message, Sources: sources, Status: "running"})
		o.service.runs.updateProgress(runID, "streaming", message)
	}
}

func (o *thinkTankOrchestrator) emitStep(eventCh chan<- StreamEvent, conv *model.Conversation, runID int64, step *model.ConversationRunStep) {
	o.service.streams.emitStep(eventCh, step)
	if conv != nil && runID > 0 && step != nil {
		o.service.runHub.publish(runID, conv.ID, StreamEvent{
			Type:      StreamEventStep,
			RunID:     runID,
			Stage:     step.Type,
			Status:    step.Status,
			StepID:    step.ID,
			AgentName: step.AgentName,
			Summary:   step.Summary,
			Detail:    step.Detail,
		})
	}
}

func (o *thinkTankOrchestrator) emitDone(eventCh chan<- StreamEvent, conv *model.Conversation, runID int64, stage string, label string) {
	o.service.streams.emitDone(eventCh, stage, label)
	if conv != nil && runID > 0 {
		o.service.runHub.publish(runID, conv.ID, StreamEvent{Type: StreamEventDone, RunID: runID, Stage: stage, Status: "completed"})
		o.service.runHub.finish(runID, "completed")
	}
}
