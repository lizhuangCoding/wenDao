package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
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

	admins, err := parseAdminSpecs(os.Environ())
	if err != nil {
		log.Fatal("Failed to parse admin configuration:", err)
	}

	for _, admin := range admins {
		if err := upsertAdmin(db, admin); err != nil {
			log.Fatalf("Failed to initialize admin %s: %v", admin.Email, err)
		}
		fmt.Printf("Admin account initialized successfully: username=%s email=%s\n", admin.Username, admin.Email)
	}
}

func parseAdminSpecs(env []string) ([]adminSpec, error) {
	values := envValues(env)

	var specs []adminSpec
	if values["ADMIN_EMAIL"] != "" || values["ADMIN_PASSWORD"] != "" || values["ADMIN_USERNAME"] != "" {
		specs = append(specs, adminSpec{
			Username: values["ADMIN_USERNAME"],
			Email:    values["ADMIN_EMAIL"],
			Password: values["ADMIN_PASSWORD"],
		})
	}

	indexes := adminIndexes(values)
	for _, index := range indexes {
		specs = append(specs, adminSpec{
			Username: values[fmt.Sprintf("ADMIN_USERNAME_%d", index)],
			Email:    values[fmt.Sprintf("ADMIN_EMAIL_%d", index)],
			Password: values[fmt.Sprintf("ADMIN_PASSWORD_%d", index)],
		})
	}

	if len(specs) == 0 {
		return nil, errors.New("set ADMIN_EMAIL and ADMIN_PASSWORD, or numbered ADMIN_EMAIL_1 and ADMIN_PASSWORD_1 variables")
	}

	for i := range specs {
		if err := normalizeAdminSpec(&specs[i]); err != nil {
			return nil, err
		}
	}

	if err := rejectDuplicateAdmins(specs); err != nil {
		return nil, err
	}

	return specs, nil
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

func adminIndexes(values map[string]string) []int {
	seen := make(map[int]struct{})
	for key := range values {
		for _, prefix := range []string{"ADMIN_EMAIL_", "ADMIN_PASSWORD_", "ADMIN_USERNAME_"} {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			index, err := strconv.Atoi(strings.TrimPrefix(key, prefix))
			if err == nil && index > 0 {
				seen[index] = struct{}{}
			}
		}
	}

	indexes := make([]int, 0, len(seen))
	for index := range seen {
		if !hasNumberedAdminValue(values, index) {
			continue
		}
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	return indexes
}

func hasNumberedAdminValue(values map[string]string, index int) bool {
	for _, key := range []string{
		fmt.Sprintf("ADMIN_EMAIL_%d", index),
		fmt.Sprintf("ADMIN_PASSWORD_%d", index),
		fmt.Sprintf("ADMIN_USERNAME_%d", index),
	} {
		if strings.TrimSpace(values[key]) != "" {
			return true
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

func rejectDuplicateAdmins(specs []adminSpec) error {
	emails := make(map[string]struct{}, len(specs))
	usernames := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		email := strings.ToLower(spec.Email)
		username := strings.ToLower(spec.Username)
		if _, ok := emails[email]; ok {
			return fmt.Errorf("duplicate admin email %q", spec.Email)
		}
		if _, ok := usernames[username]; ok {
			return fmt.Errorf("duplicate admin username %q", spec.Username)
		}
		emails[email] = struct{}{}
		usernames[username] = struct{}{}
	}
	return nil
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
