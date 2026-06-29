package main

import (
	"time"

	"wenDao/internal/middleware"
)

type routeRateLimitFactory struct{}

func newRouteRateLimitFactory() routeRateLimitFactory {
	return routeRateLimitFactory{}
}

func (routeRateLimitFactory) ip(name string, limit int, window time.Duration, action string) middleware.RateLimitConfig {
	return newRouteRateLimitConfig(name, middleware.IPLimit, limit, window, action)
}

func (routeRateLimitFactory) user(name string, limit int, window time.Duration, action string) middleware.RateLimitConfig {
	return newRouteRateLimitConfig(name, middleware.UserLimit, limit, window, action)
}

func newRouteRateLimitConfig(name string, limitType middleware.RateLimitType, limit int, window time.Duration, action string) middleware.RateLimitConfig {
	return middleware.RateLimitConfig{
		Name:    name,
		Type:    limitType,
		Limit:   limit,
		Window:  window,
		Message: rateLimitMessage(action, limit, window),
	}
}
