package comment

import (
	"context"
	"testing"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

type replyNotificationCommentRepo struct {
	parent      *model.Comment
	created     *model.Comment
	replyAuthor *model.User
	recipient   *model.User
}

func (r *replyNotificationCommentRepo) Create(comment *model.Comment) error {
	comment.ID = 101
	created := *comment
	created.User = r.replyAuthor
	created.ReplyToUser = r.recipient
	r.created = &created
	return nil
}

func (r *replyNotificationCommentRepo) GetByID(id int64) (*model.Comment, error) {
	if r.parent != nil && id == r.parent.ID {
		return r.parent, nil
	}
	if r.created != nil && id == r.created.ID {
		return r.created, nil
	}
	return nil, nil
}

func (r *replyNotificationCommentRepo) GetByArticleID(articleID int64) ([]*model.Comment, error) {
	return nil, nil
}

func (r *replyNotificationCommentRepo) ListAll(filter repository.CommentFilter) ([]*model.Comment, int64, error) {
	return nil, 0, nil
}

func (r *replyNotificationCommentRepo) Delete(id int64) error {
	return nil
}

func (r *replyNotificationCommentRepo) Restore(id int64) error {
	return nil
}

type replyNotificationArticleRepo struct {
	article *model.Article
}

func (r *replyNotificationArticleRepo) Create(article *model.Article) error { return nil }
func (r *replyNotificationArticleRepo) GetByID(id int64) (*model.Article, error) {
	return r.article, nil
}
func (r *replyNotificationArticleRepo) GetBySlug(slug string) (*model.Article, error) {
	return nil, nil
}
func (r *replyNotificationArticleRepo) GetBySource(sourceType string, sourceID int64) (*model.Article, error) {
	return nil, nil
}
func (r *replyNotificationArticleRepo) List(filter repository.ArticleFilter) ([]*model.Article, int64, error) {
	return nil, 0, nil
}
func (r *replyNotificationArticleRepo) ListOrbitArticles() ([]*model.Article, error) {
	return nil, nil
}
func (r *replyNotificationArticleRepo) Update(article *model.Article) error { return nil }
func (r *replyNotificationArticleRepo) Delete(id int64) error               { return nil }
func (r *replyNotificationArticleRepo) UpdateSlug(id int64, slug string) error {
	return nil
}
func (r *replyNotificationArticleRepo) UpdateAIIndexStatus(id int64, status string) error {
	return nil
}
func (r *replyNotificationArticleRepo) IncrementViewCount(id int64) error    { return nil }
func (r *replyNotificationArticleRepo) IncrementCommentCount(id int64) error { return nil }
func (r *replyNotificationArticleRepo) DecrementCommentCount(id int64) error { return nil }
func (r *replyNotificationArticleRepo) IncrementLikeCount(id int64) error    { return nil }
func (r *replyNotificationArticleRepo) DecrementLikeCount(id int64) error    { return nil }
func (r *replyNotificationArticleRepo) UpdateTop(id int64, isTop bool) error { return nil }
func (r *replyNotificationArticleRepo) UpdatePopularity(id int64, popularity float64) error {
	return nil
}
func (r *replyNotificationArticleRepo) GetAllPublished() ([]*model.Article, error) {
	return nil, nil
}

type recordingReplyNotificationSender struct {
	notifications []CommentReplyNotification
}

func (s *recordingReplyNotificationSender) SendCommentReplyNotification(_ context.Context, notification CommentReplyNotification) error {
	s.notifications = append(s.notifications, notification)
	return nil
}

func TestCommentServiceCreateSendsReplyEmailNotification(t *testing.T) {
	recipient := &model.User{
		ID:                       12,
		Username:                 "reader",
		Email:                    "reader@example.com",
		CommentReplyEmailEnabled: true,
		Status:                   "active",
	}
	replyAuthor := &model.User{ID: 34, Username: "author", Email: "author@example.com", Status: "active"}
	parentID := int64(56)
	article := &model.Article{ID: 7, Title: "一篇关于长期主义的文章", Slug: "long-term", Status: "published"}
	commentRepo := &replyNotificationCommentRepo{
		parent: &model.Comment{
			ID:        parentID,
			ArticleID: article.ID,
			UserID:    recipient.ID,
			User:      recipient,
			Status:    "normal",
		},
		replyAuthor: replyAuthor,
		recipient:   recipient,
	}
	sender := &recordingReplyNotificationSender{}
	svc := NewCommentService(commentRepo, &replyNotificationArticleRepo{article: article}, WithReplyNotificationSender(sender))

	comment, err := svc.Create(article.ID, replyAuthor.ID, "谢谢你的评论，这里补充一个新的角度。", &parentID, nil)
	if err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	if comment == nil {
		t.Fatal("expected created comment")
	}
	if len(sender.notifications) != 1 {
		t.Fatalf("expected one reply notification, got %d", len(sender.notifications))
	}

	notification := sender.notifications[0]
	if notification.RecipientEmail != "reader@example.com" {
		t.Fatalf("expected reader email, got %q", notification.RecipientEmail)
	}
	if notification.ReplyAuthorUsername != "author" {
		t.Fatalf("expected reply author username, got %q", notification.ReplyAuthorUsername)
	}
	if notification.ArticleTitle != article.Title || notification.ArticleSlug != article.Slug {
		t.Fatalf("expected article details to be included, got title=%q slug=%q", notification.ArticleTitle, notification.ArticleSlug)
	}
	if notification.CommentPreview == "" {
		t.Fatalf("expected comment preview to be included")
	}
}

func TestCommentServiceCreateSkipsReplyEmailWhenRecipientDisabledPreference(t *testing.T) {
	recipient := &model.User{
		ID:                       12,
		Username:                 "reader",
		Email:                    "reader@example.com",
		CommentReplyEmailEnabled: false,
		Status:                   "active",
	}
	replyAuthor := &model.User{ID: 34, Username: "author", Email: "author@example.com", Status: "active"}
	parentID := int64(56)
	article := &model.Article{ID: 7, Title: "一篇文章", Slug: "essay", Status: "published"}
	commentRepo := &replyNotificationCommentRepo{
		parent: &model.Comment{
			ID:        parentID,
			ArticleID: article.ID,
			UserID:    recipient.ID,
			User:      recipient,
			Status:    "normal",
		},
		replyAuthor: replyAuthor,
		recipient:   recipient,
	}
	sender := &recordingReplyNotificationSender{}
	svc := NewCommentService(commentRepo, &replyNotificationArticleRepo{article: article}, WithReplyNotificationSender(sender))

	if _, err := svc.Create(article.ID, replyAuthor.ID, "收到，我稍后再看。", &parentID, nil); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	if len(sender.notifications) != 0 {
		t.Fatalf("expected no notification when preference is disabled, got %d", len(sender.notifications))
	}
}
