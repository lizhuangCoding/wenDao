package chat

import (
	aisvc "wenDao/internal/service/ai"
	chatcore "wenDao/internal/service/chatcore"
	knowledgesvc "wenDao/internal/service/knowledge"
)

type AILogger = aisvc.AILogger
type AILogEntry = aisvc.AILogEntry
type KnowledgeDocumentService = knowledgesvc.KnowledgeDocumentService
type KnowledgeSourceInput = knowledgesvc.KnowledgeSourceInput
type CreateKnowledgeDocumentInput = knowledgesvc.CreateKnowledgeDocumentInput
type StreamEventType = chatcore.StreamEventType
type StreamEvent = chatcore.StreamEvent
type ThinkTankChatResponse = chatcore.ThinkTankChatResponse
type ThinkTankService = chatcore.ThinkTankService

const (
	StreamEventStage     = chatcore.StreamEventStage
	StreamEventQuestion  = chatcore.StreamEventQuestion
	StreamEventChunk     = chatcore.StreamEventChunk
	StreamEventStep      = chatcore.StreamEventStep
	StreamEventResume    = chatcore.StreamEventResume
	StreamEventSnapshot  = chatcore.StreamEventSnapshot
	StreamEventHeartbeat = chatcore.StreamEventHeartbeat
	StreamEventDone      = chatcore.StreamEventDone
)

func buildConversationTitle(question string) string {
	title := question
	runes := []rune(title)
	if len(runes) > 30 {
		return string(runes[:30]) + "..."
	}
	return title
}
