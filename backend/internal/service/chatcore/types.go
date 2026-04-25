package chatcore

import "context"

type StreamEventType string

const (
	StreamEventStage    StreamEventType = "stage"
	StreamEventQuestion StreamEventType = "question"
	StreamEventChunk    StreamEventType = "chunk"
	StreamEventStep     StreamEventType = "step"
	StreamEventResume   StreamEventType = "resume"
	StreamEventSnapshot StreamEventType = "snapshot"
	StreamEventHeartbeat StreamEventType = "heartbeat"
	StreamEventDone     StreamEventType = "done"
)

type StreamEvent struct {
	Type    StreamEventType
	RunID   int64
	Stage   string
	Label   string
	Status  string
	Message string
	Sources []string

	StepID    int64
	AgentName string
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
	ResumeChatStream(ctx context.Context, conversationID int64, runID int64, userID *int64) (<-chan StreamEvent, <-chan error)
}
