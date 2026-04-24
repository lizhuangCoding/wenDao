package chat

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"

	"wenDao/internal/model"
)

func TestConversationMemoryModel_UsesConversationMemoriesTable(t *testing.T) {
	s, err := schema.Parse(&model.ConversationMemory{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}
	if s.Table != "conversation_memories" {
		t.Fatalf("expected conversation_memories table, got %q", s.Table)
	}
}
