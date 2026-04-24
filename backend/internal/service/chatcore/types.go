package chatcore

import "context"

type StreamEventType string

const (
	StreamEventStage    StreamEventType = "stage"
	StreamEventQuestion StreamEventType = "question"
	StreamEventChunk    StreamEventType = "chunk"
	StreamEventStep     StreamEventType = "step"
	StreamEventDone     StreamEventType = "done"
)

type StreamEvent struct {
	Type    StreamEventType
	Stage   string
	Label   string
	Message string
	Sources []string

	StepID    int64
	AgentName string
	Status    string
	Summary   string
	Detail    string
}

type ThinkTankChatResponse struct {
	Message           string
	Sources           []string
	Stage             string
	RequiresUserInput bool
	Steps             any
}

type ThinkTankService interface {
	Chat(ctx context.Context, question string, conversationID *int64, userID *int64) (*ThinkTankChatResponse, error)
	ChatStream(ctx context.Context, question string, conversationID *int64, userID *int64) (<-chan StreamEvent, <-chan error)
}
