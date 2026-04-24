# 修复 User/Admin 权限模型与编译错误计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标:** 修复 `backend/internal/service/user.go` 中的编译错误，并使 `Login` 逻辑能正确识别管理员。

**架构:** 
- `User` 模型中不再有 `Role` 字段，默认为 "user"。
- `Admin` 模型存储在 `admins` 表中。
- `Login` 逻辑将同时支持管理员和普通用户。

**技术栈:** Go, GORM, JWT.

---

### 任务 1: 创建 Admin Repository

**文件:**
- 创建: `backend/internal/repository/admin.go`

- [ ] **步骤 1: 实现 `AdminRepository` 接口及其实现**

```go
package repository

import (
	"gorm.io/gorm"
	"wenDao/internal/model"
)

type AdminRepository interface {
	GetByEmail(email string) (*model.Admin, error)
	GetByID(id int64) (*model.Admin, error)
}

type adminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) GetByEmail(email string) (*model.Admin, error) {
	var admin model.Admin
	err := r.db.Where("email = ?", email).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

func (r *adminRepository) GetByID(id int64) (*model.Admin, error) {
	var admin model.Admin
	err := r.db.Where("id = ?", id).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}
```

- [ ] **步骤 2: 提交代码**

```bash
git add backend/internal/repository/admin.go
git commit -m "feat: add AdminRepository"
```

### 任务 2: 更新 UserService 依赖与修复编译错误

**文件:**
- 修改: `backend/internal/service/user.go`

- [ ] **步骤 1: 注入 `AdminRepository` 并修复 `Register` 和 `Login` 逻辑**

```go
// 修改 userService 结构体
type userService struct {
	userRepo     repository.UserRepository
	adminRepo    repository.AdminRepository // 新增
	oauthService OAuthService
	cfg          *config.Config
	rdb          *redis.Client
}

// 修改 NewUserService 函数
func NewUserService(
	userRepo repository.UserRepository,
	adminRepo repository.AdminRepository, // 新增
	oauthService OAuthService,
	cfg *config.Config,
	rdb *redis.Client,
) UserService {
	return &userService{
		userRepo:     userRepo,
		adminRepo:    adminRepo,
		oauthService: oauthService,
		cfg:          cfg,
		rdb:          rdb,
	}
}

// 修复 Login 逻辑：先找 Admin，再找 User
func (s *userService) Login(email, password string) (string, *model.User, error) {
	// 1. 先尝试从 Admin 表查询
	admin, err := s.adminRepo.GetByEmail(email)
	if err == nil && admin != nil {
		// 验证管理员密码
		if !hash.CheckPassword(password, admin.PasswordHash) {
			return "", nil, errors.New("invalid email or password")
		}
		// 生成管理员 Token (Role="admin")
		token, err := pkgjwt.GenerateToken(admin.ID, "admin", s.cfg.JWT.Secret, s.cfg.JWT.ExpireHours)
		if err != nil {
			return "", nil, err
		}
		// 返回一个由 Admin 信息构造的 User 模型对象，供 Handler 使用
		return token, &model.User{
			ID:       admin.ID,
			Username: admin.Username,
			Email:    admin.Email,
			// 注意：这里没有 Role 字段了，Role 信息通过 Token 传递
		}, nil
	}

	// 2. 尝试从 User 表查询
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("invalid email or password")
		}
		return "", nil, err
	}

	// 验证普通用户密码等逻辑...
	// 生成普通用户 Token (Role="user")
	token, err := pkgjwt.GenerateToken(user.ID, "user", s.cfg.JWT.Secret, s.cfg.JWT.ExpireHours)
	// ...
}
```

- [ ] **步骤 2: 修复 `GitHubOAuthLogin` 逻辑**
移除 `Role` 字段赋值。

- [ ] **步骤 3: 提交代码**

```bash
git add backend/internal/service/user.go
git commit -m "fix: resolve compilation errors and update login logic for admins"
```

### 任务 3: 更新 main.go 依赖注入

**文件:**
- 修改: `backend/cmd/server/main.go`

- [ ] **步骤 1: 初始化 `AdminRepository` 并传给 `UserService`**

```go
	// ... 
	adminRepo := repository.NewAdminRepository(db)
	// ...
	userService := service.NewUserService(userRepo, adminRepo, oauthService, cfg, rdb)
```

- [ ] **步骤 2: 提交代码**

```bash
git add backend/cmd/server/main.go
git commit -m "chore: update dependency injection for adminRepo"
```
