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
const thinkTankRunStaleAfter = thinkTankStreamRunTimeout + time.Minute

var errThinkTankRunAlreadyRunning = errors.New("another answer is still running for this conversation")

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

func (o *thinkTankOrchestrator) clarifyAgentQuery(ctx context.Context, question string, queryForAgents string) (string, ClarifierDecision, bool, string) {
	s := o.service
	if s.clarifier == nil {
		return queryForAgents, defaultClarifierDecision(question), false, ""
	}
	decision, err := s.clarifier.Clarify(ctx, ClarifierInput{
		OriginalQuestion: question,
		AgentQuery:       queryForAgents,
	})
	if err != nil {
		s.runs.logStage(nil, nil, "clarifier_warning", "Clarifier failed; continuing with original question", err.Error())
		return queryForAgents, defaultClarifierDecision(question), false, ""
	}
	if decision.ShouldAskUser {
		clarificationQuestion := formatClarifierQuestion(decision)
		if strings.TrimSpace(clarificationQuestion) == "" {
			clarificationQuestion = strings.TrimSpace(decision.ClarificationQuestion)
		}
		return queryForAgents, decision, true, clarificationQuestion
	}
	return buildClarifiedAgentQuery(queryForAgents, decision), decision, false, ""
}

func (o *thinkTankOrchestrator) reviewAnswer(ctx context.Context, question string, queryForAgents string, decision ClarifierDecision, answer string, revisionCount int) (AcceptanceReview, bool) {
	s := o.service
	if s.acceptanceReviewer == nil || strings.TrimSpace(answer) == "" {
		return defaultAcceptanceReview(), false
	}
	input := AcceptanceReviewInput{
		OriginalQuestion: question,
		AgentQuery:       queryForAgents,
		Decision:         decision,
		Answer:           answer,
		RevisionCount:    revisionCount,
	}
	review, err := s.acceptanceReviewer.Review(ctx, input)
	if err != nil {
		s.runs.logStage(nil, nil, "acceptance_warning", "Acceptance review failed; returning generated answer", err.Error())
		return defaultAcceptanceReview(), false
	}
	shouldRevise := normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictRevise && revisionCount < s.maxReviewRevisions
	return review, shouldRevise
}

func (o *thinkTankOrchestrator) chat(ctx context.Context, question string, conversationID *int64, userID *int64) (*ThinkTankChatResponse, error) {
	s := o.service
	conv, err := s.conversations.getOwnedConversation(conversationID, userID)
	if err != nil {
		return nil, err
	}

	var history []model.ChatMessage
	var pending *model.ConversationRun
	if conv != nil {
		history = s.conversations.loadHistory(conv.ID)
		pending = s.runs.activeRun(conv.ID)
		pending, err = o.pendingRunForNewInput(pending)
		if err != nil {
			return nil, err
		}
		s.conversations.saveMessageWithWarning(conv.ID, "user", question, "Failed to save user message")
	}

	decision := PlannerDecision{ExecutionStrategy: "eino_plan_execute_replan", PlanSummary: "由 Eino PlanExecute planner 生成计划"}
	effectiveQuestion, skipClarifier := o.effectiveQuestionFromPending(question, pending)
	pendingRunID := runIDFromPending(pending)
	queryForAgents := o.buildAgentQuery(effectiveQuestion, conv, history)
	clarifierDecision := defaultClarifierDecision(effectiveQuestion)
	if skipClarifier {
		clarifierDecision = defaultClarifierDecision(effectiveQuestion)
	} else {
		var needsUser bool
		var clarificationQuestion string
		queryForAgents, clarifierDecision, needsUser, clarificationQuestion = o.clarifyAgentQuery(ctx, effectiveQuestion, queryForAgents)
		if needsUser {
			if conv != nil {
				pendingContext := marshalAgentPendingContext("clarifier_interrupt", effectiveQuestion, clarificationQuestion)
				s.runs.persistAgentClarification(conv.ID, derefUserID(userID), pendingRunID, effectiveQuestion, clarificationQuestion, "clarifying", pendingContext, decision)
				s.conversations.saveMessageWithWarning(conv.ID, "assistant", clarificationQuestion, "Failed to save clarification message")
			}
			return &ThinkTankChatResponse{Message: clarificationQuestion, Stage: "clarifying", RequiresUserInput: true}, nil
		}
	}

	if s.adkRunner != nil && s.adkAnswerFetcher != nil {
		adkCtx := WithUserID(ctx, derefUserID(userID))
		adkCtx = WithAILogger(adkCtx, s.logger)
		adkCtx = WithWebFetchState(adkCtx, newWebFetchState())
		if conv != nil {
			adkCtx = WithConversationID(adkCtx, conv.ID)
		}
		answer, err := s.adkAnswerFetcher(adkCtx, queryForAgents)
		if err == nil && strings.TrimSpace(answer) != "" {
			finalAnswer := answer
			review, shouldRevise := o.reviewAnswer(ctx, effectiveQuestion, queryForAgents, clarifierDecision, finalAnswer, 0)
			if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictAskUser {
				return o.acceptanceQuestionResponse(conv, derefUserID(userID), pendingRunID, effectiveQuestion, review, decision), nil
			}
			if shouldRevise {
				revisedAnswer, revisionErr := s.adkAnswerFetcher(adkCtx, buildRevisionAgentQuery(queryForAgents, finalAnswer, review))
				if revisionErr != nil || strings.TrimSpace(revisedAnswer) == "" {
					finalAnswer = appendAcceptanceLimitations(finalAnswer, review)
				} else {
					finalAnswer = revisedAnswer
					review, _ = o.reviewAnswer(ctx, effectiveQuestion, queryForAgents, clarifierDecision, finalAnswer, 1)
					if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictAskUser {
						return o.acceptanceQuestionResponse(conv, derefUserID(userID), pendingRunID, effectiveQuestion, review, decision), nil
					}
					if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictRevise {
						finalAnswer = appendAcceptanceLimitations(finalAnswer, review)
					}
				}
			}
			finalAnswer, err = s.finalizeAnswerFromEvidence(ctx, finalAnswer, effectiveQuestion, nil, nil, nil, nil)
			if err == nil {
				o.persistFinalAnswer(conv, derefUserID(userID), effectiveQuestion, finalAnswer, decision, history, 0)
				return &ThinkTankChatResponse{Message: finalAnswer, Stage: "completed"}, nil
			}
			s.runs.logStage(conv, userID, "adk_answer_warning", "ADK 回答没有可展示内容，降级到手动编排流程", err.Error())
		}
	}

	localResult, localErr := o.searchLocal(ctx, queryForAgents)

	var webResult *JournalistResult
	var webErr error
	if shouldRunJournalist(localResult, localErr) && s.journalist != nil {
		webResult, webErr = s.journalist.Research(ctx, queryForAgents, localResult)
		if webErr == nil {
			s.researchDraft.saveFromJournalist(derefUserID(userID), webResult)
		} else {
			s.runs.logStage(conv, userID, "web_research_warning", "外部调研失败，尝试使用已有结果继续", webErr.Error())
		}
	}
	if err := ensureManualEvidence(localResult, webResult, localErr, webErr); err != nil {
		return nil, err
	}

	answer, sources, err := s.synthesizer.Compose(ctx, queryForAgents, localResult, webResult)
	if err != nil {
		return nil, err
	}

	review, shouldRevise := o.reviewAnswer(ctx, effectiveQuestion, queryForAgents, clarifierDecision, answer, 0)
	if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictAskUser {
		return o.acceptanceQuestionResponse(conv, derefUserID(userID), pendingRunID, effectiveQuestion, review, decision), nil
	}
	if shouldRevise && s.adkRunner != nil && s.adkAnswerFetcher != nil {
		adkCtx := WithUserID(ctx, derefUserID(userID))
		adkCtx = WithAILogger(adkCtx, s.logger)
		adkCtx = WithWebFetchState(adkCtx, newWebFetchState())
		if conv != nil {
			adkCtx = WithConversationID(adkCtx, conv.ID)
		}
		revisedAnswer, revisionErr := s.adkAnswerFetcher(adkCtx, buildRevisionAgentQuery(queryForAgents, answer, review))
		if revisionErr != nil || strings.TrimSpace(revisedAnswer) == "" {
			answer = appendAcceptanceLimitations(answer, review)
		} else {
			answer = revisedAnswer
			review, _ = o.reviewAnswer(ctx, effectiveQuestion, queryForAgents, clarifierDecision, answer, 1)
			if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictAskUser {
				return o.acceptanceQuestionResponse(conv, derefUserID(userID), pendingRunID, effectiveQuestion, review, decision), nil
			}
			if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictRevise {
				answer = appendAcceptanceLimitations(answer, review)
			}
		}
	}

	answer, err = s.finalizeAnswerFromEvidence(
		ctx,
		answer,
		effectiveQuestion,
		[]string{localResult.Summary},
		journalistSummaryNotes(webResult),
		localResult.Sources,
		webSources(webResult),
	)
	if err != nil {
		return nil, err
	}
	o.persistFinalAnswer(conv, derefUserID(userID), effectiveQuestion, answer, decision, history, 0)
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
			pending, err = o.pendingRunForNewInput(pending)
			if err != nil {
				errCh <- err
				return
			}
			s.conversations.saveMessageWithWarning(conv.ID, "user", question, "Failed to save user message")
		}

		o.emitStage(eventCh, conv, 0, "analyzing", "正在理解你的问题")
		decision := PlannerDecision{ExecutionStrategy: "eino_plan_execute_replan", PlanSummary: "由 Eino PlanExecute planner 生成计划"}
		o.emitStage(eventCh, conv, 0, "analyzing", "正在进行计划-执行-审查调研")
		s.runs.logStage(conv, userID, "adk_start", "开始 ThinkTank 计划执行流程", question)

		effectiveQuestion, skipClarifier := o.effectiveQuestionFromPending(question, pending)
		queryForAgents := o.buildAgentQuery(effectiveQuestion, conv, history)
		runID, checkpointID, resumeFromADKInterrupt := o.prepareADKRun(conv, pending, userID, effectiveQuestion, decision)
		o.emitResume(eventCh, conv, runID, "analyzing", "running")
		o.emitSnapshot(eventCh, conv, runID, "analyzing", "running", "")
		clarifierDecision := defaultClarifierDecision(effectiveQuestion)
		if !resumeFromADKInterrupt && !skipClarifier {
			o.emitStage(eventCh, conv, runID, "clarifying_intent", "正在澄清你的意图")
			clarifiedQuery, clarifiedDecision, needsUser, clarificationQuestion := o.clarifyAgentQuery(runCtx, effectiveQuestion, queryForAgents)
			clarifierDecision = clarifiedDecision
			o.emitSyntheticAgentStep(eventCh, conv, runID, "ClarifierAgent", "正在澄清用户意图", formatClarifierStepDetail(clarifiedDecision))
			if needsUser {
				if conv != nil {
					pendingContext := marshalAgentPendingContext("clarifier_interrupt", effectiveQuestion, clarificationQuestion)
					s.runs.persistAgentClarification(conv.ID, derefUserID(userID), runID, effectiveQuestion, clarificationQuestion, "clarifying", pendingContext, decision)
					s.conversations.saveMessageWithWarning(conv.ID, "assistant", clarificationQuestion, "Failed to save clarification message", runID)
				}
				o.emitStage(eventCh, conv, runID, "clarifying", "需要补充一点信息")
				o.emitQuestion(eventCh, conv, runID, "clarifying", clarificationQuestion)
				return
			}
			queryForAgents = clarifiedQuery
		}

		if s.adkRunner != nil && s.adkRunner.runner == nil && s.adkAnswerFetcher != nil {
			answer, err := s.adkAnswerFetcher(runCtx, queryForAgents)
			if err != nil {
				errCh <- err
				return
			}
			o.emitStage(eventCh, conv, runID, "reviewing", "正在审核答案质量")
			review, shouldRevise := o.reviewAnswer(runCtx, effectiveQuestion, queryForAgents, clarifierDecision, answer, 0)
			o.emitSyntheticAgentStep(eventCh, conv, runID, "AcceptanceAgent", "正在验收答案质量", formatAcceptanceStepDetail(review, false))
			if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictAskUser {
				o.emitAcceptanceQuestion(eventCh, conv, runID, derefUserID(userID), effectiveQuestion, review, decision)
				return
			}
			revised := false
			if shouldRevise {
				o.emitStage(eventCh, conv, runID, "revising", "正在根据审核意见修订答案")
				revisedAnswer, revisionErr := s.adkAnswerFetcher(runCtx, buildRevisionAgentQuery(queryForAgents, answer, review))
				if revisionErr != nil {
					s.runs.logStage(conv, userID, "acceptance_revision_warning", "答案修订失败，返回初版并附加审核说明", revisionErr.Error())
					answer = appendAcceptanceLimitations(answer, review)
				} else if strings.TrimSpace(revisedAnswer) == "" {
					answer = appendAcceptanceLimitations(answer, review)
				} else {
					answer = revisedAnswer
					review, _ = o.reviewAnswer(runCtx, effectiveQuestion, queryForAgents, clarifierDecision, answer, 1)
					revised = normalizeAcceptanceVerdict(review.Verdict) != acceptanceVerdictRevise && normalizeAcceptanceVerdict(review.Verdict) != acceptanceVerdictAskUser
					o.emitSyntheticAgentStep(eventCh, conv, runID, "AcceptanceAgent", "正在验收修订后答案", formatAcceptanceStepDetail(review, revised))
					if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictAskUser {
						o.emitAcceptanceQuestion(eventCh, conv, runID, derefUserID(userID), effectiveQuestion, review, decision)
						return
					}
					if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictRevise {
						answer = appendAcceptanceLimitations(answer, review)
					}
				}
			}
			answer, err = s.finalizeAnswerFromEvidence(runCtx, answer, effectiveQuestion, nil, nil, nil, nil)
			if err != nil {
				s.runs.logStage(conv, userID, "adk_answer_warning", "ADK 回答没有可展示内容，降级到手动编排流程", err.Error())
				o.streamManualFlow(runCtx, eventCh, errCh, conv, history, effectiveQuestion, userID, queryForAgents, decision, runID, clarifierDecision)
				return
			}
			o.persistFinalAnswer(conv, derefUserID(userID), effectiveQuestion, answer, decision, history, runID)
			o.emitChunk(eventCh, conv, runID, answer, nil)
			o.emitDone(eventCh, conv, runID, "completed", "回答已生成")
			return
		}

		if s.adkRunner != nil && s.adkRunner.runner != nil {
			if err := o.streamADKFlow(runCtx, eventCh, errCh, conv, history, effectiveQuestion, question, userID, queryForAgents, decision, checkpointID, runID, resumeFromADKInterrupt, clarifierDecision); err != nil {
				return
			}
			return
		}

		o.streamManualFlow(runCtx, eventCh, errCh, conv, history, effectiveQuestion, userID, queryForAgents, decision, runID, clarifierDecision)
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

func (o *thinkTankOrchestrator) effectiveQuestionFromPending(question string, pending *model.ConversationRun) (string, bool) {
	if pendingContext, ok := parseAgentPendingContext(pending); ok {
		return buildInterruptedFollowUpQuestion(pendingContext.OriginalQuestion, pendingContext.SystemQuestion, question), true
	}
	if pendingContext, ok := parseADKPendingContext(pending); ok && strings.TrimSpace(pendingContext.OriginalQuestion) != "" && strings.TrimSpace(pendingContext.SystemQuestion) != "" {
		return buildInterruptedFollowUpQuestion(pendingContext.OriginalQuestion, pendingContext.SystemQuestion, question), true
	}
	return question, false
}

func (o *thinkTankOrchestrator) pendingRunForNewInput(pending *model.ConversationRun) (*model.ConversationRun, error) {
	if pending == nil || pending.Status != "running" {
		return pending, nil
	}
	if isStaleThinkTankRun(pending, time.Now()) {
		o.service.runs.persistFailure(pending.ID, errors.New("previous running answer became stale"))
		return nil, nil
	}
	return nil, errThinkTankRunAlreadyRunning
}

func isStaleThinkTankRun(run *model.ConversationRun, now time.Time) bool {
	if run == nil || run.Status != "running" {
		return false
	}
	lastActivity := run.UpdatedAt
	if run.HeartbeatAt != nil {
		lastActivity = *run.HeartbeatAt
	}
	if lastActivity.IsZero() {
		return false
	}
	return now.Sub(lastActivity) > thinkTankRunStaleAfter
}

func runIDFromPending(pending *model.ConversationRun) int64 {
	if pending == nil || pending.ID <= 0 || pending.Status != "waiting_user" {
		return 0
	}
	return pending.ID
}

func (o *thinkTankOrchestrator) acceptanceQuestionResponse(conv *model.Conversation, userID int64, runID int64, question string, review AcceptanceReview, decision PlannerDecision) *ThinkTankChatResponse {
	userQuestion := strings.TrimSpace(formatAcceptanceQuestion(review))
	if userQuestion == "" {
		userQuestion = "还需要你补充一点信息，我才能继续。"
	}
	if conv != nil {
		pendingContext := marshalAgentPendingContext("acceptance_interrupt", question, userQuestion)
		o.service.runs.persistAgentClarification(conv.ID, userID, runID, question, userQuestion, "clarifying", pendingContext, decision)
		o.service.conversations.saveMessageWithWarning(conv.ID, "assistant", userQuestion, "Failed to save acceptance clarification message", runID)
	}
	return &ThinkTankChatResponse{Message: userQuestion, Stage: "clarifying", RequiresUserInput: true}
}

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

func (o *thinkTankOrchestrator) persistFinalAnswer(conv *model.Conversation, userID int64, question string, answer string, decision PlannerDecision, history []model.ChatMessage, runID int64) {
	if conv == nil || strings.TrimSpace(answer) == "" {
		return
	}
	o.service.conversations.persistAssistantTurn(conv, question, answer, runID)
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
	resumeInput string,
	userID *int64,
	queryForAgents string,
	decision PlannerDecision,
	checkpointID string,
	runID int64,
	resumeFromADKInterrupt bool,
	clarifierDecision ClarifierDecision,
) error {
	s := o.service
	adkCtx := WithUserID(ctx, derefUserID(userID))
	adkCtx = WithAILogger(adkCtx, s.logger)
	adkCtx = WithRunID(adkCtx, runID)
	adkCtx = WithWebFetchState(adkCtx, newWebFetchState())
	if conv != nil {
		adkCtx = WithConversationID(adkCtx, conv.ID)
	}

	var adkIter *adk.AsyncIterator[*adk.AgentEvent]
	if resumeFromADKInterrupt {
		resumeIter, resumeErr := s.adkRunner.runner.Resume(adkCtx, checkpointID, adk.WithToolOptions([]tool.Option{WithNewInput(resumeInput)}))
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
				s.conversations.saveMessageWithWarning(conv.ID, "assistant", clarification, "Failed to save clarification message", runID)
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

	o.emitStage(eventCh, conv, runID, "reviewing", "正在审核答案质量")
	if currentStep != nil {
		currentStep.complete()
		o.emitStep(eventCh, conv, runID, currentStep.snapshot())
	}
	review, shouldRevise := o.reviewAnswer(ctx, question, queryForAgents, clarifierDecision, fullAnswer, 0)
	o.emitSyntheticAgentStep(eventCh, conv, runID, "AcceptanceAgent", "正在验收答案质量", formatAcceptanceStepDetail(review, false))
	if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictAskUser {
		o.emitAcceptanceQuestion(eventCh, conv, runID, derefUserID(userID), question, review, decision)
		return nil
	}
	revised := false
	if shouldRevise && s.adkAnswerFetcher != nil {
		o.emitStage(eventCh, conv, runID, "revising", "正在根据审核意见修订答案")
		revisedAnswer, revisionErr := s.adkAnswerFetcher(adkCtx, buildRevisionAgentQuery(queryForAgents, fullAnswer, review))
		if revisionErr != nil {
			s.runs.logStage(conv, userID, "acceptance_revision_warning", "答案修订失败，返回初版并附加审核说明", revisionErr.Error())
			fullAnswer = appendAcceptanceLimitations(fullAnswer, review)
		} else if strings.TrimSpace(revisedAnswer) == "" {
			fullAnswer = appendAcceptanceLimitations(fullAnswer, review)
		} else {
			fullAnswer = revisedAnswer
			review, _ = o.reviewAnswer(ctx, question, queryForAgents, clarifierDecision, fullAnswer, 1)
			revised = normalizeAcceptanceVerdict(review.Verdict) != acceptanceVerdictRevise && normalizeAcceptanceVerdict(review.Verdict) != acceptanceVerdictAskUser
			o.emitSyntheticAgentStep(eventCh, conv, runID, "AcceptanceAgent", "正在验收修订后答案", formatAcceptanceStepDetail(review, revised))
			if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictAskUser {
				o.emitAcceptanceQuestion(eventCh, conv, runID, derefUserID(userID), question, review, decision)
				return nil
			}
			if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictRevise {
				fullAnswer = appendAcceptanceLimitations(fullAnswer, review)
			}
		}
	} else if shouldRevise {
		fullAnswer = appendAcceptanceLimitations(fullAnswer, review)
	}
	fullAnswer, err := s.finalizeAnswerFromEvidence(ctx, fullAnswer, question, adkLocalNotes, adkWebNotes, adkArticleSources, adkWebSources)
	if err != nil {
		if currentStep != nil {
			currentStep.fail("最终回答清洗后没有可展示内容。")
			o.emitStep(eventCh, conv, runID, currentStep.snapshot())
		}
		s.runs.logStage(conv, userID, "failed", "最终回答清洗后没有可展示内容", err.Error())
		errCh <- err
		return err
	}
	o.persistFinalAnswer(conv, derefUserID(userID), question, fullAnswer, decision, history, runID)
	s.runs.logStage(conv, userID, "completed", "ThinkTank 计划执行流程完成", fmt.Sprintf("答案长度: %d，答案内容：%v", len(fullAnswer), fullAnswer))
	o.emitChunk(eventCh, conv, runID, fullAnswer, collectSourceRefTitles(adkArticleSources, adkWebSources))
	o.emitDone(eventCh, conv, runID, "completed", "调研已完成")
	return nil
}
