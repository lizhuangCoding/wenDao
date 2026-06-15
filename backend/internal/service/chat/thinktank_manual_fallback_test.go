package chat

import (
	"errors"
	"strings"
	"testing"
)

func TestShouldRunJournalist(t *testing.T) {
	tests := []struct {
		name     string
		local    LibrarianResult
		localErr error
		want     bool
	}{
		{
			name:     "sufficient coverage skips journalist",
			local:    LibrarianResult{CoverageStatus: "sufficient"},
			localErr: nil,
			want:     false,
		},
		{
			name:     "insufficient coverage runs journalist",
			local:    LibrarianResult{CoverageStatus: "insufficient"},
			localErr: nil,
			want:     true,
		},
		{
			name:     "empty coverage status runs journalist",
			local:    LibrarianResult{CoverageStatus: ""},
			localErr: nil,
			want:     true,
		},
		{
			name:     "local error runs journalist",
			local:    LibrarianResult{CoverageStatus: "sufficient"},
			localErr: errors.New("connection refused"),
			want:     true,
		},
		{
			name:     "whitespace-only status runs journalist",
			local:    LibrarianResult{CoverageStatus: "   "},
			localErr: nil,
			want:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRunJournalist(tt.local, tt.localErr)
			if got != tt.want {
				t.Errorf("shouldRunJournalist() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasUsableLocalResult(t *testing.T) {
	tests := []struct {
		name   string
		result LibrarianResult
		want   bool
	}{
		{name: "has summary", result: LibrarianResult{Summary: "useful content"}, want: true},
		{name: "has sources", result: LibrarianResult{Sources: []SourceRef{{Title: "doc"}}}, want: true},
		{name: "both present", result: LibrarianResult{Summary: "x", Sources: []SourceRef{{}}}, want: true},
		{name: "empty", result: LibrarianResult{}, want: false},
		{name: "whitespace only", result: LibrarianResult{Summary: "  "}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasUsableLocalResult(tt.result)
			if got != tt.want {
				t.Errorf("hasUsableLocalResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasUsableJournalistResult(t *testing.T) {
	tests := []struct {
		name   string
		result *JournalistResult
		want   bool
	}{
		{name: "nil", result: nil, want: false},
		{name: "has summary", result: &JournalistResult{Summary: "research findings"}, want: true},
		{name: "has sources", result: &JournalistResult{Sources: []SourceRef{{Title: "article"}}}, want: true},
		{name: "both present", result: &JournalistResult{Summary: "x", Sources: []SourceRef{{}}}, want: true},
		{name: "empty", result: &JournalistResult{}, want: false},
		{name: "whitespace only", result: &JournalistResult{Summary: "  "}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasUsableJournalistResult(tt.result)
			if got != tt.want {
				t.Errorf("hasUsableJournalistResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnsureManualEvidence(t *testing.T) {
	localErr := errors.New("local search failed")
	webErr := errors.New("web search failed")
	usableLocal := LibrarianResult{Summary: "valid local result"}
	usableWeb := JournalistResult{Summary: "valid web result"}

	tests := []struct {
		name     string
		local    LibrarianResult
		web      *JournalistResult
		localErr error
		webErr   error
		errMsg   string
	}{
		{name: "local usable", local: usableLocal, web: nil, localErr: nil, webErr: nil, errMsg: ""},
		{name: "web usable", local: LibrarianResult{}, web: &usableWeb, localErr: nil, webErr: nil, errMsg: ""},
		{name: "both usable", local: usableLocal, web: &usableWeb, localErr: nil, webErr: nil, errMsg: ""},
		{name: "both failed", local: LibrarianResult{}, web: &JournalistResult{}, localErr: localErr, webErr: webErr, errMsg: "本地检索失败"},
		{name: "only local failed", local: LibrarianResult{}, web: &JournalistResult{}, localErr: localErr, webErr: nil, errMsg: localErr.Error()},
		{name: "only web failed", local: LibrarianResult{}, web: &JournalistResult{}, localErr: nil, webErr: webErr, errMsg: webErr.Error()},
		{name: "neither usable but no errors", local: LibrarianResult{}, web: &JournalistResult{}, localErr: nil, webErr: nil, errMsg: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ensureManualEvidence(tt.local, tt.web, tt.localErr, tt.webErr)
			if tt.errMsg == "" {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
				}
			}
		})
	}
}
