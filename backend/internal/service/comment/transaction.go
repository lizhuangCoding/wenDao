package comment

import (
	"gorm.io/gorm"

	"wenDao/internal/repository"
	articlerepo "wenDao/internal/repository/article"
	asyncjobrepo "wenDao/internal/repository/asyncjob"
	commentrepo "wenDao/internal/repository/comment"
)

type WriteTransactionRunner interface {
	Run(func(repository.CommentRepository, repository.ArticleRepository, asyncjobrepo.AsyncJobRepository) error) error
}

type gormWriteTransactionRunner struct {
	db *gorm.DB
}

func NewGormWriteTransactionRunner(db *gorm.DB) WriteTransactionRunner {
	if db == nil {
		return nil
	}
	return &gormWriteTransactionRunner{db: db}
}

func (r *gormWriteTransactionRunner) Run(fn func(repository.CommentRepository, repository.ArticleRepository, asyncjobrepo.AsyncJobRepository) error) error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(
			commentrepo.NewCommentRepository(tx),
			articlerepo.NewArticleRepository(tx),
			asyncjobrepo.NewAsyncJobRepository(tx),
		)
	})
}
