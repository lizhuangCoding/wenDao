package ai

import (
	"math"
	"testing"

	"go.uber.org/zap"

	"wenDao/internal/model"
)

type stubSemanticProfileRepository struct {
	deletedArticleID int64
	profile          *model.ArticleSemanticProfile
}

func (r *stubSemanticProfileRepository) Upsert(profile *model.ArticleSemanticProfile) error {
	r.profile = profile
	return nil
}

func (r *stubSemanticProfileRepository) DeleteByArticleID(articleID int64) error {
	r.deletedArticleID = articleID
	return nil
}

func TestVectorizeArticle_StoresArticleSemanticProfile(t *testing.T) {
	store := &stubVectorStore{}
	profileRepo := &stubSemanticProfileRepository{}
	svc := NewVectorService(store, &stubEmbedder{}, zap.NewNop(), profileRepo)

	if err := svc.VectorizeArticle(8, "语义知识大陆", "这是一篇用于测试语义聚类的文章正文。", "semantic-map"); err != nil {
		t.Fatalf("expected vectorization success, got %v", err)
	}
	if profileRepo.profile == nil {
		t.Fatalf("expected article semantic profile to be stored")
	}
	if profileRepo.profile.ArticleID != 8 {
		t.Fatalf("expected profile article id 8, got %d", profileRepo.profile.ArticleID)
	}
	vector, err := profileRepo.profile.Embedding()
	if err != nil {
		t.Fatalf("expected profile embedding to decode, got %v", err)
	}
	if len(vector) != 2 {
		t.Fatalf("expected profile embedding dimension 2, got %d", len(vector))
	}
	positionNorm := math.Sqrt(profileRepo.profile.MapX*profileRepo.profile.MapX + profileRepo.profile.MapY*profileRepo.profile.MapY + profileRepo.profile.MapZ*profileRepo.profile.MapZ)
	if math.Abs(positionNorm-1) > 0.000001 {
		t.Fatalf("expected semantic map position to be normalized, got %f", positionNorm)
	}
	if profileRepo.profile.ContentHash == "" {
		t.Fatalf("expected content hash to be stored")
	}
}

func TestDeleteArticleVector_DeletesArticleSemanticProfile(t *testing.T) {
	store := &stubVectorStore{}
	profileRepo := &stubSemanticProfileRepository{}
	svc := NewVectorService(store, &stubEmbedder{}, zap.NewNop(), profileRepo)

	if err := svc.DeleteArticleVector(42); err != nil {
		t.Fatalf("expected delete success, got %v", err)
	}
	if profileRepo.deletedArticleID != 42 {
		t.Fatalf("expected semantic profile for article 42 to be deleted, got %d", profileRepo.deletedArticleID)
	}
}
