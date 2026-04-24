package service

import (
	"testing"

	"go.uber.org/zap"

	"wenDao/internal/pkg/eino"
)

type stubVectorStore struct {
	items []eino.VectorItem
}

func (s *stubVectorStore) InitIndex(indexName string, dim int) error { return nil }
func (s *stubVectorStore) Upsert(key string, vector []float32, metadata map[string]interface{}) error {
	s.items = append(s.items, eino.VectorItem{Key: key, Vector: vector, Metadata: metadata})
	return nil
}
func (s *stubVectorStore) UpsertBatch(items []eino.VectorItem) error {
	s.items = append(s.items, items...)
	return nil
}
func (s *stubVectorStore) Delete(pattern string) error { return nil }
func (s *stubVectorStore) Search(vector []float32, topK int) ([]eino.SearchResult, error) { return nil, nil }

type stubEmbedder struct{}

func (s *stubEmbedder) Embed(text string) ([]float32, error) { return []float32{0.1, 0.2}, nil }
func (s *stubEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	result := make([][]float32, 0, len(texts))
	for range texts {
		result = append(result, []float32{0.1, 0.2})
	}
	return result, nil
}

func TestVectorizeArticle_IncludesSlugInMetadata(t *testing.T) {
	store := &stubVectorStore{}
	svc := NewVectorService(store, &stubEmbedder{}, zap.NewNop())

	if err := svc.VectorizeArticle(8, "李小龙的功夫哲学", "正文内容", "lee-philosophy"); err != nil {
		t.Fatalf("expected vectorization success, got %v", err)
	}
	if len(store.items) == 0 {
		t.Fatalf("expected vector items to be stored")
	}
	if store.items[0].Metadata["slug"] != "lee-philosophy" {
		t.Fatalf("expected slug metadata, got %#v", store.items[0].Metadata["slug"])
	}
}
