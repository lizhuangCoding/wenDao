package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"wenDao/config"
	"wenDao/internal/model"
	"wenDao/internal/pkg/hash"
)

type adminSpec struct {
	Username string
	Email    string
	Password string
}

func main() {
	if gin.Mode() != gin.ReleaseMode {
		if err := godotenv.Load("config/.env"); err != nil {
			log.Println("Warning: .env file not found, using system environment variables")
		}
	}

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 连接数据库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	admin, err := parseAdminSpec(os.Environ())
	if err != nil {
		log.Fatal("Failed to parse admin configuration:", err)
	}

	if err := upsertAdmin(db, admin); err != nil {
		log.Fatalf("Failed to initialize admin %s: %v", admin.Email, err)
	}
	fmt.Printf("Admin account initialized successfully: username=%s email=%s\n", admin.Username, admin.Email)
}

func parseAdminSpec(env []string) (adminSpec, error) {
	values := envValues(env)
	if hasNumberedAdminVariables(values) {
		return adminSpec{}, errors.New("numbered admin variables are no longer supported; use ADMIN_EMAIL, ADMIN_USERNAME, and ADMIN_PASSWORD")
	}

	spec := adminSpec{
		Username: values["ADMIN_USERNAME"],
		Email:    values["ADMIN_EMAIL"],
		Password: values["ADMIN_PASSWORD"],
	}
	if spec.Email == "" && spec.Password == "" && spec.Username == "" {
		return adminSpec{}, errors.New("set ADMIN_EMAIL and ADMIN_PASSWORD")
	}
	if err := normalizeAdminSpec(&spec); err != nil {
		return adminSpec{}, err
	}
	return spec, nil
}

func hasNumberedAdminVariables(values map[string]string) bool {
	for key := range values {
		for _, prefix := range []string{"ADMIN_EMAIL_", "ADMIN_PASSWORD_", "ADMIN_USERNAME_"} {
			if strings.HasPrefix(key, prefix) {
				return true
			}
		}
	}
	return false
}

func normalizeAdminSpec(spec *adminSpec) error {
	spec.Email = strings.TrimSpace(spec.Email)
	spec.Username = strings.TrimSpace(spec.Username)
	spec.Password = strings.TrimSpace(spec.Password)

	if spec.Email == "" {
		return errors.New("admin email is required")
	}
	if !strings.Contains(spec.Email, "@") {
		return fmt.Errorf("admin email %q is invalid", spec.Email)
	}
	if spec.Password == "" {
		return fmt.Errorf("admin password is required for %s", spec.Email)
	}
	if len(spec.Password) < 6 {
		return fmt.Errorf("admin password for %s must be at least 6 characters", spec.Email)
	}
	if spec.Username == "" {
		spec.Username = usernameFromEmail(spec.Email)
	}
	if len(spec.Username) < 2 || len(spec.Username) > 50 {
		return fmt.Errorf("admin username for %s must be 2-50 characters", spec.Email)
	}

	return nil
}

func usernameFromEmail(email string) string {
	localPart, _, ok := strings.Cut(email, "@")
	if !ok {
		return "admin"
	}
	localPart = strings.TrimSpace(localPart)
	if len(localPart) < 2 {
		return "admin"
	}
	if len(localPart) > 50 {
		return localPart[:50]
	}
	return localPart
}

func envValues(env []string) map[string]string {
	values := make(map[string]string, len(env))
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		values[key] = strings.TrimSpace(value)
	}
	return values
}

func upsertAdmin(db *gorm.DB, spec adminSpec) error {
	passwordHash, err := hash.HashPassword(spec.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	defaultAvatar := fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s", spec.Username)

	var existing model.User
	err = db.Where("email = ?", spec.Email).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		admin := &model.User{
			Username:     spec.Username,
			Email:        spec.Email,
			PasswordHash: &passwordHash,
			Role:         "admin",
			AvatarURL:    &defaultAvatar,
			AvatarSource: model.AvatarSourceDefault,
			Status:       "active",
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := db.Create(admin).Error; err != nil {
			return fmt.Errorf("create admin: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("find admin by email: %w", err)
	}

	updates := map[string]interface{}{
		"username":      spec.Username,
		"password_hash": passwordHash,
		"role":          "admin",
		"status":        "active",
		"updated_at":    now,
	}
	if existing.AvatarURL == nil || strings.TrimSpace(*existing.AvatarURL) == "" {
		updates["avatar_url"] = defaultAvatar
		updates["avatar_source"] = model.AvatarSourceDefault
	}

	if err := db.Model(&existing).Updates(updates).Error; err != nil {
		return fmt.Errorf("update admin: %w", err)
	}
	return nil
}
