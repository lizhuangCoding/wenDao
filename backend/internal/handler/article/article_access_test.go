package article

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"wenDao/internal/model"
	"wenDao/internal/service/setting"
)

type stubArticleService struct {
	articleByID        *model.Article
	articleBySlug      *model.Article
	updatedArticle     *model.Article
	incrViewCountIDs   []int64
	listPage           int
	listPageSize       int
	batchIDs           []int64
	orbitArticles      []*model.Article
	scheduledSetID     int64
	scheduledSetAt     *time.Time
	draftIDs           []int64
	likedUserID        int64
	likedArticleID     int64
	favoritedUserID    int64
	favoritedArticleID int64
	stateUserID        int64
	stateArticleID     int64
	listUserID         int64
	listType           string
}

func (s *stubArticleService) Create(title, content, summary string, categoryID, authorID int64, coverImage *string, status string) (*model.Article, error) {
	return nil, nil
}
func (s *stubArticleService) GetByID(id int64) (*model.Article, error) { return s.articleByID, nil }
func (s *stubArticleService) GetBySlug(slug string) (*model.Article, error) {
	return s.articleBySlug, nil
}
func (s *stubArticleService) List(status string, categoryID int64, keyword string, sortByPopularity bool, page, pageSize int) ([]*model.Article, int64, error) {
	s.listPage = page
	s.listPageSize = pageSize
	return nil, 0, nil
}
func (s *stubArticleService) ListOrbitArticles() ([]*model.Article, error) {
	return s.orbitArticles, nil
}
func (s *stubArticleService) Update(id int64, title, content, summary string, categoryID int64, coverImage *string) (*model.Article, error) {
	if s.updatedArticle != nil {
		return s.updatedArticle, nil
	}
	return &model.Article{ID: id, Status: "draft"}, nil
}
func (s *stubArticleService) Delete(id int64) error { return nil }
func (s *stubArticleService) DeleteBatch(ids []int64) error {
	s.batchIDs = ids
	return nil
}
func (s *stubArticleService) Publish(id int64) error                                  { return nil }
func (s *stubArticleService) Draft(id int64) error                                    { s.draftIDs = append(s.draftIDs, id); return nil }
func (s *stubArticleService) AutoSave(id int64, title, content, summary string) error { return nil }
func (s *stubArticleService) IncrViewCount(id int64) error {
	s.incrViewCountIDs = append(s.incrViewCountIDs, id)
	return nil
}
func (s *stubArticleService) LikeArticle(id int64) error   { return nil }
func (s *stubArticleService) UnlikeArticle(id int64) error { return nil }
func (s *stubArticleService) LikeArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	s.likedUserID = userID
	s.likedArticleID = articleID
	return &model.ArticleInteractionState{Liked: true}, nil
}
func (s *stubArticleService) UnlikeArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return &model.ArticleInteractionState{}, nil
}
func (s *stubArticleService) FavoriteArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	s.favoritedUserID = userID
	s.favoritedArticleID = articleID
	return &model.ArticleInteractionState{Favorited: true}, nil
}
func (s *stubArticleService) UnfavoriteArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return &model.ArticleInteractionState{}, nil
}
func (s *stubArticleService) GetArticleInteractionState(userID, articleID int64) (*model.ArticleInteractionState, error) {
	s.stateUserID = userID
	s.stateArticleID = articleID
	return &model.ArticleInteractionState{Liked: true, Favorited: true}, nil
}
func (s *stubArticleService) ListArticlesByInteraction(userID int64, interactionType string, page, pageSize int) ([]*model.Article, int64, error) {
	s.listUserID = userID
	s.listType = interactionType
	s.listPage = page
	s.listPageSize = pageSize
	return []*model.Article{}, 0, nil
}
func (s *stubArticleService) ToggleTop(id int64) (*model.Article, error) { return nil, nil }
func (s *stubArticleService) UpdatePopularityScores() error              { return nil }
func (s *stubArticleService) GetAllPublished() ([]*model.Article, error) { return nil, nil }
func (s *stubArticleService) GetDueScheduledArticles() ([]*model.Article, error) {
	return nil, nil
}
func (s *stubArticleService) PublishScheduled(articleID int64) error { return nil }
func (s *stubArticleService) SetScheduledPublishAt(articleID int64, scheduledAt *time.Time) error {
	s.scheduledSetID = articleID
	s.scheduledSetAt = scheduledAt
	return nil
}

type stubSettingService struct{}

func (s *stubSettingService) GetSortByPopularity() bool              { return false }
func (s *stubSettingService) SetSortByPopularity(enabled bool) error { return nil }
func (s *stubSettingService) GetSlogan() string                      { return "" }
func (s *stubSettingService) SetSlogan(slogan string) error          { return nil }
func (s *stubSettingService) GetContactLinks() ([]setting.ContactLink, bool) {
	return nil, false
}
func (s *stubSettingService) SetContactLinks(links []setting.ContactLink) error { return nil }

func TestArticleHandlerGetByID_HidesDraftFromPublicRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articleSvc := &stubArticleService{
		articleByID: &model.Article{ID: 18, Status: "draft", Title: "secret"},
	}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/articles/18", nil)
	c.Params = gin.Params{{Key: "id", Value: "18"}}

	h.GetByID(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for public draft access, got %d", w.Code)
	}
	if len(articleSvc.incrViewCountIDs) != 0 {
		t.Fatalf("expected no view count increments for hidden draft, got %v", articleSvc.incrViewCountIDs)
	}
}

func TestArticleHandlerGetByID_AllowsAdminToReadDraft(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articleSvc := &stubArticleService{
		articleByID: &model.Article{ID: 18, Status: "draft", Title: "secret"},
	}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/articles/18", nil)
	c.Params = gin.Params{{Key: "id", Value: "18"}}
	c.Set("user_role", "admin")

	h.GetByID(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for admin draft access, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["code"].(float64) != 0 {
		t.Fatalf("expected success response, got %#v", body)
	}
	if len(articleSvc.incrViewCountIDs) != 0 {
		t.Fatalf("expected admin article fetch to avoid public view increments, got %v", articleSvc.incrViewCountIDs)
	}
}

func TestArticleHandlerList_AcceptsCamelCasePageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articleSvc := &stubArticleService{}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/articles?page=3&pageSize=9", nil)

	h.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if articleSvc.listPage != 3 || articleSvc.listPageSize != 9 {
		t.Fatalf("expected page 3 pageSize 9, got page %d pageSize %d", articleSvc.listPage, articleSvc.listPageSize)
	}
}

func TestArticleHandlerBatchDelete_DeletesSelectedArticles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articleSvc := &stubArticleService{}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/articles/batch-delete", bytes.NewBufferString(`{"ids":[4,5,5,6]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.BatchDelete(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d and body %s", w.Code, w.Body.String())
	}
	expected := []int64{4, 5, 6}
	if len(articleSvc.batchIDs) != len(expected) {
		t.Fatalf("expected ids %v, got %v", expected, articleSvc.batchIDs)
	}
	for i := range expected {
		if articleSvc.batchIDs[i] != expected[i] {
			t.Fatalf("expected ids %v, got %v", expected, articleSvc.batchIDs)
		}
	}

	var payload struct {
		Data struct {
			DeletedCount int `json:"deleted_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Data.DeletedCount != 3 {
		t.Fatalf("expected deleted_count 3, got %d", payload.Data.DeletedCount)
	}
}

func TestArticleHandlerUpdate_SavesScheduledPublishTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articleSvc := &stubArticleService{
		updatedArticle: &model.Article{ID: 42, Status: "draft"},
	}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})
	scheduledAt := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	body, err := json.Marshal(map[string]any{
		"title":                "定时文章",
		"content":              "这是一篇足够长的定时文章内容",
		"summary":              "摘要",
		"category_id":          3,
		"scheduled_publish_at": scheduledAt.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("failed to build request body: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/admin/articles/42", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.Update(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d and body %s", w.Code, w.Body.String())
	}
	if articleSvc.scheduledSetID != 42 || articleSvc.scheduledSetAt == nil || !articleSvc.scheduledSetAt.Equal(scheduledAt) {
		t.Fatalf("expected scheduled time %s to be saved for article 42, got id=%d time=%v", scheduledAt.Format(time.RFC3339), articleSvc.scheduledSetID, articleSvc.scheduledSetAt)
	}
}

func TestArticleHandlerUpdate_DraftsPublishedArticleWhenScheduling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articleSvc := &stubArticleService{
		updatedArticle: &model.Article{ID: 42, Status: "published"},
	}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})
	scheduledAt := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	body, err := json.Marshal(map[string]any{
		"title":                "重新定时",
		"content":              "这是一篇足够长的定时文章内容",
		"summary":              "摘要",
		"category_id":          3,
		"scheduled_publish_at": scheduledAt.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("failed to build request body: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/admin/articles/42", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	h.Update(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d and body %s", w.Code, w.Body.String())
	}
	if len(articleSvc.draftIDs) != 1 || articleSvc.draftIDs[0] != 42 {
		t.Fatalf("expected scheduled published article to be drafted, got %v", articleSvc.draftIDs)
	}
}

func TestArticleHandlerListOrbitArticles_ReturnsLightweightArticleNodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	createdAt := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	coverImage := "/uploads/cover.png"
	articleSvc := &stubArticleService{
		orbitArticles: []*model.Article{
			{
				ID:           7,
				Title:        "3D 星球",
				Slug:         "planet",
				Summary:      "首屏文章星球",
				Content:      "正文不应该出现在 orbit 响应中",
				CoverImage:   &coverImage,
				Status:       "published",
				SourceType:   model.ArticleSourceTypeKnowledgeDocument,
				ViewCount:    120,
				CommentCount: 5,
				IsTop:        true,
				CreatedAt:    createdAt,
				Category: &model.Category{
					ID:   3,
					Name: "AI",
					Slug: "ai",
				},
				CollectionMembership: &model.ArticleCollectionMembership{
					CollectionID: 11,
					Name:         "知识星球",
					Slug:         "knowledge-planet",
					Position:     2,
				},
			},
		},
	}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/articles/orbit", nil)

	h.ListOrbitArticles(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d and body %s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	data := body["data"].(map[string]any)
	items := data["data"].([]any)
	if data["total"].(float64) != 1 {
		t.Fatalf("expected total 1, got %#v", data["total"])
	}
	item := items[0].(map[string]any)
	if item["title"] != "3D 星球" || item["slug"] != "planet" {
		t.Fatalf("expected article identity fields, got %#v", item)
	}
	if _, ok := item["content"]; ok {
		t.Fatalf("expected orbit response to omit content, got %#v", item)
	}
	category := item["category"].(map[string]any)
	if category["name"] != "AI" || category["slug"] != "ai" {
		t.Fatalf("expected category summary, got %#v", category)
	}
	collection := item["collection"].(map[string]any)
	if collection["name"] != "知识星球" || collection["slug"] != "knowledge-planet" || collection["position"].(float64) != 2 {
		t.Fatalf("expected collection summary, got %#v", collection)
	}
}

func TestArticleHandlerLike_UsesAuthenticatedUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articleSvc := &stubArticleService{}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/articles/42/like", nil)
	c.Params = gin.Params{{Key: "id", Value: "42"}}
	c.Set("user_id", int64(7))

	h.Like(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", w.Code, w.Body.String())
	}
	if articleSvc.likedUserID != 7 || articleSvc.likedArticleID != 42 {
		t.Fatalf("expected like to use user 7 article 42, got user=%d article=%d", articleSvc.likedUserID, articleSvc.likedArticleID)
	}
}

func TestArticleHandlerFavorite_UsesAuthenticatedUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articleSvc := &stubArticleService{}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/articles/42/favorite", nil)
	c.Params = gin.Params{{Key: "id", Value: "42"}}
	c.Set("user_id", int64(7))

	h.Favorite(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", w.Code, w.Body.String())
	}
	if articleSvc.favoritedUserID != 7 || articleSvc.favoritedArticleID != 42 {
		t.Fatalf("expected favorite to use user 7 article 42, got user=%d article=%d", articleSvc.favoritedUserID, articleSvc.favoritedArticleID)
	}
}

func TestArticleHandlerGetInteraction_ReturnsUserState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articleSvc := &stubArticleService{}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/articles/42/interaction", nil)
	c.Params = gin.Params{{Key: "id", Value: "42"}}
	c.Set("user_id", int64(7))

	h.GetInteraction(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", w.Code, w.Body.String())
	}
	if articleSvc.stateUserID != 7 || articleSvc.stateArticleID != 42 {
		t.Fatalf("expected interaction state to use user 7 article 42, got user=%d article=%d", articleSvc.stateUserID, articleSvc.stateArticleID)
	}

	var payload struct {
		Data struct {
			Liked     bool `json:"liked"`
			Favorited bool `json:"favorited"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !payload.Data.Liked || !payload.Data.Favorited {
		t.Fatalf("expected liked and favorited state, got %#v", payload.Data)
	}
}

func TestArticleHandlerListLikedArticles_ReturnsPaginatedUserArticles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articleSvc := &stubArticleService{}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/users/me/liked-articles?page=2&pageSize=8", nil)
	c.Set("user_id", int64(7))

	h.ListLikedArticles(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", w.Code, w.Body.String())
	}
	if articleSvc.listUserID != 7 || articleSvc.listType != model.ArticleInteractionTypeLike {
		t.Fatalf("expected liked list to use user 7 type like, got user=%d type=%q", articleSvc.listUserID, articleSvc.listType)
	}
	if articleSvc.listPage != 2 || articleSvc.listPageSize != 8 {
		t.Fatalf("expected page 2 pageSize 8, got page=%d pageSize=%d", articleSvc.listPage, articleSvc.listPageSize)
	}
}
