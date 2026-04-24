package service

import (
	"fmt"
	"strings"
	"time"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

type thinkTankConversationManager struct {
	convRepo repository.ConversationRepository
	msgRepo  repository.ChatMessageRepository
	logger   AILogger
}

func newThinkTankConversationManager(
	convRepo repository.ConversationRepository,
	msgRepo repository.ChatMessageRepository,
	logger AILogger,
) *thinkTankConversationManager {
	return &thinkTankConversationManager{
		convRepo: convRepo,
		msgRepo:  msgRepo,
		logger:   logger,
	}
}

func (m *thinkTankConversationManager) getOwnedConversation(conversationID *int64, userID *int64) (*model.Conversation, error) {
	if conversationID == nil || *conversationID <= 0 {
		return nil, nil
	}
	if userID == nil {
		return nil, fmt.Errorf("user authentication required")
	}
	if m == nil || m.convRepo == nil {
		return nil, fmt.Errorf("conversation repository unavailable")
	}
	conv, err := m.convRepo.GetByID(*conversationID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}
	if conv.UserID != *userID {
		return nil, fmt.Errorf("conversation access denied")
	}
	return conv, nil
}

func (m *thinkTankConversationManager) loadHistory(conversationID int64) []model.ChatMessage {
	if m == nil || m.msgRepo == nil || conversationID <= 0 {
		return nil
	}
	history, err := m.msgRepo.GetByConversationID(conversationID)
	if err != nil {
		if m.logger != nil {
			m.logger.LogError(AILogEntry{
				ConversationID: conversationID,
				Stage:          "persistence",
				Message:        "Failed to load conversation history",
				Detail:         err.Error(),
			})
		}
		return nil
	}
	return history
}

func (m *thinkTankConversationManager) saveMessageWithWarning(conversationID int64, role string, content string, logMessage string) {
	if m == nil || m.msgRepo == nil {
		return
	}
	if err := m.msgRepo.Create(&model.ChatMessage{ConversationID: conversationID, Role: role, Content: content}); err != nil {
		if m.logger != nil {
			m.logger.LogError(AILogEntry{ConversationID: conversationID, Stage: "persistence", Message: logMessage, Detail: err.Error()})
		}
	}
}

func (m *thinkTankConversationManager) updateMetadataWithWarning(conv *model.Conversation, question string) {
	if conv == nil || m == nil || m.convRepo == nil {
		return
	}
	conv.UpdatedAt = time.Now()
	if conv.Title == "" || conv.Title == "New Conversation" || conv.Title == "新会话" || conv.Title == "New Chat" {
		conv.Title = buildConversationTitle(question)
	}
	_ = m.convRepo.Update(conv)
}

func (m *thinkTankConversationManager) persistAssistantTurn(conv *model.Conversation, question string, answer string) {
	if conv == nil || strings.TrimSpace(answer) == "" {
		return
	}
	m.saveMessageWithWarning(conv.ID, "assistant", answer, "Failed to save assistant message")
	m.updateMetadataWithWarning(conv, question)
}

func (s *thinkTankService) getOwnedConversation(conversationID *int64, userID *int64) (*model.Conversation, error) {
	return s.conversations.getOwnedConversation(conversationID, userID)
}

func (s *thinkTankService) saveConversationMessageWithWarning(conversationID int64, role string, content string, logMessage string) {
	s.conversations.saveMessageWithWarning(conversationID, role, content, logMessage)
}

func (s *thinkTankService) updateConversationMetadataWithWarning(conv *model.Conversation, question string) {
	s.conversations.updateMetadataWithWarning(conv, question)
}

func derefUserID(userID *int64) int64 {
	if userID == nil {
		return 0
	}
	return *userID
}
