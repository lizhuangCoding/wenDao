package article

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"wenDao/internal/pkg/async"
	asyncjobrepo "wenDao/internal/repository/asyncjob"
	asyncjobsvc "wenDao/internal/service/asyncjob"
)

func (s *articleService) updateAIIndexStatus(id int64, status string) {
	if err := s.articleRepo.UpdateAIIndexStatus(id, status); err != nil {
		s.logger.Error("Failed to update AI index status",
			zap.Int64("article_id", id),
			zap.String("status", status),
			zap.Error(err))
	}
}

func (s *articleService) vectorizeArticleAsync(id int64, title, content, slug string) {
	if s.vectorService == nil {
		return
	}

	s.updateAIIndexStatus(id, "pending")
	async.Go(context.Background(), s.logger, "vectorize article", func(ctx context.Context) error {
		if err := s.vectorService.VectorizeArticle(id, title, content, slug); err != nil {
			s.logger.Error("Failed to vectorize article",
				zap.Int64("article_id", id),
				zap.Error(err))
			s.updateAIIndexStatus(id, "failed")
			return nil
		}
		s.updateAIIndexStatus(id, "success")
		return nil
	})
}

func (s *articleService) deleteArticleVectorAsync(id int64) {
	if s.vectorService == nil {
		return
	}

	async.Go(context.Background(), s.logger, "delete article vectors", func(ctx context.Context) error {
		if err := s.vectorService.DeleteArticleVector(id); err != nil {
			s.logger.Error("Failed to delete article vectors",
				zap.Int64("article_id", id),
				zap.Error(err))
		}
		return nil
	})
}

func (s *articleService) enqueueVectorizeJob(jobRepo asyncjobrepo.AsyncJobRepository, id int64, title, content, slug string) error {
	if jobRepo == nil || s.vectorService == nil {
		return nil
	}
	job, err := asyncjobsvc.NewJob(asyncjobsvc.JobTypeArticleVectorize, asyncjobsvc.ArticleVectorizePayload{
		ArticleID: id,
		Title:     title,
		Content:   content,
		Slug:      slug,
	})
	if err != nil {
		return err
	}
	return jobRepo.Enqueue(job)
}

func (s *articleService) enqueueVectorDeleteJob(jobRepo asyncjobrepo.AsyncJobRepository, id int64) error {
	if jobRepo == nil || s.vectorService == nil {
		return nil
	}
	job, err := asyncjobsvc.NewJob(asyncjobsvc.JobTypeArticleVectorDelete, asyncjobsvc.ArticleVectorDeletePayload{
		ArticleID: id,
	})
	if err != nil {
		return err
	}
	return jobRepo.Enqueue(job)
}

func (s *articleService) enqueueOrVectorize(id int64, title, content, slug string) error {
	if s.writeTxRunner == nil {
		s.vectorizeArticleAsync(id, title, content, slug)
		return nil
	}
	return fmt.Errorf("vector enqueue must happen inside write transaction")
}
