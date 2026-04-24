package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword 使用 bcrypt 加密密码
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateSlug 根据 ID 生成 slug（SHA256 前 10 位）
func GenerateSlug(id int64) string {
	hash := sha256.Sum256([]byte(strconv.FormatInt(id, 10)))
	return hex.EncodeToString(hash[:])[:10]
}
