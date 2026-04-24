package repository

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"

	"wenDao/internal/model"
)

func TestKnowledgeDocumentModel_TableAndStatusColumns(t *testing.T) {
	s, err := schema.Parse(&model.KnowledgeDocument{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	if s.Table != "knowledge_documents" {
		t.Fatalf("expected table knowledge_documents, got %q", s.Table)
	}

	statusField := s.LookUpField("Status")
	if statusField == nil {
		t.Fatalf("expected Status field to exist in schema")
	}
	if statusField.DBName != "status" {
		t.Fatalf("expected status DB column to be status, got %q", statusField.DBName)
	}
}
