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
