package svcerrors

import "errors"

// Article errors
var (
	ErrArticleNotFound         = errors.New("article not found")
	ErrCategoryNotFound        = errors.New("category not found")
	ErrArticleAlreadyPublished = errors.New("article is already published")
	ErrArticleAlreadyDraft     = errors.New("article is already draft")
)

// Category errors
var (
	ErrSlugAlreadyExists                = errors.New("slug already exists")
	ErrCannotDeleteCategoryWithArticles = errors.New("cannot delete category with articles")
)

// Comment errors
var (
	ErrCannotCommentOnUnpublishedArticle = errors.New("cannot comment on unpublished article")
	ErrParentCommentNotFound             = errors.New("parent comment not found")
	ErrParentCommentNotBelongToArticle   = errors.New("parent comment does not belong to this article")
	ErrCannotReplyToDeletedComment       = errors.New("cannot reply to deleted comment")
	ErrCannotReplyToReplyComment         = errors.New("cannot reply to a reply comment (only two levels allowed)")
	ErrCommentNotFound                   = errors.New("comment not found")
	ErrPermissionDenied                  = errors.New("permission denied")
	ErrCommentAlreadyDeleted             = errors.New("comment already deleted")
	ErrCommentIsNotDeleted               = errors.New("comment is not deleted")
)

// Collection errors
var (
	ErrCollectionNotFound                 = errors.New("collection not found")
	ErrInvalidCollectionStatus            = errors.New("invalid collection status")
	ErrCannotDeleteCollectionWithArticles = errors.New("cannot delete collection with articles")
)

// User errors
var (
	ErrUserNotFound           = errors.New("user not found")
	ErrEmailAlreadyExists     = errors.New("email already exists")
	ErrInvalidEmailOrPassword = errors.New("invalid email or password")
	ErrAccountBanned          = errors.New("account is banned")
	ErrUsernameAlreadyExists  = errors.New("username already exists")
)

// Upload errors
var (
	ErrFileTypeNotAllowed   = errors.New("file type not allowed")
	ErrFileSizeExceedsLimit = errors.New("file size exceeds limit")
)
