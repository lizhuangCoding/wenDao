package chat

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"

	"wenDao/internal/model"
)

func TestConversationRunModel_UsesConversationRunsTable(t *testing.T) {
	s, err := schema.Parse(&model.ConversationRun{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	if s.Table != "conversation_runs" {
		t.Fatalf("expected table conversation_runs, got %q", s.Table)
	}
	for _, column := range []string{
		"provider",
		"model_name",
		"prompt_tokens",
		"completion_tokens",
		"estimated_cost",
		"cost_currency",
		"cost_status",
		"source_quality_score",
		"failure_category",
		"failure_fingerprint",
	} {
		if _, ok := s.FieldsByDBName[column]; !ok {
			t.Fatalf("expected conversation_runs model to include column %q", column)
		}
	}
}

func TestConversationRunStepModel_UsesConversationRunStepsTable(t *testing.T) {
	s, err := schema.Parse(&model.ConversationRunStep{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	if s.Table != "conversation_run_steps" {
		t.Fatalf("expected table conversation_run_steps, got %q", s.Table)
	}
}
