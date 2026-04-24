package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"wenDao/config"
)

// JournalistResult 联网调研结果
type JournalistResult struct {
	Summary               string
	Sources               []SourceRef
	KnowledgeDraftTitle   string
	KnowledgeDraftBody    string
	KnowledgeDraftSummary string
}

// Journalist 外部调研代理
type Journalist interface {
	Research(ctx context.Context, question string, local LibrarianResult) (*JournalistResult, error)
}

type httpJournalist struct {
	cfg    *config.AIConfig
	client *http.Client
}

// NewJournalist 创建联网调研代理
func NewJournalist(cfg *config.AIConfig) Journalist {
	timeout := 15 * time.Second
	if cfg != nil && cfg.ResearchTimeoutSeconds > 0 {
		timeout = time.Duration(cfg.ResearchTimeoutSeconds) * time.Second
	}
	return &httpJournalist{
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
	}
}

func (j *httpJournalist) Research(ctx context.Context, question string, local LibrarianResult) (*JournalistResult, error) {
	if j.cfg == nil || strings.TrimSpace(j.cfg.ResearchEndpoint) == "" {
		return &JournalistResult{
			Summary:               "当前未配置联网调研服务，以下结论仅基于已有信息补充整理。",
			KnowledgeDraftTitle:   buildResearchDraftTitle(question),
			KnowledgeDraftSummary: "当前未配置联网调研服务",
			KnowledgeDraftBody:    "当前未配置联网调研服务，未获取外部来源。",
		}, nil
	}

	maxResults := 5
	if j.cfg.ResearchMaxResults > 0 {
		maxResults = j.cfg.ResearchMaxResults
	}

	payload := map[string]any{
		"query":       question,
		"max_results": maxResults,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, j.cfg.ResearchEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if j.cfg.ResearchAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+j.cfg.ResearchAPIKey)
	}

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("research service returned status %d", resp.StatusCode)
	}

	var result struct {
		Summary string `json:"summary"`
		Items   []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Domain  string `json:"domain"`
			Snippet string `json:"snippet"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	jr := &JournalistResult{
		Summary:               result.Summary,
		KnowledgeDraftTitle:   buildResearchDraftTitle(question),
		KnowledgeDraftSummary: result.Summary,
		KnowledgeDraftBody:    result.Summary,
	}
	for _, item := range result.Items {
		jr.Sources = append(jr.Sources, SourceRef{Kind: "web", Title: item.Title, URL: item.URL})
	}
	return jr, nil
}

func buildResearchDraftTitle(question string) string {
	trimmed := strings.TrimSpace(question)
	if trimmed == "" {
		return "联网调研结果"
	}
	runes := []rune(trimmed)
	if len(runes) > 24 {
		trimmed = string(runes[:24])
	}
	return trimmed + " 调研结果"
}
