package main

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"wenDao/config"
	"wenDao/internal/middleware"
)

type routeAccessGroups struct {
	public        *gin.RouterGroup
	optionalAuth  *gin.RouterGroup
	authenticated *gin.RouterGroup
	admin         *gin.RouterGroup
}

func newRouteAccessGroups(api *gin.RouterGroup, cfg *config.Config, rdb *redis.Client) routeAccessGroups {
	public := api.Group("")

	optionalAuth := api.Group("")
	optionalAuth.Use(middleware.AuthOptional(cfg.JWT.Secret, rdb), middleware.CSRFProtection())

	authenticated := api.Group("")
	authenticated.Use(middleware.AuthRequired(cfg.JWT.Secret, rdb), middleware.CSRFProtection())

	admin := api.Group("")
	admin.Use(
		middleware.AuthRequired(cfg.JWT.Secret, rdb),
		middleware.AdminOnly(),
		middleware.CSRFProtection(),
	)

	return routeAccessGroups{
		public:        public,
		optionalAuth:  optionalAuth,
		authenticated: authenticated,
		admin:         admin,
	}
}
