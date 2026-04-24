package user

import (
	"github.com/redis/go-redis/v9"

	"wenDao/config"
	authhandler "wenDao/internal/handler/auth"
	"wenDao/internal/service"
)

type AuthHandler = authhandler.AuthHandler

func NewAuthHandler(userSvc service.UserService, cfg *config.Config, rdb *redis.Client) *AuthHandler {
	return authhandler.NewAuthHandler(userSvc, cfg, rdb)
}
