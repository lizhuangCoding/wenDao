package chat

import (
	"context"
	"encoding/json"
	"html"
	neturl "net/url"
	"regexp"
	"strings"
	"sync"
)

const maxWebFetchCandidatesPerCall = 4

type webFetchStateKey struct{}

type webFetchCandidate struct {
	Title   string
	URL     string
	Snippet string
}

type webFetchState struct {
	mu         sync.Mutex
	candidates []webFetchCandidate
	attempted  map[string]struct{}
}

func newWebFetchState() *webFetchState {
	return &webFetchState{attempted: make(map[string]struct{})}
}

func WithWebFetchState(ctx context.Context, state *webFetchState) context.Context {
	if state == nil {
		state = newWebFetchState()
	}
	return context.WithValue(ctx, webFetchStateKey{}, state)
}

func getWebFetchState(ctx context.Context) *webFetchState {
	if state, ok := ctx.Value(webFetchStateKey{}).(*webFetchState); ok {
		return state
	}
	return nil
}

func recordWebSearchCandidates(ctx context.Context, payload string) {
	state := getWebFetchState(ctx)
	if state == nil {
		return
	}
	state.add(extractWebFetchCandidates(payload))
}

func extractWebFetchCandidates(payload string) []webFetchCandidate {
	var data struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic"`
		Items []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Snippet string `json:"snippet"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return nil
	}

	candidates := make([]webFetchCandidate, 0, len(data.Organic)+len(data.Items))
	for _, item := range data.Organic {
		candidates = appendValidWebFetchCandidate(candidates, webFetchCandidate{
			Title:   item.Title,
			URL:     item.Link,
			Snippet: item.Snippet,
		})
	}
	for _, item := range data.Items {
		candidates = appendValidWebFetchCandidate(candidates, webFetchCandidate{
			Title:   item.Title,
			URL:     item.URL,
			Snippet: item.Snippet,
		})
	}
	return candidates
}

func appendValidWebFetchCandidate(candidates []webFetchCandidate, candidate webFetchCandidate) []webFetchCandidate {
	candidate.URL = strings.TrimSpace(candidate.URL)
	if !isValidHTTPURL(candidate.URL) {
		return candidates
	}
	candidate.Title = strings.TrimSpace(candidate.Title)
	candidate.Snippet = strings.TrimSpace(candidate.Snippet)
	return append(candidates, candidate)
}

func isValidHTTPURL(rawURL string) bool {
	parsed, err := neturl.ParseRequestURI(strings.TrimSpace(rawURL))
	return err == nil && parsed != nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func (s *webFetchState) add(candidates []webFetchCandidate) {
	if s == nil || len(candidates) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[string]struct{}, len(s.candidates))
	for _, candidate := range s.candidates {
		seen[candidate.URL] = struct{}{}
	}
	for _, candidate := range candidates {
		if _, ok := seen[candidate.URL]; ok {
			continue
		}
		s.candidates = append(s.candidates, candidate)
		seen[candidate.URL] = struct{}{}
	}
}

func selectWebFetchCandidates(ctx context.Context, requested []webFetchCandidate, limit int) []webFetchCandidate {
	if limit <= 0 {
		limit = maxWebFetchCandidatesPerCall
	}
	state := getWebFetchState(ctx)
	if state == nil {
		return uniqueValidWebFetchCandidates(requested, limit)
	}
	return state.selectCandidates(requested, limit)
}

func uniqueValidWebFetchCandidates(requested []webFetchCandidate, limit int) []webFetchCandidate {
	selected := make([]webFetchCandidate, 0, limit)
	seen := make(map[string]struct{}, len(requested))
	for _, candidate := range requested {
		candidate.URL = strings.TrimSpace(candidate.URL)
		if !isValidHTTPURL(candidate.URL) {
			continue
		}
		if _, ok := seen[candidate.URL]; ok {
			continue
		}
		selected = append(selected, candidate)
		seen[candidate.URL] = struct{}{}
		if len(selected) >= limit {
			break
		}
	}
	return selected
}

func (s *webFetchState) selectCandidates(requested []webFetchCandidate, limit int) []webFetchCandidate {
	s.mu.Lock()
	defer s.mu.Unlock()

	selected := make([]webFetchCandidate, 0, limit)
	seen := make(map[string]struct{}, limit)
	add := func(candidate webFetchCandidate) {
		if len(selected) >= limit {
			return
		}
		candidate.URL = strings.TrimSpace(candidate.URL)
		if !isValidHTTPURL(candidate.URL) {
			return
		}
		if _, ok := seen[candidate.URL]; ok {
			return
		}
		if _, ok := s.attempted[candidate.URL]; ok {
			return
		}
		selected = append(selected, candidate)
		seen[candidate.URL] = struct{}{}
		s.attempted[candidate.URL] = struct{}{}
	}

	for _, candidate := range requested {
		add(candidate)
	}
	for _, candidate := range s.candidates {
		add(candidate)
	}
	return selected
}

var (
	webScriptPattern = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>|<style[^>]*>.*?</style>|<noscript[^>]*>.*?</noscript>|<svg[^>]*>.*?</svg>`)
	webTagPattern    = regexp.MustCompile(`(?s)<[^>]+>`)
	webSpacePattern  = regexp.MustCompile(`\s+`)
)

func readableWebContent(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	if looksLikeRawHTML(text) {
		text = webScriptPattern.ReplaceAllString(text, " ")
		text = webTagPattern.ReplaceAllString(text, " ")
		text = html.UnescapeString(text)
		text = webSpacePattern.ReplaceAllString(text, " ")
		text = strings.TrimSpace(text)
	}
	if len([]rune(text)) > 8000 {
		text = string([]rune(text)[:8000]) + "..."
	}
	return text
}
