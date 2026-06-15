package article

import (
	"time"

	"github.com/gin-gonic/gin"

	"wenDao/internal/model"
	"wenDao/internal/pkg/response"
)

type ArticleOrbitCategory struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type ArticleOrbitCollection struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Position int    `json:"position"`
}

type ArticleOrbitSemanticPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type ArticleOrbitSemanticNeighbor struct {
	ArticleID int64   `json:"article_id"`
	Score     float64 `json:"score"`
}

type ArticleOrbitItem struct {
	ID                int64                          `json:"id"`
	Title             string                         `json:"title"`
	Slug              string                         `json:"slug"`
	Summary           string                         `json:"summary"`
	CoverImage        *string                        `json:"cover_image,omitempty"`
	ViewCount         int                            `json:"view_count"`
	CommentCount      int                            `json:"comment_count"`
	IsTop             bool                           `json:"is_top"`
	SourceType        string                         `json:"source_type"`
	Category          *ArticleOrbitCategory          `json:"category,omitempty"`
	Collection        *ArticleOrbitCollection        `json:"collection,omitempty"`
	SemanticPosition  *ArticleOrbitSemanticPosition  `json:"semantic_position,omitempty"`
	SemanticNeighbors []ArticleOrbitSemanticNeighbor `json:"semantic_neighbors,omitempty"`
	CreatedAt         string                         `json:"created_at"`
	PublishedAt       string                         `json:"published_at"`
}

func toArticleOrbitItem(article *model.Article) ArticleOrbitItem {
	publishedAt := article.CreatedAt
	if article.PublishedAt != nil {
		publishedAt = *article.PublishedAt
	}
	item := ArticleOrbitItem{
		ID:           article.ID,
		Title:        article.Title,
		Slug:         article.Slug,
		Summary:      article.Summary,
		CoverImage:   article.CoverImage,
		ViewCount:    article.ViewCount,
		CommentCount: article.CommentCount,
		IsTop:        article.IsTop,
		SourceType:   article.SourceType,
		CreatedAt:    article.CreatedAt.Format(time.RFC3339),
		PublishedAt:  publishedAt.Format(time.RFC3339),
	}
	if article.Category != nil {
		item.Category = &ArticleOrbitCategory{
			ID:   article.Category.ID,
			Name: article.Category.Name,
			Slug: article.Category.Slug,
		}
	}
	if article.CollectionMembership != nil {
		item.Collection = &ArticleOrbitCollection{
			ID:       article.CollectionMembership.CollectionID,
			Name:     article.CollectionMembership.Name,
			Slug:     article.CollectionMembership.Slug,
			Position: article.CollectionMembership.Position,
		}
	}
	if article.SemanticProfile != nil {
		item.SemanticPosition = &ArticleOrbitSemanticPosition{
			X: article.SemanticProfile.MapX,
			Y: article.SemanticProfile.MapY,
			Z: article.SemanticProfile.MapZ,
		}
		if neighbors, err := article.SemanticProfile.Neighbors(); err == nil && len(neighbors) > 0 {
			item.SemanticNeighbors = make([]ArticleOrbitSemanticNeighbor, 0, len(neighbors))
			for _, neighbor := range neighbors {
				item.SemanticNeighbors = append(item.SemanticNeighbors, ArticleOrbitSemanticNeighbor{
					ArticleID: neighbor.ArticleID,
					Score:     neighbor.Score,
				})
			}
		}
	}
	return item
}

// ListOrbitArticles 获取首页 3D 文章星球所需的轻量文章列表。
func (h *ArticleHandler) ListOrbitArticles(c *gin.Context) {
	articles, err := h.articleService.ListOrbitArticles()
	if err != nil {
		response.InternalErrorWithErr(c, "Failed to list orbit articles", err)
		return
	}

	items := make([]ArticleOrbitItem, 0, len(articles))
	for _, article := range articles {
		if article == nil {
			continue
		}
		if h.collectionService != nil {
			if err := h.collectionService.HydrateArticleCollectionData(article, false); err != nil {
				response.InternalErrorWithErr(c, "Failed to get article collection data", err)
				return
			}
		}
		items = append(items, toArticleOrbitItem(article))
	}

	response.Success(c, gin.H{
		"data":  items,
		"total": len(items),
	})
}
