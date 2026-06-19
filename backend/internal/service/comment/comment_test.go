package comment

import (
	"context"
	"strings"
	"testing"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

type actorUserRepo struct {
	users map[int64]*model.User
}

func (r *actorUserRepo) Create(user *model.User) error { return nil }
func (r *actorUserRepo) GetByID(id int64) (*model.User, error) {
	if user, ok := r.users[id]; ok {
		return user, nil
	}
	return nil, nil
}
func (r *actorUserRepo) GetByEmail(email string) (*model.User, error)       { return nil, nil }
func (r *actorUserRepo) GetByUsername(username string) (*model.User, error) { return nil, nil }
func (r *actorUserRepo) GetByOAuth(provider string, oauthID string) (*model.User, error) {
	return nil, nil
}
func (r *actorUserRepo) Update(user *model.User) error { return nil }
func (r *actorUserRepo) ListUsers(page, pageSize int, role, status, search string) ([]*model.User, int64, error) {
	return nil, 0, nil
}
func (r *actorUserRepo) UpdateUserRole(userID int64, role string) error     { return nil }
func (r *actorUserRepo) UpdateUserStatus(userID int64, status string) error { return nil }
func (r *actorUserRepo) GetAllActiveUserIDs() ([]int64, error)              { return nil, nil }

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

func (r *replyNotificationCommentRepo) GetByArticleIDSorted(articleID int64, sort string) ([]*model.Comment, error) {
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

func (r *replyNotificationCommentRepo) IncrementLike(id int64) error {
	return nil
}

func (r *replyNotificationCommentRepo) DecrementLike(id int64) error {
	return nil
}

func (r *replyNotificationCommentRepo) IncrementDislike(id int64) error {
	return nil
}

func (r *replyNotificationCommentRepo) DecrementDislike(id int64) error {
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
func (r *replyNotificationArticleRepo) Search(filter repository.ArticleSearchFilter) ([]repository.ArticleSearchResult, int64, error) {
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
func (r *replyNotificationArticleRepo) GetDueScheduledArticles() ([]*model.Article, error) {
	return nil, nil
}
func (r *replyNotificationArticleRepo) PublishScheduled(articleID int64) error {
	return nil
}
func (r *replyNotificationArticleRepo) AddInteraction(userID, articleID int64, interactionType string) (bool, error) {
	return false, nil
}
func (r *replyNotificationArticleRepo) RemoveInteraction(userID, articleID int64, interactionType string) (bool, error) {
	return false, nil
}
func (r *replyNotificationArticleRepo) GetInteractionState(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return nil, nil
}
func (r *replyNotificationArticleRepo) ListByInteraction(userID int64, interactionType string, filter repository.ArticleFilter) ([]*model.Article, int64, error) {
	return nil, 0, nil
}

type recordingReplyNotificationSender struct {
	notifications []CommentReplyNotification
}

func (s *recordingReplyNotificationSender) SendCommentReplyNotification(_ context.Context, notification CommentReplyNotification) error {
	s.notifications = append(s.notifications, notification)
	return nil
}

type voteCommentRepo struct {
	decrementLikeID    int64
	decrementDislikeID int64
	incrementDislikeID int64
}

func (r *voteCommentRepo) Create(comment *model.Comment) error { return nil }
func (r *voteCommentRepo) GetByID(id int64) (*model.Comment, error) {
	return nil, nil
}
func (r *voteCommentRepo) GetByArticleID(articleID int64) ([]*model.Comment, error) {
	return nil, nil
}
func (r *voteCommentRepo) GetByArticleIDSorted(articleID int64, sort string) ([]*model.Comment, error) {
	return nil, nil
}
func (r *voteCommentRepo) ListAll(filter repository.CommentFilter) ([]*model.Comment, int64, error) {
	return nil, 0, nil
}
func (r *voteCommentRepo) Delete(id int64) error        { return nil }
func (r *voteCommentRepo) Restore(id int64) error       { return nil }
func (r *voteCommentRepo) IncrementLike(id int64) error { return nil }
func (r *voteCommentRepo) IncrementDislike(id int64) error {
	r.incrementDislikeID = id
	return nil
}
func (r *voteCommentRepo) DecrementLike(id int64) error {
	r.decrementLikeID = id
	return nil
}
func (r *voteCommentRepo) DecrementDislike(id int64) error {
	r.decrementDislikeID = id
	return nil
}

type likeNotificationCommentRepo struct {
	comment            *model.Comment
	incrementLikeID    int64
	decrementLikeID    int64
	incrementDislikeID int64
	decrementDislikeID int64
}

func (r *likeNotificationCommentRepo) Create(comment *model.Comment) error { return nil }
func (r *likeNotificationCommentRepo) GetByID(id int64) (*model.Comment, error) {
	return r.comment, nil
}
func (r *likeNotificationCommentRepo) GetByArticleID(articleID int64) ([]*model.Comment, error) {
	return nil, nil
}
func (r *likeNotificationCommentRepo) GetByArticleIDSorted(articleID int64, sort string) ([]*model.Comment, error) {
	return nil, nil
}
func (r *likeNotificationCommentRepo) ListAll(filter repository.CommentFilter) ([]*model.Comment, int64, error) {
	return nil, 0, nil
}
func (r *likeNotificationCommentRepo) Delete(id int64) error  { return nil }
func (r *likeNotificationCommentRepo) Restore(id int64) error { return nil }
func (r *likeNotificationCommentRepo) IncrementLike(id int64) error {
	r.incrementLikeID = id
	return nil
}
func (r *likeNotificationCommentRepo) DecrementLike(id int64) error {
	r.decrementLikeID = id
	return nil
}
func (r *likeNotificationCommentRepo) IncrementDislike(id int64) error {
	r.incrementDislikeID = id
	return nil
}
func (r *likeNotificationCommentRepo) DecrementDislike(id int64) error {
	r.decrementDislikeID = id
	return nil
}

type recordingNotificationService struct {
	creates []struct {
		userID    int64
		notifType string
		title     string
		content   string
		linkURL   string
	}
}

func (s *recordingNotificationService) Create(userID int64, notifType, title, content, linkURL string) error {
	s.creates = append(s.creates, struct {
		userID    int64
		notifType string
		title     string
		content   string
		linkURL   string
	}{
		userID:    userID,
		notifType: notifType,
		title:     title,
		content:   content,
		linkURL:   linkURL,
	})
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
	notifSvc := &recordingNotificationService{}
	svc := NewCommentService(
		commentRepo,
		&replyNotificationArticleRepo{article: article},
		WithReplyNotificationSender(sender),
		WithNotificationService(notifSvc),
	)

	if _, err := svc.Create(article.ID, replyAuthor.ID, "收到，我稍后再看。", &parentID, nil); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	if len(sender.notifications) != 0 {
		t.Fatalf("expected no notification when preference is disabled, got %d", len(sender.notifications))
	}
	if len(notifSvc.creates) != 1 {
		t.Fatalf("expected one in-app notification even when email is disabled, got %d", len(notifSvc.creates))
	}
	if notifSvc.creates[0].notifType != model.NotificationTypeCommentReply {
		t.Fatalf("expected comment reply in-app notification, got %q", notifSvc.creates[0].notifType)
	}
}

func TestCommentServiceUnlikeDecrementsLikeCount(t *testing.T) {
	commentRepo := &likeNotificationCommentRepo{
		comment: &model.Comment{
			ID:        42,
			ArticleID: 7,
			UserID:    88,
			Status:    "normal",
		},
	}
	svc := NewCommentService(commentRepo, &replyNotificationArticleRepo{})

	if err := svc.Unlike(42, 7); err != nil {
		t.Fatalf("expected unlike to succeed, got %v", err)
	}
	if commentRepo.decrementLikeID != 42 {
		t.Fatalf("expected comment 42 to be unliked, got %d", commentRepo.decrementLikeID)
	}
}

func TestCommentServiceUndislikeDecrementsDislikeCount(t *testing.T) {
	commentRepo := &likeNotificationCommentRepo{
		comment: &model.Comment{
			ID:        43,
			ArticleID: 7,
			UserID:    88,
			Status:    "normal",
		},
	}
	svc := NewCommentService(commentRepo, &replyNotificationArticleRepo{})

	if err := svc.Undislike(43, 7); err != nil {
		t.Fatalf("expected undislike to succeed, got %v", err)
	}
	if commentRepo.decrementDislikeID != 43 {
		t.Fatalf("expected comment 43 to be undisliked, got %d", commentRepo.decrementDislikeID)
	}
}

func TestCommentServiceDislikeCreatesCommentDislikeNotification(t *testing.T) {
	author := &model.User{ID: 88, Username: "writer", Status: "active"}
	article := &model.Article{ID: 7, Title: "评论文章", Slug: "comment-article", Status: "published"}
	commentRepo := &likeNotificationCommentRepo{
		comment: &model.Comment{
			ID:        42,
			ArticleID: article.ID,
			UserID:    author.ID,
			User:      author,
			Content:   "这是一条会收到点踩的评论",
			Status:    "normal",
		},
	}
	notifSvc := &recordingNotificationService{}
	userRepo := &actorUserRepo{users: map[int64]*model.User{7: {ID: 7, Username: "alice"}}}
	svc := NewCommentService(commentRepo, &replyNotificationArticleRepo{article: article}, WithNotificationService(notifSvc), WithUserRepository(userRepo))

	if err := svc.Dislike(42, 7); err != nil {
		t.Fatalf("expected dislike to succeed, got %v", err)
	}
	if commentRepo.incrementDislikeID != 42 {
		t.Fatalf("expected comment 42 to be disliked, got %d", commentRepo.incrementDislikeID)
	}
	if len(notifSvc.creates) != 1 {
		t.Fatalf("expected one notification, got %d", len(notifSvc.creates))
	}

	created := notifSvc.creates[0]
	if created.userID != author.ID {
		t.Fatalf("expected notification for author %d, got %d", author.ID, created.userID)
	}
	if created.notifType != model.NotificationTypeCommentLike {
		t.Fatalf("expected comment_like notification, got %q", created.notifType)
	}
	if created.linkURL != "/article/comment-article" {
		t.Fatalf("expected article link, got %q", created.linkURL)
	}
	if created.title == "" || created.content == "" {
		t.Fatalf("expected title and content to be populated, got %#v", created)
	}
	if created.title != "alice点踩了你的评论" {
		t.Fatalf("expected actor username in title, got %q", created.title)
	}
	if !strings.Contains(created.content, "在《评论文章》中，alice点踩了你的评论") {
		t.Fatalf("expected actor username in content, got %q", created.content)
	}
}

func TestCommentServiceUndislikeCreatesCommentDislikeNotification(t *testing.T) {
	author := &model.User{ID: 88, Username: "writer", Status: "active"}
	article := &model.Article{ID: 7, Title: "评论文章", Slug: "comment-article", Status: "published"}
	commentRepo := &likeNotificationCommentRepo{
		comment: &model.Comment{
			ID:        42,
			ArticleID: article.ID,
			UserID:    author.ID,
			User:      author,
			Content:   "这是一条会收到点踩的评论",
			Status:    "normal",
		},
	}
	notifSvc := &recordingNotificationService{}
	userRepo := &actorUserRepo{users: map[int64]*model.User{7: {ID: 7, Username: "alice"}}}
	svc := NewCommentService(commentRepo, &replyNotificationArticleRepo{article: article}, WithNotificationService(notifSvc), WithUserRepository(userRepo))

	if err := svc.Undislike(42, 7); err != nil {
		t.Fatalf("expected undislike to succeed, got %v", err)
	}
	if commentRepo.decrementDislikeID != 42 {
		t.Fatalf("expected comment 42 to be undisliked, got %d", commentRepo.decrementDislikeID)
	}
	if len(notifSvc.creates) != 1 {
		t.Fatalf("expected one notification, got %d", len(notifSvc.creates))
	}
	if notifSvc.creates[0].notifType != model.NotificationTypeCommentLike {
		t.Fatalf("expected comment_like notification, got %q", notifSvc.creates[0].notifType)
	}
	if notifSvc.creates[0].title == "" || notifSvc.creates[0].content == "" {
		t.Fatalf("expected undislike notification content to be populated, got %#v", notifSvc.creates[0])
	}
	if notifSvc.creates[0].title != "alice取消了对你评论的点踩" {
		t.Fatalf("expected actor username in title, got %q", notifSvc.creates[0].title)
	}
}

func TestCommentServiceDislikeSkipsSelfNotification(t *testing.T) {
	author := &model.User{ID: 88, Username: "writer", Status: "active"}
	article := &model.Article{ID: 7, Title: "评论文章", Slug: "comment-article", Status: "published"}
	commentRepo := &likeNotificationCommentRepo{
		comment: &model.Comment{
			ID:        42,
			ArticleID: article.ID,
			UserID:    author.ID,
			User:      author,
			Content:   "这是一条会收到点踩的评论",
			Status:    "normal",
		},
	}
	notifSvc := &recordingNotificationService{}
	userRepo := &actorUserRepo{users: map[int64]*model.User{7: {ID: 7, Username: "alice"}}}
	svc := NewCommentService(commentRepo, &replyNotificationArticleRepo{article: article}, WithNotificationService(notifSvc), WithUserRepository(userRepo))

	if err := svc.Dislike(42, author.ID); err != nil {
		t.Fatalf("expected dislike to succeed, got %v", err)
	}
	if len(notifSvc.creates) != 0 {
		t.Fatalf("expected no notification for self-dislike, got %d", len(notifSvc.creates))
	}
}

func TestCommentServiceLikeCreatesCommentLikeNotification(t *testing.T) {
	author := &model.User{ID: 88, Username: "writer", Status: "active"}
	article := &model.Article{ID: 7, Title: "评论文章", Slug: "comment-article", Status: "published"}
	commentRepo := &likeNotificationCommentRepo{
		comment: &model.Comment{
			ID:        42,
			ArticleID: article.ID,
			UserID:    author.ID,
			User:      author,
			Content:   "这是一条会收到点赞的评论",
			Status:    "normal",
		},
	}
	notifSvc := &recordingNotificationService{}
	userRepo := &actorUserRepo{users: map[int64]*model.User{7: {ID: 7, Username: "alice"}}}
	svc := NewCommentService(commentRepo, &replyNotificationArticleRepo{article: article}, WithNotificationService(notifSvc), WithUserRepository(userRepo))

	if err := svc.Like(42, 7); err != nil {
		t.Fatalf("expected like to succeed, got %v", err)
	}
	if commentRepo.incrementLikeID != 42 {
		t.Fatalf("expected comment 42 to be liked, got %d", commentRepo.incrementLikeID)
	}
	if len(notifSvc.creates) != 1 {
		t.Fatalf("expected one notification, got %d", len(notifSvc.creates))
	}

	created := notifSvc.creates[0]
	if created.userID != author.ID {
		t.Fatalf("expected notification for author %d, got %d", author.ID, created.userID)
	}
	if created.notifType != model.NotificationTypeCommentLike {
		t.Fatalf("expected comment_like notification, got %q", created.notifType)
	}
	if created.linkURL != "/article/comment-article" {
		t.Fatalf("expected article link, got %q", created.linkURL)
	}
	if created.title == "" || created.content == "" {
		t.Fatalf("expected title and content to be populated, got %#v", created)
	}
	if created.title != "alice点赞了你的评论" {
		t.Fatalf("expected actor username in title, got %q", created.title)
	}
	if !strings.Contains(created.content, "在《评论文章》中，alice点赞了你的评论") {
		t.Fatalf("expected actor username in content, got %q", created.content)
	}
}

func TestCommentServiceUnlikeCreatesCommentLikeNotification(t *testing.T) {
	author := &model.User{ID: 88, Username: "writer", Status: "active"}
	article := &model.Article{ID: 7, Title: "评论文章", Slug: "comment-article", Status: "published"}
	commentRepo := &likeNotificationCommentRepo{
		comment: &model.Comment{
			ID:        42,
			ArticleID: article.ID,
			UserID:    author.ID,
			User:      author,
			Content:   "这是一条会收到点赞的评论",
			Status:    "normal",
		},
	}
	notifSvc := &recordingNotificationService{}
	userRepo := &actorUserRepo{users: map[int64]*model.User{7: {ID: 7, Username: "alice"}}}
	svc := NewCommentService(commentRepo, &replyNotificationArticleRepo{article: article}, WithNotificationService(notifSvc), WithUserRepository(userRepo))

	if err := svc.Unlike(42, 7); err != nil {
		t.Fatalf("expected unlike to succeed, got %v", err)
	}
	if commentRepo.decrementLikeID != 42 {
		t.Fatalf("expected comment 42 to be unliked, got %d", commentRepo.decrementLikeID)
	}
	if len(notifSvc.creates) != 1 {
		t.Fatalf("expected one notification, got %d", len(notifSvc.creates))
	}
	if notifSvc.creates[0].notifType != model.NotificationTypeCommentLike {
		t.Fatalf("expected comment_like notification, got %q", notifSvc.creates[0].notifType)
	}
	if notifSvc.creates[0].title == "" || notifSvc.creates[0].content == "" {
		t.Fatalf("expected unlike notification content to be populated, got %#v", notifSvc.creates[0])
	}
	if notifSvc.creates[0].title != "alice取消了对你评论的点赞" {
		t.Fatalf("expected actor username in title, got %q", notifSvc.creates[0].title)
	}
}

func TestCommentServiceLikeSkipsSelfNotification(t *testing.T) {
	author := &model.User{ID: 88, Username: "writer", Status: "active"}
	article := &model.Article{ID: 7, Title: "评论文章", Slug: "comment-article", Status: "published"}
	commentRepo := &likeNotificationCommentRepo{
		comment: &model.Comment{
			ID:        42,
			ArticleID: article.ID,
			UserID:    author.ID,
			User:      author,
			Content:   "这是一条会收到点赞的评论",
			Status:    "normal",
		},
	}
	notifSvc := &recordingNotificationService{}
	userRepo := &actorUserRepo{users: map[int64]*model.User{7: {ID: 7, Username: "alice"}}}
	svc := NewCommentService(commentRepo, &replyNotificationArticleRepo{article: article}, WithNotificationService(notifSvc), WithUserRepository(userRepo))

	if err := svc.Like(42, author.ID); err != nil {
		t.Fatalf("expected like to succeed, got %v", err)
	}
	if len(notifSvc.creates) != 0 {
		t.Fatalf("expected no notification for self-like, got %d", len(notifSvc.creates))
	}
}
