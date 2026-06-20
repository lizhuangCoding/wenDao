package site

import (
	"encoding/xml"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"wenDao/config"
	"wenDao/internal/pkg/response"
	"wenDao/internal/service/article"
	"wenDao/internal/service/setting"
)

// SiteHandler 网站配置处理器
type SiteHandler struct {
	cfg            *config.Config
	articleService article.ArticleService
	settingService setting.SettingService
}

// NewSiteHandler 创建网站配置处理器
func NewSiteHandler(cfg *config.Config, articleService article.ArticleService, settingService setting.SettingService) *SiteHandler {
	return &SiteHandler{
		cfg:            cfg,
		articleService: articleService,
		settingService: settingService,
	}
}

// GetSlogan 获取网站标语
func (h *SiteHandler) GetSlogan(c *gin.Context) {
	slogan := h.cfg.Site.Slogan
	if h.settingService != nil {
		if dbSlogan := h.settingService.GetSlogan(); dbSlogan != "" {
			slogan = dbSlogan
		}
	}
	response.Success(c, gin.H{
		"slogan": slogan,
	})
}

// GetContactLinks 获取联系方式
func (h *SiteHandler) GetContactLinks(c *gin.Context) {
	links := h.defaultContactLinks()
	if h.settingService != nil {
		if storedLinks, ok := h.settingService.GetContactLinks(); ok {
			links = storedLinks
		}
	}

	response.Success(c, gin.H{
		"contact_links": links,
	})
}

// SetSlogan 设置网站标语（管理员）
func (h *SiteHandler) SetSlogan(c *gin.Context) {
	var req struct {
		Slogan string `json:"slogan" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "标语不能为空")
		return
	}
	if h.settingService == nil {
		response.InternalError(c, "Setting service unavailable")
		return
	}
	if err := h.settingService.SetSlogan(req.Slogan); err != nil {
		response.InternalErrorWithErr(c, "设置标语失败", err)
		return
	}
	response.Success(c, gin.H{"slogan": req.Slogan})
}

// SetContactLinks 设置联系方式（管理员）
func (h *SiteHandler) SetContactLinks(c *gin.Context) {
	var req struct {
		ContactLinks []setting.ContactLink `json:"contact_links"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "联系方式格式错误")
		return
	}
	if h.settingService == nil {
		response.InternalError(c, "Setting service unavailable")
		return
	}

	if err := h.settingService.SetContactLinks(req.ContactLinks); err != nil {
		response.InternalErrorWithErr(c, "设置联系方式失败", err)
		return
	}

	if savedLinks, ok := h.settingService.GetContactLinks(); ok {
		response.Success(c, gin.H{"contact_links": savedLinks})
		return
	}

	response.Success(c, gin.H{"contact_links": req.ContactLinks})
}

// RobotsTxt 返回 robots.txt 内容
func (h *SiteHandler) RobotsTxt(c *gin.Context) {
	sitemapURL := h.siteBaseURL(c) + "/sitemap.xml"
	content := strings.Join([]string{
		"User-agent: *",
		"Allow: /",
		"Disallow: /api/",
		"Disallow: /admin/",
		"Sitemap: " + sitemapURL,
		"",
	}, "\n")
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(200, content)
}

func (h *SiteHandler) siteBaseURL(c *gin.Context) string {
	if h != nil && h.cfg != nil {
		if siteURL := strings.TrimRight(strings.TrimSpace(h.cfg.Site.URL), "/"); siteURL != "" {
			return siteURL
		}
	}
	if c == nil || c.Request == nil {
		return ""
	}

	host := firstForwardedValue(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}
	if host == "" {
		return ""
	}

	scheme := firstForwardedValue(c.GetHeader("X-Forwarded-Proto"))
	if scheme == "" {
		scheme = "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
	}

	return scheme + "://" + host
}

func firstForwardedValue(value string) string {
	if value == "" {
		return ""
	}
	return strings.TrimSpace(strings.Split(value, ",")[0])
}

func (h *SiteHandler) defaultContactLinks() []setting.ContactLink {
	return []setting.ContactLink{
		{
			Type:      "email",
			Label:     "QQ 邮箱",
			Value:     "3174285493@qq.com",
			URL:       "mailto:3174285493@qq.com",
			Enabled:   true,
			SortOrder: 1,
		},
		{
			Type:      "github",
			Label:     "GitHub",
			Value:     "lizhuangCoding",
			URL:       "https://github.com/lizhuangCoding",
			Enabled:   true,
			SortOrder: 2,
		},
	}
}

// SitemapXml 返回 sitemap.xml 内容
func (h *SiteHandler) SitemapXml(c *gin.Context) {
	articles, err := h.articleService.GetAllPublished()
	if err != nil {
		response.InternalErrorWithErr(c, "生成站点地图失败，请稍后重试", err)
		return
	}

	type sitemapURL struct {
		XMLName    xml.Name `xml:"url"`
		Loc        string   `xml:"loc"`
		LastMod    string   `xml:"lastmod,omitempty"`
		ChangeFreq string   `xml:"changefreq"`
		Priority   string   `xml:"priority"`
	}

	type sitemapURLSet struct {
		XMLName xml.Name     `xml:"urlset"`
		Xmlns   string       `xml:"xmlns,attr"`
		URLs    []sitemapURL `xml:"url"`
	}

	baseURL := h.siteBaseURL(c)
	urls := []sitemapURL{
		{
			Loc:        baseURL + "/",
			ChangeFreq: "daily",
			Priority:   "0.9",
		},
	}

	for _, a := range articles {
		lastMod := a.UpdatedAt
		if a.PublishedAt != nil {
			lastMod = *a.PublishedAt
		}
		urls = append(urls, sitemapURL{
			Loc:        baseURL + "/article/" + a.Slug,
			LastMod:    lastMod.Format(time.RFC3339),
			ChangeFreq: "weekly",
			Priority:   "0.7",
		})
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.XML(200, sitemapURLSet{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: urls})
}
