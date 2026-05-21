package httpcookie

import (
	"net/url"
	"strings"

	"wenDao/config"
)

func ShouldUseSecureCookies(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}

	siteURL := strings.TrimSpace(cfg.Site.URL)
	if siteURL == "" {
		return cfg.Server.Mode == "release"
	}

	parsed, err := url.Parse(siteURL)
	if err != nil {
		return cfg.Server.Mode == "release"
	}
	return strings.EqualFold(parsed.Scheme, "https")
}
