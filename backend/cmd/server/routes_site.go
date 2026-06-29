package main

import "wenDao/internal/handler"

func registerSiteRoutes(
	access routeAccessGroups,
	siteHandler *handler.SiteHandler,
	articleHandler *handler.ArticleHandler,
) {
	access.public.GET("/slogan", siteHandler.GetSlogan)
	access.public.GET("/contact-links", siteHandler.GetContactLinks)
	access.public.GET("/settings/sort-mode", articleHandler.GetSortMode)
}
