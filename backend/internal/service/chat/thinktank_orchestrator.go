package chat

import (
	"context"
	"errors"
	"strings"
	"time"

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

func (o *thinkTankOrchestrator) persistFinalAnswer(conv *model.Conversation, userID int64, question string, answer string, decision PlannerDecision, history []model.ChatMessage, runID int64) {
	if conv == nil || strings.TrimSpace(answer) == "" {
		return
	}
	o.service.conversations.persistAssistantTurn(conv, question, answer, runID)
	o.service.runs.persistCompletedRun(conv.ID, userID, question, answer, decision)
	o.service.memories.updateConversationMemoryWithWarning(conv.ID, userID, appendConversationTurn(history, question, answer))
}
