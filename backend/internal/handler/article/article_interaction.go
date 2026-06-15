package article

import (
	"errors"

	"github.com/gin-gonic/gin"

	"wenDao/internal/model"
	"wenDao/internal/pkg/pagination"
	"wenDao/internal/pkg/response"
	"wenDao/internal/svcerrors"
)

func (h *ArticleHandler) GetInteraction(c *gin.Context) {
	id, ok := parseArticleIDParam(c)
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	state, err := h.articleService.GetArticleInteractionState(userID, id)
	if err != nil {
		if errors.Is(err, svcerrors.ErrArticleNotFound) {
			response.NotFound(c, "Article not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to get article interaction state", err)
		return
	}
	response.Success(c, state)
}

func (h *ArticleHandler) Like(c *gin.Context) {
	h.updateInteraction(c, h.articleService.LikeArticleForUser)
}

func (h *ArticleHandler) Unlike(c *gin.Context) {
	h.updateInteraction(c, h.articleService.UnlikeArticleForUser)
}

func (h *ArticleHandler) Favorite(c *gin.Context) {
	h.updateInteraction(c, h.articleService.FavoriteArticleForUser)
}

func (h *ArticleHandler) Unfavorite(c *gin.Context) {
	h.updateInteraction(c, h.articleService.UnfavoriteArticleForUser)
}

func (h *ArticleHandler) updateInteraction(
	c *gin.Context,
	update func(userID, articleID int64) (*model.ArticleInteractionState, error),
) {
	id, ok := parseArticleIDParam(c)
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	state, err := update(userID, id)
	if err != nil {
		if errors.Is(err, svcerrors.ErrArticleNotFound) {
			response.NotFound(c, "Article not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to update article interaction", err)
		return
	}
	response.Success(c, state)
}

func (h *ArticleHandler) ListLikedArticles(c *gin.Context) {
	h.listInteractedArticles(c, model.ArticleInteractionTypeLike)
}

func (h *ArticleHandler) ListFavoriteArticles(c *gin.Context) {
	h.listInteractedArticles(c, model.ArticleInteractionTypeFavorite)
}

func (h *ArticleHandler) listInteractedArticles(c *gin.Context, interactionType string) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	p := pagination.FromQuery(c)

	articles, total, err := h.articleService.ListArticlesByInteraction(userID, interactionType, p.Page, p.PageSize)
	if err != nil {
		response.InternalErrorWithErr(c, "Failed to list articles", err)
		return
	}

	response.Success(c, gin.H{
		"data":       articles,
		"total":      total,
		"page":       p.Page,
		"pageSize":   p.PageSize,
		"totalPages": pagination.TotalPages(total, p.PageSize),
	})
}
