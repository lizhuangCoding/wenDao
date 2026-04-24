package site

import (
	"github.com/gin-gonic/gin"

	"wenDao/config"
	"wenDao/internal/pkg/response"
)

// SiteHandler 网站配置处理器
type SiteHandler struct {
	cfg *config.Config
}

// NewSiteHandler 创建网站配置处理器
func NewSiteHandler(cfg *config.Config) *SiteHandler {
	return &SiteHandler{
		cfg: cfg,
	}
}

// GetSlogan 获取网站标语
func (h *SiteHandler) GetSlogan(c *gin.Context) {
	response.Success(c, gin.H{
		"slogan": h.cfg.Site.Slogan,
	})
}