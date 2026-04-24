package user

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// UserRepository 用户数据访问接口
type UserRepository interface {
	Create(user *model.User) error
	GetByID(id int64) (*model.User, error)
	GetByEmail(email string) (*model.User, error)
	GetByUsername(username string) (*model.User, error)
	GetByOAuth(provider string, oauthID string) (*model.User, error)
	Update(user *model.User) error
}

// userRepository 用户数据访问实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户数据访问实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 创建用户
func (r *userRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// GetByID 根据 ID 查询用户
func (r *userRepository) GetByID(id int64) (*model.User, error) {
	var user model.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail 根据邮箱查询用户
func (r *userRepository) GetByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根据用户名查询用户
func (r *userRepository) GetByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByOAuth 根据 OAuth 信息查询用户
func (r *userRepository) GetByOAuth(provider string, oauthID string) (*model.User, error) {
	var user model.User
	err := r.db.Where("o_auth_provider = ? AND o_auth_id = ?", provider, oauthID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update 更新用户信息
func (r *userRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}
