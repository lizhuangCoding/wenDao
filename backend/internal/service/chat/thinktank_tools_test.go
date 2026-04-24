package chat

import (
	"context"
	"testing"
)

func TestWithRunID_PropagatesRunIDThroughToolContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithRunID(ctx, 42)

	if got := getRunID(ctx); got != 42 {
		t.Fatalf("expected run_id 42, got %d", got)
	}
}
