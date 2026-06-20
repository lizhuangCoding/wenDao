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
		response.Unauthorized(c, "登录状态已失效，请重新登录后操作")
		return
	}

	state, err := h.articleService.GetArticleInteractionState(userID, id)
	if err != nil {
		if errors.Is(err, svcerrors.ErrArticleNotFound) {
			response.NotFound(c, "文章不存在或已被删除")
			return
		}
		response.InternalErrorWithErr(c, "获取文章点赞收藏状态失败，请稍后重试", err)
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
		response.Unauthorized(c, "登录状态已失效，请重新登录后操作")
		return
	}

	state, err := update(userID, id)
	if err != nil {
		if errors.Is(err, svcerrors.ErrArticleNotFound) {
			response.NotFound(c, "文章不存在或已被删除，无法更新互动状态")
			return
		}
		response.InternalErrorWithErr(c, "更新文章点赞收藏状态失败，请稍后重试", err)
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
		response.Unauthorized(c, "登录状态已失效，请重新登录后操作")
		return
	}
	p := pagination.FromQuery(c)

	articles, total, err := h.articleService.ListArticlesByInteraction(userID, interactionType, p.Page, p.PageSize)
	if err != nil {
		response.InternalErrorWithErr(c, "加载互动文章列表失败，请稍后重试", err)
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
