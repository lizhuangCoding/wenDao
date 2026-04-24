package article

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"wenDao/internal/model"
)

// getArticleFromCache 从 Redis 缓存获取文章
func (s *articleService) getArticleFromCache(id int64) (*model.Article, error) {
	ctx := context.Background()
	key := fmt.Sprintf("article:detail:%d", id)

	data, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var article model.Article
	if err := json.Unmarshal([]byte(data), &article); err != nil {
		return nil, err
	}

	return &article, nil
}

// setArticleToCache 将文章保存到 Redis 缓存
func (s *articleService) setArticleToCache(article *model.Article) {
	ctx := context.Background()
	key := fmt.Sprintf("article:detail:%d", article.ID)

	data, err := json.Marshal(article)
	if err != nil {
		return
	}

	s.rdb.Set(ctx, key, data, 30*time.Minute)
}

// deleteArticleFromCache 从 Redis 删除文章缓存
func (s *articleService) deleteArticleFromCache(id int64) {
	ctx := context.Background()
	key := fmt.Sprintf("article:detail:%d", id)
	s.rdb.Del(ctx, key)
}
