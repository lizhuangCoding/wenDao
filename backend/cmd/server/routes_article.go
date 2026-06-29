package main

import "wenDao/internal/handler"

func registerArticleRoutes(
	access routeAccessGroups,
	categoryHandler *handler.CategoryHandler,
	tagHandler *handler.TagHandler,
	collectionHandler *handler.CollectionHandler,
	articleHandler *handler.ArticleHandler,
) {
	access.public.GET("/articles", articleHandler.List)
	access.public.GET("/articles/orbit", articleHandler.ListOrbitArticles)
	access.public.GET("/articles/:id", articleHandler.GetByID)
	access.public.GET("/articles/slug/:slug", articleHandler.GetBySlug)
	access.public.GET("/search/articles", articleHandler.Search)
	access.public.GET("/categories", categoryHandler.List)
	access.public.GET("/tags", tagHandler.List)
	access.public.GET("/collections", collectionHandler.List)
	access.public.GET("/categories/:id/articles", articleHandler.List)

	access.authenticated.GET("/users/me/liked-articles", articleHandler.ListLikedArticles)
	access.authenticated.GET("/users/me/favorite-articles", articleHandler.ListFavoriteArticles)
	access.authenticated.GET("/articles/:id/interaction", articleHandler.GetInteraction)
	access.authenticated.POST("/articles/:id/like", articleHandler.Like)
	access.authenticated.DELETE("/articles/:id/like", articleHandler.Unlike)
	access.authenticated.POST("/articles/:id/favorite", articleHandler.Favorite)
	access.authenticated.DELETE("/articles/:id/favorite", articleHandler.Unfavorite)
}
