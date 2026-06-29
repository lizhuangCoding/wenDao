package article

import (
	"gorm.io/gorm"

	"wenDao/internal/repository"
	articlerepo "wenDao/internal/repository/article"
	asyncjobrepo "wenDao/internal/repository/asyncjob"
	categoryrepo "wenDao/internal/repository/category"
)

type WriteTransactionRunner interface {
	Run(func(repository.ArticleRepository, repository.CategoryRepository, asyncjobrepo.AsyncJobRepository) error) error
}

type ArticleServiceOption func(*articleService)

type gormWriteTransactionRunner struct {
	db *gorm.DB
}

func NewGormWriteTransactionRunner(db *gorm.DB) WriteTransactionRunner {
	if db == nil {
		return nil
	}
	return &gormWriteTransactionRunner{db: db}
}

func WithWriteTransactionRunner(runner WriteTransactionRunner) ArticleServiceOption {
	return func(s *articleService) {
		s.writeTxRunner = runner
	}
}

func (r *gormWriteTransactionRunner) Run(fn func(repository.ArticleRepository, repository.CategoryRepository, asyncjobrepo.AsyncJobRepository) error) error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(
			articlerepo.NewArticleRepository(tx),
			categoryrepo.NewCategoryRepository(tx),
			asyncjobrepo.NewAsyncJobRepository(tx),
		)
	})
}
