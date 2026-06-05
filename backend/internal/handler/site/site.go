package site

import (
	"encoding/xml"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"wenDao/config"
	"wenDao/internal/pkg/response"
	"wenDao/internal/service/article"
)

// SiteHandler 网站配置处理器
type SiteHandler struct {
	cfg            *config.Config
	articleService article.ArticleService
}

// NewSiteHandler 创建网站配置处理器
func NewSiteHandler(cfg *config.Config, articleService article.ArticleService) *SiteHandler {
	return &SiteHandler{
		cfg:            cfg,
		articleService: articleService,
	}
}

// GetSlogan 获取网站标语
func (h *SiteHandler) GetSlogan(c *gin.Context) {
	response.Success(c, gin.H{
		"slogan": h.cfg.Site.Slogan,
	})
}

// RobotsTxt 返回 robots.txt 内容
func (h *SiteHandler) RobotsTxt(c *gin.Context) {
	sitemapURL := strings.TrimRight(h.cfg.Site.URL, "/") + "/sitemap.xml"
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

// SitemapXml 返回 sitemap.xml 内容
func (h *SiteHandler) SitemapXml(c *gin.Context) {
	articles, err := h.articleService.GetAllPublished()
	if err != nil {
		response.InternalErrorWithErr(c, "Failed to generate sitemap", err)
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

	baseURL := strings.TrimRight(h.cfg.Site.URL, "/")
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
