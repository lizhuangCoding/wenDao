package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func (o *thinkTankOrchestrator) searchLocal(ctx context.Context, query string) (LibrarianResult, error) {
	if o == nil || o.service == nil || o.service.librarian == nil {
		return LibrarianResult{CoverageStatus: "insufficient"}, errors.New("local search unavailable")
	}
	result, err := o.service.librarian.Search(ctx, query)
	if strings.TrimSpace(result.CoverageStatus) == "" {
		result.CoverageStatus = "insufficient"
	}
	return result, err
}

func shouldRunJournalist(local LibrarianResult, localErr error) bool {
	return localErr != nil || strings.TrimSpace(local.CoverageStatus) != "sufficient"
}

func ensureManualEvidence(local LibrarianResult, web *JournalistResult, localErr error, webErr error) error {
	if hasUsableLocalResult(local) || hasUsableJournalistResult(web) {
		return nil
	}
	switch {
	case localErr != nil && webErr != nil:
		return fmt.Errorf("本地检索失败: %v；外部调研失败: %v", localErr, webErr)
	case localErr != nil:
		return localErr
	case webErr != nil:
		return webErr
	default:
		return nil
	}
}

func hasUsableLocalResult(result LibrarianResult) bool {
	return strings.TrimSpace(result.Summary) != "" || len(result.Sources) > 0
}

func hasUsableJournalistResult(result *JournalistResult) bool {
	if result == nil {
		return false
	}
	return strings.TrimSpace(result.Summary) != "" || len(result.Sources) > 0
}
