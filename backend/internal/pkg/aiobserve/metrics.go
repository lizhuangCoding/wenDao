package aiobserve

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"wenDao/config"
)

type TokenEstimate struct {
	PromptTokens     int64
	CompletionTokens int64
}

type CostEstimate struct {
	PromptTokens     int64
	CompletionTokens int64
	EstimatedCost    float64
	Currency         string
	Status           string
}

func EstimateTokens(prompt string, completion string) TokenEstimate {
	return TokenEstimate{
		PromptTokens:     estimateTextTokens(prompt),
		CompletionTokens: estimateTextTokens(completion),
	}
}

func EstimateCost(cfg config.AIConfig, prompt string, completion string) CostEstimate {
	tokens := EstimateTokens(prompt, completion)
	currency := strings.TrimSpace(cfg.CostCurrency)
	if currency == "" {
		currency = "USD"
	}
	status := "estimated"
	if cfg.PromptPricePer1K <= 0 && cfg.CompletionPricePer1K <= 0 {
		status = "tokens_only"
	}
	cost := float64(tokens.PromptTokens)/1000*cfg.PromptPricePer1K +
		float64(tokens.CompletionTokens)/1000*cfg.CompletionPricePer1K
	return CostEstimate{
		PromptTokens:     tokens.PromptTokens,
		CompletionTokens: tokens.CompletionTokens,
		EstimatedCost:    cost,
		Currency:         currency,
		Status:           status,
	}
}

func estimateTextTokens(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	runes := utf8.RuneCountInString(value)
	words := len(strings.Fields(value))
	estimate := (runes + 3) / 4
	if words > estimate {
		estimate = words
	}
	if estimate <= 0 {
		return 1
	}
	return int64(estimate)
}

func ClassifyFailure(message string) string {
	normalized := strings.ToLower(strings.TrimSpace(message))
	switch {
	case normalized == "":
		return ""
	case strings.Contains(normalized, "timeout") || strings.Contains(normalized, "deadline exceeded") || strings.Contains(normalized, "超时"):
		return "timeout"
	case strings.Contains(normalized, "rate limit") || strings.Contains(normalized, "too many requests") || strings.Contains(normalized, "429") || strings.Contains(normalized, "限流"):
		return "rate_limit"
	case strings.Contains(normalized, "unauthorized") || strings.Contains(normalized, "forbidden") || strings.Contains(normalized, "401") || strings.Contains(normalized, "403") || strings.Contains(normalized, "api key") || strings.Contains(normalized, "鉴权"):
		return "auth"
	case strings.Contains(normalized, "connection") || strings.Contains(normalized, "network") || strings.Contains(normalized, "dns") || strings.Contains(normalized, "no such host") || strings.Contains(normalized, "联网"):
		return "network"
	case strings.Contains(normalized, "tool") || strings.Contains(normalized, "localsearch") || strings.Contains(normalized, "websearch") || strings.Contains(normalized, "webfetch") || strings.Contains(normalized, "工具"):
		return "tool"
	case strings.Contains(normalized, "model") || strings.Contains(normalized, "llm") || strings.Contains(normalized, "provider") || strings.Contains(normalized, "generation") || strings.Contains(normalized, "模型"):
		return "provider"
	default:
		return "unknown"
	}
}

func FailureFingerprint(category string, message string) string {
	normalized := strings.ToLower(strings.TrimSpace(message))
	normalized = regexp.MustCompile(`\d+`).ReplaceAllString(normalized, "#")
	if len([]rune(normalized)) > 180 {
		normalized = string([]rune(normalized)[:180])
	}
	sum := sha1.Sum([]byte(strings.TrimSpace(category) + ":" + normalized))
	return hex.EncodeToString(sum[:])[:16]
}

func ScoreSourceQuality(urls []string, localHits int, webHits int) int {
	score := 0
	if localHits > 0 {
		score += 45
	}
	if webHits > 0 {
		score += 10
	}
	seen := map[string]struct{}{}
	for _, rawURL := range urls {
		u, err := url.Parse(rawURL)
		if err != nil || u.Hostname() == "" {
			continue
		}
		host := strings.ToLower(u.Hostname())
		if _, exists := seen[host]; exists {
			continue
		}
		seen[host] = struct{}{}
		switch {
		case strings.HasSuffix(host, ".edu") || strings.HasSuffix(host, ".gov"):
			score += 18
		case strings.Contains(host, "wikipedia.org") || strings.Contains(host, "github.com") || strings.Contains(host, "docs.") || strings.Contains(host, "developer."):
			score += 14
		case strings.HasPrefix(host, "www."):
			score += 8
		default:
			score += 6
		}
	}
	if score > 100 {
		return 100
	}
	return score
}

func RedactLogDetail(stage string, detail string) string {
	if strings.TrimSpace(detail) == "" {
		return ""
	}
	if isSensitiveStage(stage) {
		return fmt.Sprintf("[redacted:%d chars]", len([]rune(detail)))
	}
	return truncate(detail, 1200)
}

func RedactLogMetadata(stage string, metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return metadata
	}
	redacted := make(map[string]any, len(metadata))
	for key, value := range metadata {
		normalizedKey := strings.ToLower(key)
		if isSensitiveStage(stage) || normalizedKey == "content" || normalizedKey == "output" || normalizedKey == "answer" || strings.Contains(normalizedKey, "html") {
			redacted[key] = summarizeValue(value)
			continue
		}
		redacted[key] = value
	}
	return redacted
}

func isSensitiveStage(stage string) bool {
	normalized := strings.ToLower(stage)
	return strings.Contains(normalized, "web_fetch_result") ||
		strings.Contains(normalized, "completed") ||
		strings.Contains(normalized, "final") ||
		strings.Contains(normalized, "answer")
}

func summarizeValue(value any) string {
	switch v := value.(type) {
	case string:
		urls := ExtractHTTPURLs(v)
		if len(urls) > 0 {
			return fmt.Sprintf("[redacted:%d chars urls=%v]", len([]rune(v)), urls)
		}
		return fmt.Sprintf("[redacted:%d chars]", len([]rune(v)))
	default:
		return fmt.Sprintf("[redacted:%T]", value)
	}
}

func ExtractHTTPURLs(text string) []string {
	fields := strings.Fields(text)
	urls := make([]string, 0)
	seen := map[string]struct{}{}
	for _, field := range fields {
		candidate := strings.Trim(field, `"'(),.，。；;[]{}<>`)
		if !(strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://")) {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		urls = append(urls, candidate)
	}
	return urls
}

func truncate(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "..."
}
