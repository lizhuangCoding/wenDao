package chat

import (
	"context"
	"fmt"
	"strings"

	"wenDao/internal/model"
)

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
	clarifierDecision ClarifierDecision,
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

	o.emitStage(eventCh, conv, runID, "reviewing", "正在审核答案质量")
	review, shouldRevise := o.reviewAnswer(ctx, question, queryForAgents, clarifierDecision, answer, 0)
	o.emitSyntheticAgentStep(eventCh, conv, runID, "AcceptanceAgent", "正在验收答案质量", formatAcceptanceStepDetail(review, false))
	if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictAskUser {
		o.emitAcceptanceQuestion(eventCh, conv, runID, derefUserID(userID), question, review, decision)
		return
	}

	revised := false
	if shouldRevise && s.adkRunner != nil && s.adkAnswerFetcher != nil {
		o.emitStage(eventCh, conv, runID, "revising", "正在根据审核意见修订答案")
		revisedAnswer, revisionErr := s.adkAnswerFetcher(ctx, buildRevisionAgentQuery(queryForAgents, answer, review))
		if revisionErr != nil {
			s.runs.logStage(conv, userID, "acceptance_revision_warning", "答案修订失败，返回初版并附加审核说明", revisionErr.Error())
			answer = appendAcceptanceLimitations(answer, review)
		} else if strings.TrimSpace(revisedAnswer) == "" {
			answer = appendAcceptanceLimitations(answer, review)
		} else if strings.TrimSpace(revisedAnswer) != "" {
			answer = revisedAnswer
			review, _ = o.reviewAnswer(ctx, question, queryForAgents, clarifierDecision, answer, 1)
			revised = normalizeAcceptanceVerdict(review.Verdict) != acceptanceVerdictRevise && normalizeAcceptanceVerdict(review.Verdict) != acceptanceVerdictAskUser
			o.emitSyntheticAgentStep(eventCh, conv, runID, "AcceptanceAgent", "正在验收修订后答案", formatAcceptanceStepDetail(review, revised))
			if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictAskUser {
				o.emitAcceptanceQuestion(eventCh, conv, runID, derefUserID(userID), question, review, decision)
				return
			}
			if normalizeAcceptanceVerdict(review.Verdict) == acceptanceVerdictRevise {
				answer = appendAcceptanceLimitations(answer, review)
			}
		}
	} else if shouldRevise {
		answer = appendAcceptanceLimitations(answer, review)
	}

	answer = sanitizeFinalAnswerForUser(answer)
	o.persistFinalAnswer(conv, derefUserID(userID), question, answer, decision, history, runID)
	for _, chunk := range splitStreamChunks(answer) {
		o.emitChunk(eventCh, conv, runID, chunk, sources)
	}
	o.emitDone(eventCh, conv, runID, "completed", "回答已生成")
}
