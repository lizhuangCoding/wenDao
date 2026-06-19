package article

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

const (
	articleDetailCacheTTL   = 30 * time.Minute
	articleListCacheTTL     = 15 * time.Minute
	articleOrbitCacheTTL    = 15 * time.Minute
	articleListVersionKey   = "article:list:version"
	articleOrbitVersionKey  = "article:orbit:version"
	articleCacheVersionZero = int64(0)
)

type articleCacheStore interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	Del(ctx context.Context, key string) error
	Incr(ctx context.Context, key string) (int64, error)
}

// InvalidateArticleCaches 清理单篇文章的详情缓存。
func InvalidateArticleCaches(rdb *redis.Client, articleID int64, slug string) {
	if rdb == nil {
		return
	}
	ctx := context.Background()
	_ = rdb.Del(ctx, articleDetailIDKey(articleID)).Err()
	if slug != "" {
		_ = rdb.Del(ctx, articleDetailSlugKey(slug)).Err()
	}
}

// BumpArticleCollectionCacheVersions 让文章列表和文章星球缓存版本失效。
func BumpArticleCollectionCacheVersions(rdb *redis.Client) {
	if rdb == nil {
		return
	}
	ctx := context.Background()
	_, _ = rdb.Incr(ctx, articleListVersionKey).Result()
	_, _ = rdb.Incr(ctx, articleOrbitVersionKey).Result()
}

type redisArticleCacheStore struct {
	client *redis.Client
}

func newRedisArticleCacheStore(client *redis.Client) articleCacheStore {
	if client == nil {
		return nil
	}
	return &redisArticleCacheStore{client: client}
}

func (s *redisArticleCacheStore) Get(ctx context.Context, key string) (string, error) {
	return s.client.Get(ctx, key).Result()
}

func (s *redisArticleCacheStore) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return s.client.Set(ctx, key, value, expiration).Err()
}

func (s *redisArticleCacheStore) Del(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

func (s *redisArticleCacheStore) Incr(ctx context.Context, key string) (int64, error) {
	return s.client.Incr(ctx, key).Result()
}

type cachedArticleList struct {
	Articles []*model.Article `json:"articles"`
	Total    int64            `json:"total"`
}

func (s *articleService) getArticleFromCache(id int64) (*model.Article, error) {
	return s.getArticleFromCacheByKey(articleDetailIDKey(id))
}

func (s *articleService) getArticleFromSlugCache(slug string) (*model.Article, error) {
	return s.getArticleFromCacheByKey(articleDetailSlugKey(slug))
}

func (s *articleService) getArticleFromCacheByKey(key string) (*model.Article, error) {
	if s == nil || s.cache == nil {
		return nil, redis.Nil
	}
	ctx := context.Background()
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var article model.Article
	if err := json.Unmarshal([]byte(data), &article); err != nil {
		return nil, err
	}
	return &article, nil
}

func (s *articleService) setArticleToCache(article *model.Article) {
	if s == nil || s.cache == nil || article == nil {
		return
	}
	ctx := context.Background()
	data, err := json.Marshal(article)
	if err != nil {
		return
	}
	_ = s.cache.Set(ctx, articleDetailIDKey(article.ID), data, articleDetailCacheTTL)
	if article.Slug != "" {
		_ = s.cache.Set(ctx, articleDetailSlugKey(article.Slug), data, articleDetailCacheTTL)
	}
}

func (s *articleService) deleteArticleFromCache(article *model.Article) {
	if s == nil || s.cache == nil || article == nil {
		return
	}
	ctx := context.Background()
	_ = s.cache.Del(ctx, articleDetailIDKey(article.ID))
	if article.Slug != "" {
		_ = s.cache.Del(ctx, articleDetailSlugKey(article.Slug))
	}
}

func (s *articleService) getCachedArticleList(filter repository.ArticleFilter) ([]*model.Article, int64, bool) {
	if s == nil || s.cache == nil {
		return nil, 0, false
	}
	ctx := context.Background()
	key := s.articleListCacheKey(filter)
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, 0, false
	}
	var payload cachedArticleList
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return nil, 0, false
	}
	return payload.Articles, payload.Total, true
}

func (s *articleService) setCachedArticleList(filter repository.ArticleFilter, articles []*model.Article, total int64) {
	if s == nil || s.cache == nil {
		return
	}
	ctx := context.Background()
	payload, err := json.Marshal(cachedArticleList{Articles: articles, Total: total})
	if err != nil {
		return
	}
	_ = s.cache.Set(ctx, s.articleListCacheKey(filter), payload, articleListCacheTTL)
}

func (s *articleService) getCachedOrbitArticles() ([]*model.Article, bool) {
	if s == nil || s.cache == nil {
		return nil, false
	}
	ctx := context.Background()
	data, err := s.cache.Get(ctx, s.articleOrbitCacheKey())
	if err != nil {
		return nil, false
	}
	var articles []*model.Article
	if err := json.Unmarshal([]byte(data), &articles); err != nil {
		return nil, false
	}
	return articles, true
}

func (s *articleService) setCachedOrbitArticles(articles []*model.Article) {
	if s == nil || s.cache == nil {
		return
	}
	ctx := context.Background()
	payload, err := json.Marshal(articles)
	if err != nil {
		return
	}
	_ = s.cache.Set(ctx, s.articleOrbitCacheKey(), payload, articleOrbitCacheTTL)
}

func (s *articleService) invalidateArticleCollections() {
	if s == nil || s.cache == nil {
		return
	}
	ctx := context.Background()
	_, _ = s.cache.Incr(ctx, articleListVersionKey)
	_, _ = s.cache.Incr(ctx, articleOrbitVersionKey)
}

func (s *articleService) articleListCacheKey(filter repository.ArticleFilter) string {
	return fmt.Sprintf(
		"article:list:v%d:status=%s:category=%d:tag=%d:keyword=%s:sort=%t:page=%d:size=%d:ai=%s:content=%t",
		s.articleListCacheVersion(),
		filter.Status,
		filter.CategoryID,
		filter.TagID,
		url.QueryEscape(filter.Keyword),
		filter.SortByPopularity,
		filter.Page,
		filter.PageSize,
		strings.Join(filter.AIIndexStatuses, ","),
		filter.IncludeContent,
	)
}

func (s *articleService) articleOrbitCacheKey() string {
	return fmt.Sprintf("article:orbit:v%d", s.articleOrbitCacheVersion())
}

func (s *articleService) articleListCacheVersion() int64 {
	return s.cacheVersion(articleListVersionKey)
}

func (s *articleService) articleOrbitCacheVersion() int64 {
	return s.cacheVersion(articleOrbitVersionKey)
}

func (s *articleService) cacheVersion(key string) int64 {
	if s == nil || s.cache == nil {
		return articleCacheVersionZero
	}
	ctx := context.Background()
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return articleCacheVersionZero
	}
	version, err := parseCacheVersion(data)
	if err != nil {
		return articleCacheVersionZero
	}
	return version
}

func parseCacheVersion(value string) (int64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	version, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, err
	}
	return version, nil
}

func articleDetailIDKey(id int64) string {
	return fmt.Sprintf("article:detail:id:%d", id)
}

func articleDetailSlugKey(slug string) string {
	return fmt.Sprintf("article:detail:slug:%s", slug)
}
