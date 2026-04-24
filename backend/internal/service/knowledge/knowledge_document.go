package knowledge

import (
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"wenDao/internal/model"
	"wenDao/internal/pkg/hash"
	"wenDao/internal/repository"
)

// KnowledgeSourceInput 知识来源输入
type KnowledgeSourceInput struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Domain  string `json:"domain"`
	Snippet string `json:"snippet"`
}

// CreateKnowledgeDocumentInput 创建知识文档输入
type CreateKnowledgeDocumentInput struct {
	Title           string
	Summary         string
	Content         string
	CreatedByUserID int64
	Sources         []KnowledgeSourceInput
}

// KnowledgeDocumentService 知识文档服务接口
type KnowledgeDocumentService interface {
	CreateResearchDraft(input CreateKnowledgeDocumentInput) (*model.KnowledgeDocument, error)
	Approve(id int64, reviewerID int64, note string) (*model.KnowledgeDocument, error)
	Reject(id int64, reviewerID int64, note string) (*model.KnowledgeDocument, error)
	GetByID(id int64) (*model.KnowledgeDocument, []*model.KnowledgeDocumentSource, error)
	List(filter repository.KnowledgeDocumentFilter) ([]*model.KnowledgeDocument, int64, error)
	Delete(id int64) error
}

type knowledgeDocumentService struct {
	docRepo       repository.KnowledgeDocumentRepository
	sourceRepo    repository.KnowledgeDocumentSourceRepository
	vectorService VectorService
	articleRepo   repository.ArticleRepository
	categoryRepo  repository.CategoryRepository
	logger        *zap.Logger
}

// NewKnowledgeDocumentService 创建知识文档服务实例
func NewKnowledgeDocumentService(
	docRepo repository.KnowledgeDocumentRepository,
	sourceRepo repository.KnowledgeDocumentSourceRepository,
	vectorService VectorService,
	articleRepo repository.ArticleRepository,
	categoryRepo repository.CategoryRepository,
	logger *zap.Logger,
) KnowledgeDocumentService {
	return &knowledgeDocumentService{
		docRepo:       docRepo,
		sourceRepo:    sourceRepo,
		vectorService: vectorService,
		articleRepo:   articleRepo,
		categoryRepo:  categoryRepo,
		logger:        logger,
	}
}

func (s *knowledgeDocumentService) CreateResearchDraft(input CreateKnowledgeDocumentInput) (*model.KnowledgeDocument, error) {
	doc := &model.KnowledgeDocument{
		Title:           input.Title,
		Summary:         input.Summary,
		Content:         input.Content,
		Status:          model.KnowledgeDocumentStatusPendingReview,
		SourceType:      "research",
		CreatedByUserID: input.CreatedByUserID,
	}
	if err := s.docRepo.Create(doc); err != nil {
		return nil, err
	}

	sources := make([]*model.KnowledgeDocumentSource, 0, len(input.Sources))
	for i, item := range input.Sources {
		sources = append(sources, &model.KnowledgeDocumentSource{
			KnowledgeDocumentID: doc.ID,
			SourceURL:           item.URL,
			SourceTitle:         item.Title,
			SourceDomain:        item.Domain,
			SourceSnippet:       item.Snippet,
			SortOrder:           i,
		})
	}
	if len(sources) > 0 {
		if err := s.sourceRepo.CreateBatch(sources); err != nil {
			return nil, err
		}
	}

	return doc, nil
}

func (s *knowledgeDocumentService) Approve(id int64, reviewerID int64, note string) (*model.KnowledgeDocument, error) {
	doc, err := s.docRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, gorm.ErrRecordNotFound
	}

	now := time.Now()
	doc.Status = model.KnowledgeDocumentStatusApproved
	doc.ReviewedByUserID = &reviewerID
	doc.ReviewedAt = &now
	doc.ReviewNote = note
	if s.articleRepo != nil && s.categoryRepo != nil {
		if err := s.upsertGeneratedArticle(doc, reviewerID, now); err != nil {
			return nil, err
		}
	} else {
		if err := s.docRepo.Update(doc); err != nil {
			return nil, err
		}
	}

	if s.vectorService == nil {
		return doc, nil
	}
	if err := s.vectorService.VectorizeKnowledgeDocument(doc.ID, doc.Title, doc.Content); err != nil {
		return nil, err
	}
	doc.VectorizedAt = &now
	if err := s.docRepo.Update(doc); err != nil {
		return nil, err
	}

	return doc, nil
}

func (s *knowledgeDocumentService) upsertGeneratedArticle(doc *model.KnowledgeDocument, authorID int64, now time.Time) error {
	if s.articleRepo == nil || s.categoryRepo == nil {
		return nil
	}

	existing, err := s.findGeneratedArticle(doc)
	if err != nil {
		return err
	}
	if existing != nil {
		existing.Title = doc.Title
		existing.Summary = doc.Summary
		existing.Content = doc.Content
		existing.Status = "published"
		existing.SourceType = model.ArticleSourceTypeKnowledgeDocument
		existing.SourceID = &doc.ID
		if existing.PublishedAt == nil {
			existing.PublishedAt = &now
		}
		if err := s.articleRepo.Update(existing); err != nil {
			return err
		}
		s.vectorizeGeneratedArticle(existing)
		doc.ArticleID = &existing.ID
		return s.docRepo.Update(doc)
	}

	categories, err := s.categoryRepo.List()
	if err != nil {
		return err
	}
	if len(categories) == 0 {
		return errors.New("no category available for generated knowledge article")
	}

	sourceID := doc.ID
	article := &model.Article{
		Title:         doc.Title,
		Summary:       doc.Summary,
		Content:       doc.Content,
		CategoryID:    categories[0].ID,
		AuthorID:      authorID,
		Status:        "published",
		AIIndexStatus: "pending",
		SourceType:    model.ArticleSourceTypeKnowledgeDocument,
		SourceID:      &sourceID,
		PublishedAt:   &now,
	}
	if err := s.articleRepo.Create(article); err != nil {
		return err
	}
	slug := hash.GenerateSlug(article.ID)
	if err := s.articleRepo.UpdateSlug(article.ID, slug); err != nil {
		return err
	}
	article.Slug = slug
	doc.ArticleID = &article.ID
	if err := s.docRepo.Update(doc); err != nil {
		return err
	}
	if err := s.categoryRepo.IncrementArticleCount(article.CategoryID); err != nil {
		return err
	}
	s.vectorizeGeneratedArticle(article)
	return nil
}

func (s *knowledgeDocumentService) vectorizeGeneratedArticle(article *model.Article) {
	if s.vectorService == nil || article == nil || article.Status != "published" {
		return
	}
	if err := s.vectorService.VectorizeArticle(article.ID, article.Title, article.Content, article.Slug); err != nil {
		if s.logger != nil {
			s.logger.Error("Failed to vectorize generated knowledge article", zap.Int64("article_id", article.ID), zap.Error(err))
		}
		_ = s.articleRepo.UpdateAIIndexStatus(article.ID, "failed")
		return
	}
	_ = s.articleRepo.UpdateAIIndexStatus(article.ID, "success")
}

func (s *knowledgeDocumentService) findGeneratedArticle(doc *model.KnowledgeDocument) (*model.Article, error) {
	if doc.ArticleID != nil {
		article, err := s.articleRepo.GetByID(*doc.ArticleID)
		if err == nil {
			return article, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	article, err := s.articleRepo.GetBySource(model.ArticleSourceTypeKnowledgeDocument, doc.ID)
	if err == nil {
		return article, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return nil, err
}

func (s *knowledgeDocumentService) Reject(id int64, reviewerID int64, note string) (*model.KnowledgeDocument, error) {
	doc, err := s.docRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, gorm.ErrRecordNotFound
	}
	if doc.Status == model.KnowledgeDocumentStatusApproved {
		return nil, errors.New("approved document cannot be rejected")
	}

	now := time.Now()
	doc.Status = model.KnowledgeDocumentStatusRejected
	doc.ReviewedByUserID = &reviewerID
	doc.ReviewedAt = &now
	doc.ReviewNote = note
	if err := s.docRepo.Update(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *knowledgeDocumentService) GetByID(id int64) (*model.KnowledgeDocument, []*model.KnowledgeDocumentSource, error) {
	doc, err := s.docRepo.GetByID(id)
	if err != nil {
		return nil, nil, err
	}
	sources, err := s.sourceRepo.ListByDocumentID(id)
	if err != nil {
		return nil, nil, err
	}
	return doc, sources, nil
}

func (s *knowledgeDocumentService) List(filter repository.KnowledgeDocumentFilter) ([]*model.KnowledgeDocument, int64, error) {
	return s.docRepo.List(filter)
}

func (s *knowledgeDocumentService) Delete(id int64) error {
	doc, err := s.docRepo.GetByID(id)
	if err != nil {
		return err
	}
	if doc == nil {
		return gorm.ErrRecordNotFound
	}

	if s.articleRepo != nil {
		article, err := s.findGeneratedArticle(doc)
		if err != nil {
			return err
		}
		if article != nil {
			if err := s.articleRepo.Delete(article.ID); err != nil {
				return fmt.Errorf("failed to delete generated article: %w", err)
			}
			if s.categoryRepo != nil && article.Status == "published" {
				_ = s.categoryRepo.DecrementArticleCount(article.CategoryID)
			}
			if s.vectorService != nil {
				_ = s.vectorService.DeleteArticleVector(article.ID)
			}
		}
	}
	if s.vectorService != nil {
		_ = s.vectorService.DeleteKnowledgeDocumentVector(doc.ID)
	}
	if s.sourceRepo != nil {
		if err := s.sourceRepo.DeleteByDocumentID(doc.ID); err != nil {
			return err
		}
	}
	return s.docRepo.Delete(doc.ID)
}
