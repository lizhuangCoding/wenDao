package chat

import (
	"sync"
	"time"

	"wenDao/internal/model"
)

const runHubRetention = 2 * time.Minute

type runSnapshot struct {
	RunID           int64
	ConversationID  int64
	Status          string
	Stage           string
	Message         string
	PendingQuestion string
	Steps           []model.ConversationRunStep
	HeartbeatAt     time.Time
}

type runHubEntry struct {
	snapshot    runSnapshot
	subscribers map[chan StreamEvent]struct{}
	cleanup     *time.Timer
}

type chatRunHub struct {
	mu   sync.RWMutex
	runs map[int64]*runHubEntry
}

func newChatRunHub() *chatRunHub {
	return &chatRunHub{runs: make(map[int64]*runHubEntry)}
}

func (h *chatRunHub) ensure(runID int64, conversationID int64) *runHubEntry {
	if h == nil || runID <= 0 {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	entry, ok := h.runs[runID]
	if !ok {
		entry = &runHubEntry{
			snapshot: runSnapshot{
				RunID:          runID,
				ConversationID: conversationID,
			},
			subscribers: make(map[chan StreamEvent]struct{}),
		}
		h.runs[runID] = entry
	}
	if conversationID > 0 {
		entry.snapshot.ConversationID = conversationID
	}
	if entry.cleanup != nil {
		entry.cleanup.Stop()
		entry.cleanup = nil
	}
	return entry
}

func (h *chatRunHub) update(runID int64, conversationID int64, update func(snapshot *runSnapshot)) {
	entry := h.ensure(runID, conversationID)
	if entry == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	entry = h.runs[runID]
	if entry == nil {
		return
	}
	update(&entry.snapshot)
}

func (h *chatRunHub) publish(runID int64, conversationID int64, event StreamEvent) {
	entry := h.ensure(runID, conversationID)
	if entry == nil {
		return
	}

	h.mu.Lock()
	entry = h.runs[runID]
	if entry == nil {
		h.mu.Unlock()
		return
	}
	updateRunSnapshot(&entry.snapshot, event)
	subscribers := make([]chan StreamEvent, 0, len(entry.subscribers))
	for subscriber := range entry.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	h.mu.Unlock()

	for _, subscriber := range subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (h *chatRunHub) snapshot(runID int64) (runSnapshot, bool) {
	if h == nil || runID <= 0 {
		return runSnapshot{}, false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	entry, ok := h.runs[runID]
	if !ok {
		return runSnapshot{}, false
	}
	clone := entry.snapshot
	if len(entry.snapshot.Steps) > 0 {
		clone.Steps = append([]model.ConversationRunStep(nil), entry.snapshot.Steps...)
	}
	return clone, true
}

func (h *chatRunHub) subscribe(runID int64) (<-chan StreamEvent, func(), bool) {
	if h == nil || runID <= 0 {
		return nil, nil, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	entry, ok := h.runs[runID]
	if !ok {
		return nil, nil, false
	}
	ch := make(chan StreamEvent, 32)
	entry.subscribers[ch] = struct{}{}
	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if current := h.runs[runID]; current != nil {
			delete(current.subscribers, ch)
		}
		close(ch)
	}
	return ch, cancel, true
}

func (h *chatRunHub) finish(runID int64, status string) {
	if h == nil || runID <= 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	entry, ok := h.runs[runID]
	if !ok {
		return
	}
	entry.snapshot.Status = status
	if entry.cleanup != nil {
		entry.cleanup.Stop()
	}
	entry.cleanup = time.AfterFunc(runHubRetention, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.runs, runID)
	})
}

func updateRunSnapshot(snapshot *runSnapshot, event StreamEvent) {
	if snapshot == nil {
		return
	}
	if event.RunID > 0 {
		snapshot.RunID = event.RunID
	}
	if event.Stage != "" {
		snapshot.Stage = event.Stage
	}
	if event.Status != "" {
		snapshot.Status = event.Status
	}
	switch event.Type {
	case StreamEventChunk, StreamEventSnapshot:
		snapshot.Message = event.Message
	case StreamEventQuestion:
		snapshot.PendingQuestion = event.Message
	case StreamEventHeartbeat:
		snapshot.HeartbeatAt = time.Now()
	case StreamEventStep:
		updated := false
		for i := range snapshot.Steps {
			if snapshot.Steps[i].ID == event.StepID && event.StepID > 0 {
				snapshot.Steps[i].AgentName = event.AgentName
				snapshot.Steps[i].Summary = event.Summary
				snapshot.Steps[i].Detail = event.Detail
				snapshot.Steps[i].Status = event.Status
				updated = true
				break
			}
		}
		if !updated {
			snapshot.Steps = append(snapshot.Steps, model.ConversationRunStep{
				ID:        event.StepID,
				RunID:     event.RunID,
				AgentName: event.AgentName,
				Summary:   event.Summary,
				Detail:    event.Detail,
				Status:    event.Status,
			})
		}
	}
	if event.Type != StreamEventHeartbeat {
		snapshot.HeartbeatAt = time.Now()
	}
}
