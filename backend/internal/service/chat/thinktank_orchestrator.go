package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"wenDao/internal/model"
)

const thinkTankStreamRunTimeout = 15 * time.Minute

type thinkTankOrchestrator struct {
	service *thinkTankService
}

type thinkTankResearchDraftSink struct {
	knowledgeSvc KnowledgeDocumentService
}

func newThinkTankOrchestrator(service *thinkTankService) *thinkTankOrchestrator {
	return &thinkTankOrchestrator{service: service}
}

func newThinkTankResearchDraftSink(knowledgeSvc KnowledgeDocumentService) *thinkTankResearchDraftSink {
	return &thinkTankResearchDraftSink{knowledgeSvc: knowledgeSvc}
}

func (s *thinkTankResearchDraftSink) saveFromJournalist(userID int64, result *JournalistResult) {
	if s == nil || s.knowledgeSvc == nil || result == nil || strings.TrimSpace(result.KnowledgeDraftBody) == "" {
		return
	}
	sources := make([]KnowledgeSourceInput, 0, len(result.Sources))
	for _, source := range result.Sources {
		sources = append(sources, KnowledgeSourceInput{URL: source.URL, Title: source.Title})
	}
	_, _ = s.knowledgeSvc.CreateResearchDraft(CreateKnowledgeDocumentInput{
		Title:           result.KnowledgeDraftTitle,
		Summary:         result.KnowledgeDraftSummary,
		Content:         result.KnowledgeDraftBody,
		CreatedByUserID: userID,
		Sources:         sources,
	})
}

func (o *thinkTankOrchestrator) chat(ctx context.Context, question string, conversationID *int64, userID *int64) (*ThinkTankChatResponse, error) {
	s := o.service
	conv, err := s.conversations.getOwnedConversation(conversationID, userID)
	if err != nil {
		return nil, err
	}

	var history []model.ChatMessage
	if conv != nil {
		history = s.conversations.loadHistory(conv.ID)
		s.conversations.saveMessageWithWarning(conv.ID, "user", question, "Failed to save user message")
	}

	decision := PlannerDecision{ExecutionStrategy: "eino_plan_execute_replan", PlanSummary: "由 Eino PlanExecute planner 生成计划"}
	queryForAgents := o.buildAgentQuery(question, conv, history)

	if s.adkRunner != nil && s.adkAnswerFetcher != nil {
		adkCtx := WithUserID(ctx, derefUserID(userID))
		adkCtx = WithAILogger(adkCtx, s.logger)
		if conv != nil {
			adkCtx = WithConversationID(adkCtx, conv.ID)
		}
		answer, err := s.adkAnswerFetcher(adkCtx, queryForAgents)
		if err == nil && strings.TrimSpace(answer) != "" {
			o.persistFinalAnswer(conv, derefUserID(userID), question, answer, decision, history)
			return &ThinkTankChatResponse{Message: answer, Stage: "completed"}, nil
		}
	}

	localResult, err := s.librarian.Search(ctx, queryForAgents)
	if err != nil {
		return nil, err
	}

	var webResult *JournalistResult
	if localResult.CoverageStatus != "sufficient" && s.journalist != nil {
		webResult, err = s.journalist.Research(ctx, queryForAgents, localResult)
		if err != nil {
			return nil, err
		}
		s.researchDraft.saveFromJournalist(derefUserID(userID), webResult)
	}

	answer, sources, err := s.synthesizer.Compose(ctx, queryForAgents, localResult, webResult)
	if err != nil {
		return nil, err
	}

	o.persistFinalAnswer(conv, derefUserID(userID), question, answer, decision, history)
	return &ThinkTankChatResponse{Message: answer, Sources: sources, Stage: "completed"}, nil
}

func (o *thinkTankOrchestrator) chatStream(ctx context.Context, question string, conversationID *int64, userID *int64) (<-chan StreamEvent, <-chan error) {
	eventCh := make(chan StreamEvent, 48)
	errCh := make(chan error, 1)
	go func() {
		defer close(eventCh)
		defer close(errCh)

		runCtx, cancel := context.WithTimeout(context.Background(), thinkTankStreamRunTimeout)
		defer cancel()

		s := o.service
		conv, err := s.conversations.getOwnedConversation(conversationID, userID)
		if err != nil {
			errCh <- err
			return
		}

		var history []model.ChatMessage
		var pending *model.ConversationRun
		if conv != nil {
			history = s.conversations.loadHistory(conv.ID)
			pending = s.runs.activeRun(conv.ID)
			s.conversations.saveMessageWithWarning(conv.ID, "user", question, "Failed to save user message")
		}

		o.emitStage(eventCh, conv, 0, "analyzing", "正在理解你的问题")
		decision := PlannerDecision{ExecutionStrategy: "eino_plan_execute_replan", PlanSummary: "由 Eino PlanExecute planner 生成计划"}
		o.emitStage(eventCh, conv, 0, "analyzing", "正在进行多 Agent 深度调研")
		s.runs.logStage(conv, userID, "adk_start", "开始多 Agent 协作流", question)

		queryForAgents := o.buildAgentQuery(question, conv, history)
		runID, checkpointID, resumeFromADKInterrupt := o.prepareADKRun(conv, pending, userID, question, decision)
		o.emitResume(eventCh, conv, runID, "analyzing", "running")
		o.emitSnapshot(eventCh, conv, runID, "analyzing", "running", "")

		if s.adkRunner != nil && s.adkRunner.runner != nil {
			if err := o.streamADKFlow(runCtx, eventCh, errCh, conv, history, question, userID, queryForAgents, decision, checkpointID, runID, resumeFromADKInterrupt); err != nil {
				return
			}
			return
		}

		o.streamManualFlow(runCtx, eventCh, errCh, conv, history, question, userID, queryForAgents, decision, runID)
	}()
	return eventCh, errCh
}

func (o *thinkTankOrchestrator) resumeChatStream(ctx context.Context, conversationID int64, runID int64, userID *int64) (<-chan StreamEvent, <-chan error) {
	eventCh := make(chan StreamEvent, 48)
	errCh := make(chan error, 1)
	go func() {
		defer close(eventCh)
		defer close(errCh)

		s := o.service
		conv, err := s.conversations.getOwnedConversation(&conversationID, userID)
		if err != nil {
			errCh <- err
			return
		}
		if conv == nil {
			errCh <- errors.New("conversation not found")
			return
		}

		run, err := s.runs.runRepo.GetByID(runID)
		if err != nil || run == nil || run.ConversationID != conversationID || run.UserID != derefUserID(userID) {
			errCh <- errors.New("run not found")
			return
		}

		s.streams.emitResume(eventCh, run.ID, run.CurrentStage, run.Status)
		if snapshot, ok := s.runHub.snapshot(run.ID); ok {
			s.streams.emitSnapshot(eventCh, run.ID, snapshot.Stage, snapshot.Status, snapshot.Message)
			for _, step := range snapshot.Steps {
				step := step
				s.streams.emitStep(eventCh, &step)
			}
			if o.emitTerminalResumeState(eventCh, errCh, snapshot.Status, snapshot.PendingQuestion, "") {
				return
			}
			if run.Status == "running" {
				sub, cancel, ok := s.runHub.subscribe(run.ID)
				if !ok {
					errCh <- errors.New("运行记录正在恢复，但后台任务已经不在当前进程中")
					return
				}
				defer cancel()
				if latest, ok := s.runHub.snapshot(run.ID); ok {
					if o.emitTerminalResumeState(eventCh, errCh, latest.Status, latest.PendingQuestion, "") {
						return
					}
				}
				for {
					select {
					case <-ctx.Done():
						return
					case event, ok := <-sub:
						if !ok {
							return
						}
						select {
						case <-ctx.Done():
							return
						case eventCh <- event:
						}
						if event.Type == StreamEventDone {
							return
						}
					}
				}
			}
		}

		s.streams.emitSnapshot(eventCh, run.ID, run.CurrentStage, run.Status, run.LastAnswer)
		steps, _ := s.runs.runStepRepo.GetByRunID(run.ID)
		for _, step := range steps {
			step := step
			s.streams.emitStep(eventCh, &step)
		}
		switch run.Status {
		case "running":
			errCh <- errors.New("后台任务已经断开，请重新发送问题")
		case "waiting_user":
			pendingQuestion := ""
			if run.PendingQuestion != nil {
				pendingQuestion = *run.PendingQuestion
			}
			o.emitTerminalResumeState(eventCh, errCh, run.Status, pendingQuestion, "")
		case "completed":
			o.emitTerminalResumeState(eventCh, errCh, run.Status, "", "")
		case "failed":
			message := "本次执行失败"
			if run.LastError != nil && strings.TrimSpace(*run.LastError) != "" {
				message = *run.LastError
			}
			o.emitTerminalResumeState(eventCh, errCh, run.Status, "", message)
		}
	}()
	return eventCh, errCh
}

func (o *thinkTankOrchestrator) emitTerminalResumeState(eventCh chan<- StreamEvent, errCh chan<- error, status string, pendingQuestion string, errorMessage string) bool {
	switch status {
	case "waiting_user":
		if strings.TrimSpace(pendingQuestion) != "" {
			o.service.streams.emitQuestion(eventCh, "clarifying", pendingQuestion)
		}
		return true
	case "completed":
		o.service.streams.emitDone(eventCh, "completed", "回答已生成")
		return true
	case "failed":
		message := strings.TrimSpace(errorMessage)
		if message == "" {
			message = "本次执行失败"
		}
		errCh <- errors.New(message)
		return true
	default:
		return false
	}
}

func (o *thinkTankOrchestrator) buildAgentQuery(question string, conv *model.Conversation, history []model.ChatMessage) string {
	memory := compressConversationMemory(history)
	if conv != nil {
		memory = buildConversationMemoryForQuestion(question, history, o.service.memories.loadConversationMemories(conv.ID))
	}
	return buildAgentQuery(question, memory)
}

func (o *thinkTankOrchestrator) persistFinalAnswer(conv *model.Conversation, userID int64, question string, answer string, decision PlannerDecision, history []model.ChatMessage) {
	if conv == nil || strings.TrimSpace(answer) == "" {
		return
	}
	o.service.conversations.persistAssistantTurn(conv, question, answer)
	o.service.runs.persistCompletedRun(conv.ID, userID, question, answer, decision)
	o.service.memories.updateConversationMemoryWithWarning(conv.ID, userID, appendConversationTurn(history, question, answer))
}

func (o *thinkTankOrchestrator) prepareADKRun(conv *model.Conversation, pending *model.ConversationRun, userID *int64, question string, decision PlannerDecision) (int64, string, bool) {
	checkpointID := buildADKCheckpointID(conv, question)
	resumeFromADKInterrupt := false
	if ctxInfo, ok := parseADKPendingContext(pending); ok && strings.TrimSpace(ctxInfo.Checkpoint) != "" {
		resumeFromADKInterrupt = true
		checkpointID = ctxInfo.Checkpoint
	}
	if conv == nil {
		return 0, checkpointID, resumeFromADKInterrupt
	}
	runID := o.service.runs.startADKRun(conv.ID, derefUserID(userID), question, decision, checkpointID, pending)
	return runID, checkpointID, resumeFromADKInterrupt
}

func (o *thinkTankOrchestrator) streamADKFlow(
	ctx context.Context,
	eventCh chan<- StreamEvent,
	errCh chan<- error,
	conv *model.Conversation,
	history []model.ChatMessage,
	question string,
	userID *int64,
	queryForAgents string,
	decision PlannerDecision,
	checkpointID string,
	runID int64,
	resumeFromADKInterrupt bool,
) error {
	s := o.service
	adkCtx := WithUserID(ctx, derefUserID(userID))
	adkCtx = WithAILogger(adkCtx, s.logger)
	adkCtx = WithRunID(adkCtx, runID)
	if conv != nil {
		adkCtx = WithConversationID(adkCtx, conv.ID)
	}

	var adkIter *adk.AsyncIterator[*adk.AgentEvent]
	if resumeFromADKInterrupt {
		resumeIter, resumeErr := s.adkRunner.runner.Resume(adkCtx, checkpointID, adk.WithToolOptions([]tool.Option{WithNewInput(question)}))
		if resumeErr != nil {
			errCh <- resumeErr
			return resumeErr
		}
		adkIter = resumeIter
	} else {
		adkIter = s.adkRunner.runner.Run(adkCtx, []adk.Message{schema.UserMessage(queryForAgents)}, adk.WithCheckPointID(checkpointID))
	}

	var currentStep *thinkTankStepTracker
	conversationID := int64(0)
	if conv != nil {
		conversationID = conv.ID
	}
	adkArticleSources := make([]SourceRef, 0)
	adkWebSources := make([]SourceRef, 0)
	adkLocalNotes := make([]string, 0)
	adkWebNotes := make([]string, 0)
	fullAnswer := ""

	for {
		e, ok := adkIter.Next()
		if !ok {
			break
		}
		if e == nil {
			continue
		}
		if e.Err != nil {
			if currentStep != nil {
				currentStep.fail("Error: " + e.Err.Error())
				o.emitStep(eventCh, conv, runID, currentStep.snapshot())
			}
			s.runs.logStage(conv, userID, "failed", "ADK 运行错误", e.Err.Error())
			errCh <- e.Err
			return e.Err
		}

		if e.Action != nil && e.Action.Interrupted != nil {
			clarification := extractADKClarificationQuestion(e.Action.Interrupted)
			if strings.TrimSpace(clarification) == "" {
				clarification = "还需要你补充一点信息，我才能继续。"
			}
			if currentStep != nil {
				currentStep.setStatus("waiting_user")
				currentStep.appendDetail("流程中断：ask_for_clarification\n问题：" + clarification)
				o.emitStep(eventCh, conv, runID, currentStep.snapshot())
			}
			if conv != nil {
				s.runs.persistADKClarification(conv.ID, derefUserID(userID), runID, question, clarification, checkpointID, decision)
				s.conversations.saveMessageWithWarning(conv.ID, "assistant", clarification, "Failed to save clarification message")
			}
			o.emitStage(eventCh, conv, runID, "clarifying", "需要补充一点信息")
			o.emitQuestion(eventCh, conv, runID, "clarifying", clarification)
			return nil
		}

		if e.AgentName != "" && (currentStep == nil || currentStep.step.AgentName != e.AgentName) {
			if currentStep != nil {
				currentStep.complete()
				o.emitStep(eventCh, conv, runID, currentStep.snapshot())
			}
			summary, label := adkAgentStepMetadata(e.AgentName)
			if label != "" {
				o.emitStage(eventCh, conv, runID, "adk_event", label)
			}
			currentStep = s.runs.newStepTracker(conversationID, runID, e.AgentName, summary)
			o.emitStep(eventCh, conv, runID, currentStep.snapshot())
		}

		if actionDetail := formatADKActionDetail(e.Action); actionDetail != "" && currentStep != nil {
			currentStep.appendDetail(actionDetail)
			o.emitStep(eventCh, conv, runID, currentStep.snapshot())
		}

		msg, _, err := adk.GetMessage(e)
		if err != nil || msg == nil {
			continue
		}

		detail := formatADKMessageDetail(msg)
		if strings.TrimSpace(msg.ToolName) == "LocalSearch" {
			adkArticleSources = mergeSourceRefs(adkArticleSources, extractLocalSearchArticleSources(msg.Content))
			adkLocalNotes = appendNonEmptyNote(adkLocalNotes, extractLocalSearchSummary(msg.Content))
		}
		if strings.TrimSpace(msg.ToolName) == "WebSearch" {
			adkWebSources = mergeSourceRefs(adkWebSources, extractWebSearchSources(msg.Content))
			adkWebNotes = appendNonEmptyNote(adkWebNotes, summarizeWebSearchResult(msg.Content))
		}
		if e.AgentName == "executor" && strings.TrimSpace(msg.ToolName) == "" && len(msg.ToolCalls) == 0 {
			adkWebNotes = appendNonEmptyNote(adkWebNotes, msg.Content)
		}
		if currentStep != nil && detail != "" {
			currentStep.appendDetail(detail)
			o.emitStep(eventCh, conv, runID, currentStep.snapshot())
		}

		if e.AgentName == "replanner" && strings.TrimSpace(msg.Content) != "" {
			if response, ok := extractPlanExecuteFinalResponse(msg.Content); ok {
				if isNonFinalToolLimitationAnswer(response) {
					adkWebNotes = appendNonEmptyNote(adkWebNotes, "replanner returned a tool limitation instead of a user-facing answer: "+response)
				} else {
					fullAnswer = appendGroupedReferences(response, adkArticleSources, adkWebSources)
					o.emitChunk(eventCh, conv, runID, fullAnswer, nil)
				}
			}
		}
	}

	if strings.TrimSpace(fullAnswer) == "" {
		answer, fallbackErr := s.composeADKFallbackAnswer(ctx, queryForAgents, adkLocalNotes, adkWebNotes, adkArticleSources, adkWebSources)
		if fallbackErr == nil && strings.TrimSpace(answer) != "" {
			fullAnswer = answer
			if currentStep != nil {
				currentStep.appendDetail("replanner 未通过 respond 工具产出最终答案，已根据已执行步骤和检索结果生成兜底回答。")
			}
			o.emitChunk(eventCh, conv, runID, fullAnswer, nil)
		} else {
			err := fmt.Errorf("ADK run completed without final respond output")
			if fallbackErr != nil {
				err = fmt.Errorf("%w: %v", err, fallbackErr)
			}
			if currentStep != nil {
				currentStep.fail("ADK 运行结束，但 replanner 没有通过 respond 工具产出最终答案。")
				o.emitStep(eventCh, conv, runID, currentStep.snapshot())
			}
			s.runs.logStage(conv, userID, "failed", "ADK 未产出最终回答", err.Error())
			errCh <- err
			return err
		}
	}

	if currentStep != nil {
		currentStep.complete()
		o.emitStep(eventCh, conv, runID, currentStep.snapshot())
	}
	o.persistFinalAnswer(conv, derefUserID(userID), question, fullAnswer, decision, history)
	s.runs.logStage(conv, userID, "completed", "多 Agent 协作完成", fmt.Sprintf("答案长度: %d，答案内容：%v", len(fullAnswer), fullAnswer))
	o.emitDone(eventCh, conv, runID, "completed", "调研已完成")
	return nil
}

func (o *thinkTankOrchestrator) streamManualFlow(
	ctx context.Context,
	eventCh chan<- StreamEvent,
	errCh chan<- error,
	conv *model.Conversation,
	history []model.ChatMessage,
	question string,
	userID *int64,
	queryForAgents string,
	decision PlannerDecision,
	runID int64,
) {
	s := o.service
	s.runs.logStage(conv, userID, "manual_start", "开始手动编排流程", queryForAgents)

	conversationID := int64(0)
	if conv != nil {
		conversationID = conv.ID
	}

	o.emitStage(eventCh, conv, runID, "local_search", "正在检索站内知识")
	libStep := s.runs.newStepTracker(conversationID, runID, "Librarian", "正在检索站内知识")
	o.emitStep(eventCh, conv, runID, libStep.snapshot())
	localResult, err := s.librarian.Search(ctx, queryForAgents)
	if err != nil {
		libStep.fail(err.Error())
		o.emitStep(eventCh, conv, runID, libStep.snapshot())
		s.runs.logStage(conv, userID, "failed", "本地检索失败", err.Error())
		errCh <- err
		return
	}
	libStep.appendDetail(formatLibrarianStepDetail(localResult))
	libStep.complete()
	o.emitStep(eventCh, conv, runID, libStep.snapshot())
	s.runs.logStage(conv, userID, "local_search_done", "本地检索完成", fmt.Sprintf("状态: %s", localResult.CoverageStatus))

	var webResult *JournalistResult
	if localResult.CoverageStatus != "sufficient" && s.journalist != nil {
		o.emitStage(eventCh, conv, runID, "web_research", "正在进行外部调研")
		s.runs.logStage(conv, userID, "web_research_start", "开始外部调研", "")
		jouStep := s.runs.newStepTracker(conversationID, runID, "Journalist", "正在进行外部调研")
		o.emitStep(eventCh, conv, runID, jouStep.snapshot())
		webResult, err = s.journalist.Research(ctx, queryForAgents, localResult)
		if err != nil {
			jouStep.fail(err.Error())
			o.emitStep(eventCh, conv, runID, jouStep.snapshot())
			s.runs.logStage(conv, userID, "failed", "外部调研失败", err.Error())
			errCh <- err
			return
		}
		jouStep.appendDetail(formatJournalistStepDetail(webResult))
		jouStep.complete()
		o.emitStep(eventCh, conv, runID, jouStep.snapshot())
		s.researchDraft.saveFromJournalist(derefUserID(userID), webResult)
		s.runs.logStage(conv, userID, "web_research_done", "外部调研完成", fmt.Sprintf("来源数: %d", len(webResult.Sources)))
	}

	o.emitStage(eventCh, conv, runID, "integration", "正在整合专家结果")
	synStep := s.runs.newStepTracker(conversationID, runID, "Synthesizer", "正在整合专家结果")
	o.emitStep(eventCh, conv, runID, synStep.snapshot())
	answer, sources, err := s.synthesizer.Compose(ctx, queryForAgents, localResult, webResult)
	if err != nil {
		synStep.fail(err.Error())
		o.emitStep(eventCh, conv, runID, synStep.snapshot())
		s.runs.logStage(conv, userID, "failed", "结果整合失败", err.Error())
		errCh <- err
		return
	}
	synStep.appendDetail(formatSynthesizerStepDetail(localResult, webResult, sources))
	synStep.complete()
	o.emitStep(eventCh, conv, runID, synStep.snapshot())
	s.runs.logStage(conv, userID, "integration_done", "结果整合完成", "")

	o.persistFinalAnswer(conv, derefUserID(userID), question, answer, decision, history)
	for _, chunk := range splitStreamChunks(answer) {
		o.emitChunk(eventCh, conv, runID, chunk, sources)
	}
	o.emitDone(eventCh, conv, runID, "completed", "回答已生成")
}

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
