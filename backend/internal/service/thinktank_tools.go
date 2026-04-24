package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

type localSearchInput struct {
	Query string `json:"query"`
}

type webSearchInput struct {
	Query string `json:"query"`
}

type webFetchInput struct {
	URL string `json:"url"`
}

type userIDKey struct{}
type aiLoggerKey struct{}
type conversationIDKey struct{}
type runIDKey struct{}

func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

func getUserID(ctx context.Context) int64 {
	if v, ok := ctx.Value(userIDKey{}).(int64); ok {
		return v
	}
	return 0
}

func WithAILogger(ctx context.Context, logger AILogger) context.Context {
	return context.WithValue(ctx, aiLoggerKey{}, logger)
}

func getAILogger(ctx context.Context) AILogger {
	if v, ok := ctx.Value(aiLoggerKey{}).(AILogger); ok {
		return v
	}
	return nil
}

func WithConversationID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, conversationIDKey{}, id)
}

func getConversationID(ctx context.Context) int64 {
	if v, ok := ctx.Value(conversationIDKey{}).(int64); ok {
		return v
	}
	return 0
}

func WithRunID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, runIDKey{}, id)
}

func getRunID(ctx context.Context) int64 {
	if v, ok := ctx.Value(runIDKey{}).(int64); ok {
		return v
	}
	return 0
}

func logToolStage(ctx context.Context, stage, message string, metadata map[string]any) {
	logger := getAILogger(ctx)
	if logger == nil {
		return
	}
	logger.LogStage(AILogEntry{
		ConversationID: getConversationID(ctx),
		RunID:          getRunID(ctx),
		UserID:         getUserID(ctx),
		Stage:          stage,
		Message:        message,
		Metadata:       metadata,
	})
}

func newLocalSearchTool(librarian Librarian) (tool.BaseTool, error) {
	return toolutils.NewTool(
		&schema.ToolInfo{
			Name: "LocalSearch",
			Desc: "检索站内知识库并返回本地资料总结与文章引用",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"query": {Type: schema.String, Required: true},
			}),
		},
		func(ctx context.Context, input localSearchInput) (string, error) {
			if librarian == nil {
				return "本地知识库不可用", nil
			}
			logToolStage(ctx, "tool_local_search_call", "执行 LocalSearch 工具", map[string]any{"input": input.Query})
			result, err := librarian.Search(ctx, input.Query)
			if err != nil {
				logToolStage(ctx, "tool_local_search_error", "本地检索报错", map[string]any{"error": err.Error()})
				return fmt.Sprintf("本地检索发生错误: %v", err), nil // 不返回 error，防止 adk 崩溃
			}
			payload := map[string]any{
				"coverage_status": result.CoverageStatus,
				"summary":         result.Summary,
				"sources":         result.Sources,
			}
			logToolStage(ctx, "tool_local_search_result", "本地检索结果详情", payload)
			bytes, err := json.Marshal(payload)
			if err != nil {
				return "结果解析失败", nil
			}
			return string(bytes), nil
		},
	), nil
}

func newWebSearchTool(cfg ResearchConfig) (tool.BaseTool, error) {
	return toolutils.NewTool(
		&schema.ToolInfo{
			Name: "WebSearch",
			Desc: "调用配置化搜索服务，执行联网搜索并返回摘要结果",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"query": {Type: schema.String, Required: true},
			}),
		},
		func(ctx context.Context, input webSearchInput) (string, error) {
			logToolStage(ctx, "tool_web_search_call", "执行 WebSearch 工具", map[string]any{"input": input.Query})
			res, err := callResearchService(ctx, cfg, input.Query)
			if err != nil {
				logToolStage(ctx, "tool_web_search_error", "联网搜索报错", map[string]any{"error": err.Error()})
				return fmt.Sprintf("联网搜索暂不可用: %v", err), nil // 返回描述，不中断流程
			}
			logToolStage(ctx, "tool_web_search_result", "联网搜索结果详情", map[string]any{"output": res})
			return res, nil
		},
	), nil
}

func newWebFetchTool(cfg ResearchConfig) (tool.BaseTool, error) {
	return toolutils.NewTool(
		&schema.ToolInfo{
			Name: "WebFetch",
			Desc: "抓取单个外部 URL 的文本内容",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"url": {Type: schema.String, Required: true},
			}),
		},
		func(ctx context.Context, input webFetchInput) (string, error) {
			input.URL = strings.TrimSpace(input.URL)
			logToolStage(ctx, "tool_web_fetch_call", "执行 WebFetch 工具", map[string]any{"url": input.URL})

			parsed, err := neturl.ParseRequestURI(input.URL)
			if err != nil || parsed == nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				msg := "WebFetch 需要有效的 http(s) URL；当前输入不是 URL。请改用 WebSearch 或 LocalSearch 的结果摘要，或提供有效 URL。"
				logToolStage(ctx, "tool_web_fetch_error", "网页抓取参数不是有效 URL", map[string]any{"url": input.URL})
				return msg, nil
			}

			// 增加更加鲁棒的 Http Client
			client := &http.Client{
				Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, input.URL, nil)
			if err != nil {
				return fmt.Sprintf("无法构建抓取请求: %v", err), nil
			}

			// 模拟浏览器 Header，减少被拦截几率
			req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

			resp, err := client.Do(req)
			if err != nil {
				logToolStage(ctx, "tool_web_fetch_error", "网页抓取网络报错", map[string]any{"url": input.URL, "error": err.Error()})
				return fmt.Sprintf("网页抓取失败 (网络原因): %v", err), nil // 重点：返回错误描述字符串，不返回 error 对象
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Sprintf("网页抓取失败: 网站返回状态码 %d", resp.StatusCode), nil
			}

			body, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024)) // 增加限制到 32KB
			if err != nil {
				return fmt.Sprintf("网页内容读取失败: %v", err), nil
			}

			res := strings.TrimSpace(string(body))
			logToolStage(ctx, "tool_web_fetch_result", "网页抓取结果详情", map[string]any{
				"url":     input.URL,
				"content": res,
			})
			return res, nil
		},
	), nil
}

type docWriterInput struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Content string `json:"content"`
	Sources []any  `json:"sources"`
}

func newDocWriterTool(svc KnowledgeDocumentService) (tool.BaseTool, error) {
	return toolutils.NewTool(
		&schema.ToolInfo{
			Name: "DocWriter",
			Desc: "将调研结果保存为 Markdown 格式的知识文档草稿，供后续审核与入库",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"title":   {Type: schema.String, Required: true, Desc: "调研文档的标题"},
				"summary": {Type: schema.String, Required: true, Desc: "调研文档的简要总结"},
				"content": {Type: schema.String, Required: true, Desc: "调研文档的详细 Markdown 正文"},
				"sources": {
					Type:     schema.Array,
					Required: false,
					Desc:     "来源列表，可以是 URL 字符串数组，也可以是包含 url 和 title 的对象数组",
				},
			}),
		},
		func(ctx context.Context, input docWriterInput) (string, error) {
			if svc == nil {
				return "知识文档服务不可用", nil
			}
			userID := getUserID(ctx)

			logToolStage(ctx, "tool_doc_writer_call", "开始执行 DocWriter", map[string]any{
				"title":   input.Title,
				"summary": input.Summary,
				"sources": input.Sources,
			})

			finalSources := make([]KnowledgeSourceInput, 0, len(input.Sources))
			for _, s := range input.Sources {
				switch v := s.(type) {
				case string:
					finalSources = append(finalSources, KnowledgeSourceInput{URL: v, Title: "外部来源"})
				case map[string]any:
					ks := KnowledgeSourceInput{}
					if url, ok := v["url"].(string); ok {
						ks.URL = url
					}
					if title, ok := v["title"].(string); ok {
						ks.Title = title
					} else {
						ks.Title = ks.URL
					}
					if domain, ok := v["domain"].(string); ok {
						ks.Domain = domain
					}
					if snippet, ok := v["snippet"].(string); ok {
						ks.Snippet = snippet
					}
					finalSources = append(finalSources, ks)
				}
			}

			doc, err := svc.CreateResearchDraft(CreateKnowledgeDocumentInput{
				Title:           input.Title,
				Summary:         input.Summary,
				Content:         input.Content,
				CreatedByUserID: userID,
				Sources:         finalSources,
			})
			if err != nil {
				logToolStage(ctx, "tool_doc_writer_error", "调研文档存盘失败", map[string]any{"error": err.Error()})
				return fmt.Sprintf("调研文档存盘失败: %v", err), nil
			}
			res := fmt.Sprintf("成功创建知识文档草稿：ID=%d, Title=%s", doc.ID, doc.Title)
			logToolStage(ctx, "tool_doc_writer_result", "调研文档存盘成功", map[string]any{"doc_id": doc.ID})
			return res, nil
		},
	), nil
}

type ResearchConfig struct {
	Endpoint       string
	APIKey         string
	MaxResults     int
	TimeoutSeconds int
}

func callResearchService(ctx context.Context, cfg ResearchConfig, query string) (string, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return "未配置 research_endpoint，无法执行联网搜索", nil
	}

	var payload []byte
	var err error

	if strings.Contains(cfg.Endpoint, "serper.dev") {
		payload, err = json.Marshal(map[string]any{
			"q": query,
		})
	} else {
		payload, err = json.Marshal(map[string]any{
			"query":       query,
			"max_results": cfg.MaxResults,
		})
	}
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("X-API-KEY", cfg.APIKey)
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return string(body), fmt.Errorf("research service returned status %d", resp.StatusCode)
	}

	return strings.TrimSpace(string(body)), nil
}
