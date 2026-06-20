package async

import (
	"context"
	"runtime/debug"

	"go.uber.org/zap"
)

// Go starts a fire-and-forget task with panic recovery and structured logging.
func Go(ctx context.Context, logger *zap.Logger, task string, fn func(context.Context) error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("Async task panicked",
					zap.String("task", task),
					zap.Any("panic", recovered),
					zap.ByteString("stack", debug.Stack()))
			}
		}()

		if err := fn(ctx); err != nil {
			logger.Warn("Async task failed",
				zap.String("task", task),
				zap.Error(err))
		}
	}()
}
