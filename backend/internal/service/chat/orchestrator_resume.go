package chat

import (
	"context"
	"errors"
	"strings"
)

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
