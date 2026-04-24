package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"wenDao/config"
	"wenDao/internal/model"
	"wenDao/internal/pkg/hash"
)

// 初始化作者信息脚本

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

	// 设置管理员信息
	username := "admin"
	email := "admin@example.com"
	password := "admin123" // 请稍后自行修改

	// 加密密码
	passwordHash, err := hash.HashPassword(password)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	// 创建管理员
	admin := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: &passwordHash,
		Role:         "admin",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := db.Create(admin).Error; err != nil {
		log.Fatal("Failed to create admin:", err)
	}

	fmt.Printf("Admin account created successfully!\nUsername: %s\nEmail: %s\nPassword: %s\n", username, email, password)
}
