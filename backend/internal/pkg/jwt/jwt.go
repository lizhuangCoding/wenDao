package jwt

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

// Claims JWT 载荷
type Claims struct {
	UserID    int64  `json:"user_id"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"` // access 或 refresh
	jwt.RegisteredClaims
}

// GenerateAccessToken 生成 Access Token（短期）
func GenerateAccessToken(userID int64, role string, secret string, expireHours int) (string, error) {
	claims := Claims{
		UserID:    userID,
		Role:      role,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken 生成 Refresh Token（长期）
func GenerateRefreshToken(userID int64, role string, secret string, expireDays int) (string, error) {
	claims := Claims{
		UserID:    userID,
		Role:      role,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireDays) * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateToken 生成 JWT token（兼容旧接口）
func GenerateToken(userID int64, role string, secret string, expireHours int) (string, error) {
	return GenerateAccessToken(userID, role, secret, expireHours)
}

// ParseToken 解析并验证 token
func ParseToken(tokenString string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ValidateRefreshToken 验证 Refresh Token
func ValidateRefreshToken(tokenString string, secret string) (*Claims, error) {
	claims, err := ParseToken(tokenString, secret)
	if err != nil {
		return nil, err
	}
	// 检查 token 类型
	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("invalid token type")
	}
	return claims, nil
}

// IsTokenBlacklisted 检查 token 是否在黑名单中（已登出）
func IsTokenBlacklisted(rdb *redis.Client, token string) (bool, error) {
	ctx := context.Background()
	key := fmt.Sprintf("auth:blacklist:%s", token)

	exists, err := rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return exists > 0, nil
}

// AddToBlacklist 将 token 加入黑名单（登出时调用）
func AddToBlacklist(rdb *redis.Client, token string, expireTime time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("auth:blacklist:%s", token)

	return rdb.Set(ctx, key, "1", expireTime).Err()
}
