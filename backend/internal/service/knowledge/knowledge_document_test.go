package knowledge

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

type stubKnowledgeDocumentRepository struct {
	created []*model.KnowledgeDocument
	updated []*model.KnowledgeDocument
	deleted int64
	byID    *model.KnowledgeDocument
	listed  []*model.KnowledgeDocument
	total   int64
}

func (r *stubKnowledgeDocumentRepository) Create(doc *model.KnowledgeDocument) error {
	doc.ID = int64(len(r.created) + 1)
	r.created = append(r.created, doc)
	return nil
}

func (r *stubKnowledgeDocumentRepository) GetByID(id int64) (*model.KnowledgeDocument, error) {
	return r.byID, nil
}

func (r *stubKnowledgeDocumentRepository) List(filter repository.KnowledgeDocumentFilter) ([]*model.KnowledgeDocument, int64, error) {
	return r.listed, r.total, nil
}

func (r *stubKnowledgeDocumentRepository) Update(doc *model.KnowledgeDocument) error {
	r.updated = append(r.updated, doc)
	return nil
}

func (r *stubKnowledgeDocumentRepository) Delete(id int64) error {
	r.deleted = id
	return nil
}

type stubKnowledgeDocumentSourceRepository struct {
	created           []*model.KnowledgeDocumentSource
	listed            []*model.KnowledgeDocumentSource
	deletedDocumentID int64
}

func (r *stubKnowledgeDocumentSourceRepository) CreateBatch(sources []*model.KnowledgeDocumentSource) error {
	r.created = append(r.created, sources...)
	return nil
}

func (r *stubKnowledgeDocumentSourceRepository) ListByDocumentID(documentID int64) ([]*model.KnowledgeDocumentSource, error) {
	return r.listed, nil
}

func (r *stubKnowledgeDocumentSourceRepository) DeleteByDocumentID(documentID int64) error {
	r.deletedDocumentID = documentID
	return nil
}

type stubKnowledgeArticleRepository struct {
	created       []*model.Article
	updated       []*model.Article
	deleted       int64
	sourceArticle *model.Article
}

func (r *stubKnowledgeArticleRepository) Create(article *model.Article) error {
	article.ID = int64(len(r.created) + 100)
	r.created = append(r.created, article)
	return nil
}

func (r *stubKnowledgeArticleRepository) GetByID(id int64) (*model.Article, error) {
	if r.sourceArticle != nil && r.sourceArticle.ID == id {
		return r.sourceArticle, nil
	}
	return &model.Article{ID: id, Status: "published", CategoryID: 3}, nil
}

func (r *stubKnowledgeArticleRepository) GetBySlug(slug string) (*model.Article, error) {
	return nil, nil
}

func (r *stubKnowledgeArticleRepository) GetBySource(sourceType string, sourceID int64) (*model.Article, error) {
	if r.sourceArticle != nil {
		return r.sourceArticle, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *stubKnowledgeArticleRepository) List(filter repository.ArticleFilter) ([]*model.Article, int64, error) {
	return nil, 0, nil
}

func (r *stubKnowledgeArticleRepository) Update(article *model.Article) error {
	r.updated = append(r.updated, article)
	return nil
}

func (r *stubKnowledgeArticleRepository) Delete(id int64) error {
	r.deleted = id
	return nil
}

func (r *stubKnowledgeArticleRepository) UpdateSlug(id int64, slug string) error {
	if len(r.created) > 0 && r.created[len(r.created)-1].ID == id {
		r.created[len(r.created)-1].Slug = slug
	}
	return nil
}

func (r *stubKnowledgeArticleRepository) UpdateAIIndexStatus(id int64, status string) error {
	return nil
}

func (r *stubKnowledgeArticleRepository) IncrementViewCount(id int64) error    { return nil }
func (r *stubKnowledgeArticleRepository) IncrementCommentCount(id int64) error { return nil }
func (r *stubKnowledgeArticleRepository) DecrementCommentCount(id int64) error { return nil }
func (r *stubKnowledgeArticleRepository) IncrementLikeCount(id int64) error    { return nil }
func (r *stubKnowledgeArticleRepository) DecrementLikeCount(id int64) error    { return nil }
func (r *stubKnowledgeArticleRepository) UpdateTop(id int64, isTop bool) error { return nil }
func (r *stubKnowledgeArticleRepository) UpdatePopularity(id int64, popularity float64) error {
	return nil
}
func (r *stubKnowledgeArticleRepository) GetAllPublished() ([]*model.Article, error) {
	return nil, nil
}

type stubKnowledgeCategoryRepository struct {
	categories  []*model.Category
	incremented int64
	decremented int64
}

func (r *stubKnowledgeCategoryRepository) Create(category *model.Category) error { return nil }
func (r *stubKnowledgeCategoryRepository) GetByID(id int64) (*model.Category, error) {
	return &model.Category{ID: id, Name: "默认分类", Slug: "default"}, nil
}
func (r *stubKnowledgeCategoryRepository) GetBySlug(slug string) (*model.Category, error) {
	return nil, nil
}
func (r *stubKnowledgeCategoryRepository) List() ([]*model.Category, error) {
	if r.categories != nil {
		return r.categories, nil
	}
	return []*model.Category{{ID: 3, Name: "默认分类", Slug: "default"}}, nil
}
func (r *stubKnowledgeCategoryRepository) Update(category *model.Category) error { return nil }
func (r *stubKnowledgeCategoryRepository) Delete(id int64) error                 { return nil }
func (r *stubKnowledgeCategoryRepository) IncrementArticleCount(id int64) error {
	r.incremented = id
	return nil
}
func (r *stubKnowledgeCategoryRepository) DecrementArticleCount(id int64) error {
	r.decremented = id
	return nil
}

type stubKnowledgeVectorService struct {
	vectorizedDocumentID int64
	deletedDocumentID    int64
}

func (s *stubKnowledgeVectorService) VectorizeArticle(articleID int64, title, content, slug string) error {
	return nil
}

func (s *stubKnowledgeVectorService) DeleteArticleVector(articleID int64) error {
	return nil
}

func (s *stubKnowledgeVectorService) SearchArticles(query string, topK int) ([]ArticleChunk, error) {
	return nil, nil
}

func (s *stubKnowledgeVectorService) VectorizeKnowledgeDocument(documentID int64, title, content string) error {
	s.vectorizedDocumentID = documentID
	return nil
}

func (s *stubKnowledgeVectorService) DeleteKnowledgeDocumentVector(documentID int64) error {
	s.deletedDocumentID = documentID
	return nil
}

func TestKnowledgeDocumentService_CreateDraft_PersistsDocumentAndSources(t *testing.T) {
	docRepo := &stubKnowledgeDocumentRepository{}
	sourceRepo := &stubKnowledgeDocumentSourceRepository{}
	vectorSvc := &stubKnowledgeVectorService{}
	svc := NewKnowledgeDocumentService(docRepo, sourceRepo, vectorSvc, nil, nil, zap.NewNop())

	doc, err := svc.CreateResearchDraft(CreateKnowledgeDocumentInput{
		Title:           "工业大模型落地调研",
		Summary:         "总结 2025 年行业案例",
		Content:         "正文内容",
		CreatedByUserID: 9,
		Sources: []KnowledgeSourceInput{{
			URL:     "https://example.com/a",
			Title:   "案例 A",
			Domain:  "example.com",
			Snippet: "示例摘要",
		}},
	})
	if err != nil {
		t.Fatalf("expected draft to be created, got %v", err)
	}
	if doc.Status != model.KnowledgeDocumentStatusPendingReview {
		t.Fatalf("expected pending_review, got %q", doc.Status)
	}
	if len(sourceRepo.created) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sourceRepo.created))
	}
	if vectorSvc.vectorizedDocumentID != 0 {
		t.Fatalf("did not expect vectorization before approval")
	}
}

func TestKnowledgeDocumentService_Approve_TriggersVectorization(t *testing.T) {
	nowDoc := &model.KnowledgeDocument{ID: 7, Title: "标题", Content: "正文", Status: model.KnowledgeDocumentStatusPendingReview}
	docRepo := &stubKnowledgeDocumentRepository{byID: nowDoc}
	sourceRepo := &stubKnowledgeDocumentSourceRepository{}
	vectorSvc := &stubKnowledgeVectorService{}
	articleRepo := &stubKnowledgeArticleRepository{}
	categoryRepo := &stubKnowledgeCategoryRepository{}
	svc := NewKnowledgeDocumentService(docRepo, sourceRepo, vectorSvc, articleRepo, categoryRepo, zap.NewNop())

	approved, err := svc.Approve(7, 1, "审核通过")
	if err != nil {
		t.Fatalf("expected approval success, got %v", err)
	}
	if approved.Status != model.KnowledgeDocumentStatusApproved {
		t.Fatalf("expected approved, got %q", approved.Status)
	}
	if approved.ReviewedByUserID == nil || *approved.ReviewedByUserID != 1 {
		t.Fatalf("expected reviewed_by_user_id 1, got %#v", approved.ReviewedByUserID)
	}
	if approved.ReviewedAt == nil {
		t.Fatalf("expected reviewed_at to be set")
	}
	if approved.VectorizedAt == nil {
		t.Fatalf("expected vectorized_at to be set")
	}
	if vectorSvc.vectorizedDocumentID != 7 {
		t.Fatalf("expected vectorization for document 7, got %d", vectorSvc.vectorizedDocumentID)
	}
	if len(articleRepo.created) != 1 {
		t.Fatalf("expected generated article, got %d", len(articleRepo.created))
	}
	generated := articleRepo.created[0]
	if generated.Title != "标题" || generated.Content != "正文" || generated.Status != "published" {
		t.Fatalf("unexpected generated article: %#v", generated)
	}
	if generated.SourceType != model.ArticleSourceTypeKnowledgeDocument || generated.SourceID == nil || *generated.SourceID != 7 {
		t.Fatalf("expected knowledge document source link, got %#v", generated)
	}
	if approved.ArticleID == nil || *approved.ArticleID != generated.ID {
		t.Fatalf("expected document article_id %d, got %#v", generated.ID, approved.ArticleID)
	}
	if categoryRepo.incremented != generated.CategoryID {
		t.Fatalf("expected category count increment for %d, got %d", generated.CategoryID, categoryRepo.incremented)
	}
}

func TestKnowledgeDocumentService_Approve_UpdatesExistingGeneratedArticle(t *testing.T) {
	articleID := int64(42)
	nowDoc := &model.KnowledgeDocument{
		ID:        9,
		Title:     "新标题",
		Summary:   "新摘要",
		Content:   "新正文",
		Status:    model.KnowledgeDocumentStatusPendingReview,
		ArticleID: &articleID,
	}
	existing := &model.Article{ID: articleID, Title: "旧标题", Summary: "旧摘要", Content: "旧正文", Status: "published", CategoryID: 3, Slug: "existing"}
	docRepo := &stubKnowledgeDocumentRepository{byID: nowDoc}
	articleRepo := &stubKnowledgeArticleRepository{sourceArticle: existing}
	svc := NewKnowledgeDocumentService(docRepo, &stubKnowledgeDocumentSourceRepository{}, &stubKnowledgeVectorService{}, articleRepo, &stubKnowledgeCategoryRepository{}, zap.NewNop())

	approved, err := svc.Approve(9, 1, "通过")
	if err != nil {
		t.Fatalf("expected approval success, got %v", err)
	}
	if len(articleRepo.created) != 0 {
		t.Fatalf("did not expect duplicate article, got %d", len(articleRepo.created))
	}
	if len(articleRepo.updated) == 0 {
		t.Fatalf("expected existing article update")
	}
	updated := articleRepo.updated[0]
	if updated.Title != "新标题" || updated.Summary != "新摘要" || updated.Content != "新正文" {
		t.Fatalf("unexpected updated article: %#v", updated)
	}
	if approved.ArticleID == nil || *approved.ArticleID != articleID {
		t.Fatalf("expected article id to remain %d, got %#v", articleID, approved.ArticleID)
	}
}

func TestKnowledgeDocumentService_Delete_RemovesGeneratedArticleAndVectors(t *testing.T) {
	articleID := int64(55)
	doc := &model.KnowledgeDocument{ID: 10, ArticleID: &articleID, Status: model.KnowledgeDocumentStatusApproved}
	docRepo := &stubKnowledgeDocumentRepository{byID: doc}
	sourceRepo := &stubKnowledgeDocumentSourceRepository{}
	articleRepo := &stubKnowledgeArticleRepository{sourceArticle: &model.Article{ID: articleID, Status: "published", CategoryID: 3}}
	categoryRepo := &stubKnowledgeCategoryRepository{}
	vectorSvc := &stubKnowledgeVectorService{}
	svc := NewKnowledgeDocumentService(docRepo, sourceRepo, vectorSvc, articleRepo, categoryRepo, zap.NewNop())

	if err := svc.Delete(10); err != nil {
		t.Fatalf("expected delete success, got %v", err)
	}
	if articleRepo.deleted != articleID {
		t.Fatalf("expected generated article delete %d, got %d", articleID, articleRepo.deleted)
	}
	if docRepo.deleted != 10 {
		t.Fatalf("expected knowledge document delete 10, got %d", docRepo.deleted)
	}
	if sourceRepo.deletedDocumentID != 10 {
		t.Fatalf("expected document sources deleted, got %d", sourceRepo.deletedDocumentID)
	}
	if vectorSvc.deletedDocumentID != 10 {
		t.Fatalf("expected document vectors deleted, got %d", vectorSvc.deletedDocumentID)
	}
	if categoryRepo.decremented != 3 {
		t.Fatalf("expected category decrement 3, got %d", categoryRepo.decremented)
	}
}

func TestKnowledgeDocumentService_Reject_DoesNotVectorize(t *testing.T) {
	nowDoc := &model.KnowledgeDocument{ID: 8, Title: "标题", Content: "正文", Status: model.KnowledgeDocumentStatusPendingReview}
	docRepo := &stubKnowledgeDocumentRepository{byID: nowDoc}
	sourceRepo := &stubKnowledgeDocumentSourceRepository{}
	vectorSvc := &stubKnowledgeVectorService{}
	svc := NewKnowledgeDocumentService(docRepo, sourceRepo, vectorSvc, nil, nil, zap.NewNop())

	rejected, err := svc.Reject(8, 2, "信息不足")
	if err != nil {
		t.Fatalf("expected reject success, got %v", err)
	}
	if rejected.Status != model.KnowledgeDocumentStatusRejected {
		t.Fatalf("expected rejected, got %q", rejected.Status)
	}
	if rejected.ReviewedByUserID == nil || *rejected.ReviewedByUserID != 2 {
		t.Fatalf("expected reviewed_by_user_id 2, got %#v", rejected.ReviewedByUserID)
	}
	if rejected.ReviewedAt == nil {
		t.Fatalf("expected reviewed_at to be set")
	}
	if vectorSvc.vectorizedDocumentID != 0 {
		t.Fatalf("did not expect vectorization on reject")
	}
}

func TestKnowledgeDocumentService_GetByID_ReturnsDocumentAndSources(t *testing.T) {
	reviewedBy := int64(1)
	reviewedAt := time.Now()
	docRepo := &stubKnowledgeDocumentRepository{byID: &model.KnowledgeDocument{
		ID:               12,
		Title:            "知识标题",
		Status:           model.KnowledgeDocumentStatusApproved,
		ReviewedByUserID: &reviewedBy,
		ReviewedAt:       &reviewedAt,
	}}
	sourceRepo := &stubKnowledgeDocumentSourceRepository{listed: []*model.KnowledgeDocumentSource{{
		ID:                  5,
		KnowledgeDocumentID: 12,
		SourceURL:           "https://example.com/source",
	}}}
	svc := NewKnowledgeDocumentService(docRepo, sourceRepo, &stubKnowledgeVectorService{}, nil, nil, zap.NewNop())

	doc, sources, err := svc.GetByID(12)
	if err != nil {
		t.Fatalf("expected get by id success, got %v", err)
	}
	if doc.ID != 12 {
		t.Fatalf("expected document 12, got %d", doc.ID)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
}
