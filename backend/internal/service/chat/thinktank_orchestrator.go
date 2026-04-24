package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"wenDao/internal/model"
)

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

		s.streams.emitStage(eventCh, "analyzing", "正在理解你的问题")
		decision := PlannerDecision{ExecutionStrategy: "eino_plan_execute_replan", PlanSummary: "由 Eino PlanExecute planner 生成计划"}
		s.streams.emitStage(eventCh, "analyzing", "正在进行多 Agent 深度调研")
		s.runs.logStage(conv, userID, "adk_start", "开始多 Agent 协作流", question)

		queryForAgents := o.buildAgentQuery(question, conv, history)
		runID, checkpointID, resumeFromADKInterrupt := o.prepareADKRun(conv, pending, userID, question, decision)

		if s.adkRunner != nil && s.adkRunner.runner != nil {
			if err := o.streamADKFlow(ctx, eventCh, errCh, conv, history, question, userID, queryForAgents, decision, checkpointID, runID, resumeFromADKInterrupt); err != nil {
				return
			}
			return
		}

		o.streamManualFlow(ctx, eventCh, errCh, conv, history, question, userID, queryForAgents, decision, runID)
	}()
	return eventCh, errCh
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
				s.streams.emitStep(eventCh, currentStep.snapshot())
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
				s.streams.emitStep(eventCh, currentStep.snapshot())
			}
			if conv != nil {
				s.runs.persistADKClarification(conv.ID, derefUserID(userID), runID, question, clarification, checkpointID, decision)
				s.conversations.saveMessageWithWarning(conv.ID, "assistant", clarification, "Failed to save clarification message")
			}
			s.streams.emitStage(eventCh, "clarifying", "需要补充一点信息")
			s.streams.emitQuestion(eventCh, "clarifying", clarification)
			return nil
		}

		if e.AgentName != "" && (currentStep == nil || currentStep.step.AgentName != e.AgentName) {
			if currentStep != nil {
				currentStep.complete()
				s.streams.emitStep(eventCh, currentStep.snapshot())
			}
			summary, label := adkAgentStepMetadata(e.AgentName)
			if label != "" {
				s.streams.emitStage(eventCh, "adk_event", label)
			}
			currentStep = s.runs.newStepTracker(conversationID, runID, e.AgentName, summary)
			s.streams.emitStep(eventCh, currentStep.snapshot())
		}

		if actionDetail := formatADKActionDetail(e.Action); actionDetail != "" && currentStep != nil {
			currentStep.appendDetail(actionDetail)
			s.streams.emitStep(eventCh, currentStep.snapshot())
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
			s.streams.emitStep(eventCh, currentStep.snapshot())
		}

		if e.AgentName == "replanner" && strings.TrimSpace(msg.Content) != "" {
			if response, ok := extractPlanExecuteFinalResponse(msg.Content); ok {
				if isNonFinalToolLimitationAnswer(response) {
					adkWebNotes = appendNonEmptyNote(adkWebNotes, "replanner returned a tool limitation instead of a user-facing answer: "+response)
				} else {
					fullAnswer = appendGroupedReferences(response, adkArticleSources, adkWebSources)
					s.streams.emitChunk(eventCh, fullAnswer, nil)
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
			s.streams.emitChunk(eventCh, fullAnswer, nil)
		} else {
			err := fmt.Errorf("ADK run completed without final respond output")
			if fallbackErr != nil {
				err = fmt.Errorf("%w: %v", err, fallbackErr)
			}
			if currentStep != nil {
				currentStep.fail("ADK 运行结束，但 replanner 没有通过 respond 工具产出最终答案。")
				s.streams.emitStep(eventCh, currentStep.snapshot())
			}
			s.runs.logStage(conv, userID, "failed", "ADK 未产出最终回答", err.Error())
			errCh <- err
			return err
		}
	}

	if currentStep != nil {
		currentStep.complete()
		s.streams.emitStep(eventCh, currentStep.snapshot())
	}
	o.persistFinalAnswer(conv, derefUserID(userID), question, fullAnswer, decision, history)
	s.runs.logStage(conv, userID, "completed", "多 Agent 协作完成", fmt.Sprintf("答案长度: %d，答案内容：%v", len(fullAnswer), fullAnswer))
	s.streams.emitDone(eventCh, "completed", "调研已完成")
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

	s.streams.emitStage(eventCh, "local_search", "正在检索站内知识")
	libStep := s.runs.newStepTracker(conversationID, runID, "Librarian", "正在检索站内知识")
	s.streams.emitStep(eventCh, libStep.snapshot())
	localResult, err := s.librarian.Search(ctx, queryForAgents)
	if err != nil {
		libStep.fail(err.Error())
		s.streams.emitStep(eventCh, libStep.snapshot())
		s.runs.logStage(conv, userID, "failed", "本地检索失败", err.Error())
		errCh <- err
		return
	}
	libStep.appendDetail(formatLibrarianStepDetail(localResult))
	libStep.complete()
	s.streams.emitStep(eventCh, libStep.snapshot())
	s.runs.logStage(conv, userID, "local_search_done", "本地检索完成", fmt.Sprintf("状态: %s", localResult.CoverageStatus))

	var webResult *JournalistResult
	if localResult.CoverageStatus != "sufficient" && s.journalist != nil {
		s.streams.emitStage(eventCh, "web_research", "正在进行外部调研")
		s.runs.logStage(conv, userID, "web_research_start", "开始外部调研", "")
		jouStep := s.runs.newStepTracker(conversationID, runID, "Journalist", "正在进行外部调研")
		s.streams.emitStep(eventCh, jouStep.snapshot())
		webResult, err = s.journalist.Research(ctx, queryForAgents, localResult)
		if err != nil {
			jouStep.fail(err.Error())
			s.streams.emitStep(eventCh, jouStep.snapshot())
			s.runs.logStage(conv, userID, "failed", "外部调研失败", err.Error())
			errCh <- err
			return
		}
		jouStep.appendDetail(formatJournalistStepDetail(webResult))
		jouStep.complete()
		s.streams.emitStep(eventCh, jouStep.snapshot())
		s.researchDraft.saveFromJournalist(derefUserID(userID), webResult)
		s.runs.logStage(conv, userID, "web_research_done", "外部调研完成", fmt.Sprintf("来源数: %d", len(webResult.Sources)))
	}

	s.streams.emitStage(eventCh, "integration", "正在整合专家结果")
	synStep := s.runs.newStepTracker(conversationID, runID, "Synthesizer", "正在整合专家结果")
	s.streams.emitStep(eventCh, synStep.snapshot())
	answer, sources, err := s.synthesizer.Compose(ctx, queryForAgents, localResult, webResult)
	if err != nil {
		synStep.fail(err.Error())
		s.streams.emitStep(eventCh, synStep.snapshot())
		s.runs.logStage(conv, userID, "failed", "结果整合失败", err.Error())
		errCh <- err
		return
	}
	synStep.appendDetail(formatSynthesizerStepDetail(localResult, webResult, sources))
	synStep.complete()
	s.streams.emitStep(eventCh, synStep.snapshot())
	s.runs.logStage(conv, userID, "integration_done", "结果整合完成", "")

	o.persistFinalAnswer(conv, derefUserID(userID), question, answer, decision, history)
	for _, chunk := range splitStreamChunks(answer) {
		s.streams.emitChunk(eventCh, chunk, sources)
	}
	s.streams.emitDone(eventCh, "completed", "回答已生成")
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
