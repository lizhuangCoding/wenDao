package service

import (
	"fmt"
	"time"

	"wenDao/internal/model"
)

func (s *thinkTankService) getOwnedConversation(conversationID *int64, userID *int64) (*model.Conversation, error) {
	if conversationID == nil || *conversationID <= 0 {
		return nil, nil
	}
	if userID == nil {
		return nil, fmt.Errorf("user authentication required")
	}
	conv, err := s.convRepo.GetByID(*conversationID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}
	if conv.UserID != *userID {
		return nil, fmt.Errorf("conversation access denied")
	}
	return conv, nil
}

func (s *thinkTankService) saveConversationMessageWithWarning(conversationID int64, role string, content string, logMessage string) {
	if s.msgRepo == nil {
		return
	}
	if err := s.msgRepo.Create(&model.ChatMessage{ConversationID: conversationID, Role: role, Content: content}); err != nil {
		if s.logger != nil {
			s.logger.LogError(AILogEntry{ConversationID: conversationID, Stage: "persistence", Message: logMessage, Detail: err.Error()})
		}
	}
}

func (s *thinkTankService) updateConversationMetadataWithWarning(conv *model.Conversation, question string) {
	if conv == nil || s.convRepo == nil {
		return
	}
	conv.UpdatedAt = time.Now()
	if conv.Title == "" || conv.Title == "New Conversation" || conv.Title == "新会话" || conv.Title == "New Chat" {
		conv.Title = buildConversationTitle(question)
	}
	_ = s.convRepo.Update(conv)
}

func derefUserID(userID *int64) int64 {
	if userID == nil {
		return 0
	}
	return *userID
}
