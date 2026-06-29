package async

import (
	"context"
	"errors"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

var ErrTaskRunnerClosed = errors.New("task runner closed")

type TaskRunnerStats struct {
	Submitted int64
	Running   int64
	Succeeded int64
	Failed    int64
	Retried   int64
	Panicked  int64
	TimedOut  int64
	Canceled  int64
}

type Runner interface {
	Submit(ctx context.Context, task string, fn func(context.Context) error, opts ...TaskOption) error
	Shutdown(ctx context.Context) error
	Stats() TaskRunnerStats
}

type TaskRunner struct {
	ctx           context.Context
	cancel        context.CancelFunc
	logger        *zap.Logger
	defaultConfig taskConfig
	wg            sync.WaitGroup
	closed        atomic.Bool
	submitted     atomic.Int64
	running       atomic.Int64
	succeeded     atomic.Int64
	failed        atomic.Int64
	retried       atomic.Int64
	panicked      atomic.Int64
	timedOut      atomic.Int64
	canceled      atomic.Int64
}

type RunnerOption func(*TaskRunner)
type TaskOption func(*taskConfig)

type taskConfig struct {
	timeout    time.Duration
	retries    int
	retryDelay func(int) time.Duration
}

func NewTaskRunner(parent context.Context, logger *zap.Logger, opts ...RunnerOption) *TaskRunner {
	if parent == nil {
		parent = context.Background()
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	ctx, cancel := context.WithCancel(parent)
	runner := &TaskRunner{
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
		defaultConfig: taskConfig{
			timeout: 5 * time.Second,
			retryDelay: func(attempt int) time.Duration {
				if attempt <= 0 {
					return 0
				}
				return time.Duration(attempt) * 100 * time.Millisecond
			},
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(runner)
		}
	}
	return runner
}

func WithDefaultTimeout(timeout time.Duration) RunnerOption {
	return func(r *TaskRunner) {
		r.defaultConfig.timeout = timeout
	}
}

func WithDefaultRetryDelay(fn func(int) time.Duration) RunnerOption {
	return func(r *TaskRunner) {
		if fn != nil {
			r.defaultConfig.retryDelay = fn
		}
	}
}

func WithTimeout(timeout time.Duration) TaskOption {
	return func(cfg *taskConfig) {
		cfg.timeout = timeout
	}
}

func WithRetries(retries int) TaskOption {
	return func(cfg *taskConfig) {
		if retries < 0 {
			retries = 0
		}
		cfg.retries = retries
	}
}

func WithRetryDelay(fn func(int) time.Duration) TaskOption {
	return func(cfg *taskConfig) {
		if fn != nil {
			cfg.retryDelay = fn
		}
	}
}

func (r *TaskRunner) Submit(ctx context.Context, task string, fn func(context.Context) error, opts ...TaskOption) error {
	if r == nil || fn == nil {
		return nil
	}
	if r.closed.Load() {
		return ErrTaskRunnerClosed
	}

	cfg := r.defaultConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	r.submitted.Add(1)
	r.running.Add(1)
	r.wg.Add(1)

	go func() {
		defer r.wg.Done()
		defer r.running.Add(-1)
		defer func() {
			if recovered := recover(); recovered != nil {
				r.panicked.Add(1)
				r.failed.Add(1)
				r.logger.Error("Async task panicked",
					zap.String("task", task),
					zap.Any("panic", recovered),
					zap.ByteString("stack", debug.Stack()))
			}
		}()

		valuesCtx := ctx
		if valuesCtx == nil {
			valuesCtx = context.Background()
		}
		runCtx, stopBaseForward := mergeRunnerContext(r.ctx, ctx)
		defer stopBaseForward()

		for attempt := 0; ; attempt++ {
			attemptCtx := context.Context(runCtx)
			attemptStop := func() {}
			if cfg.timeout > 0 {
				var cancel context.CancelFunc
				attemptCtx, cancel = context.WithTimeout(valueContext{lifecycle: runCtx, values: valuesCtx}, cfg.timeout)
				attemptStop = cancel
			} else {
				attemptCtx = valueContext{lifecycle: runCtx, values: valuesCtx}
			}

			err := fn(attemptCtx)
			attemptStop()

			if err == nil {
				r.succeeded.Add(1)
				return
			}
			if errors.Is(err, context.DeadlineExceeded) {
				r.timedOut.Add(1)
			}
			if errors.Is(err, context.Canceled) && runCtx.Err() != nil {
				r.canceled.Add(1)
				return
			}
			if attempt < cfg.retries && runCtx.Err() == nil {
				r.retried.Add(1)
				delay := time.Duration(0)
				if cfg.retryDelay != nil {
					delay = cfg.retryDelay(attempt + 1)
				}
				if delay > 0 {
					timer := time.NewTimer(delay)
					select {
					case <-timer.C:
					case <-runCtx.Done():
						timer.Stop()
						r.canceled.Add(1)
						return
					}
				}
				continue
			}

			r.failed.Add(1)
			r.logger.Warn("Async task failed",
				zap.String("task", task),
				zap.Int("attempt", attempt+1),
				zap.Error(err))
			return
		}
	}()

	return nil
}

func (r *TaskRunner) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if !r.closed.CompareAndSwap(false, true) {
		return ErrTaskRunnerClosed
	}
	r.cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.wg.Wait()
	}()

	if ctx == nil {
		<-done
		return nil
	}

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *TaskRunner) Stats() TaskRunnerStats {
	if r == nil {
		return TaskRunnerStats{}
	}
	return TaskRunnerStats{
		Submitted: r.submitted.Load(),
		Running:   r.running.Load(),
		Succeeded: r.succeeded.Load(),
		Failed:    r.failed.Load(),
		Retried:   r.retried.Load(),
		Panicked:  r.panicked.Load(),
		TimedOut:  r.timedOut.Load(),
		Canceled:  r.canceled.Load(),
	}
}

type valueContext struct {
	lifecycle context.Context
	values    context.Context
}

func (c valueContext) Deadline() (time.Time, bool) {
	return c.lifecycle.Deadline()
}

func (c valueContext) Done() <-chan struct{} {
	return c.lifecycle.Done()
}

func (c valueContext) Err() error {
	return c.lifecycle.Err()
}

func (c valueContext) Value(key any) any {
	if c.values == nil {
		return nil
	}
	return c.values.Value(key)
}

func mergeRunnerContext(runnerCtx, baseCtx context.Context) (context.Context, func()) {
	if runnerCtx == nil {
		runnerCtx = context.Background()
	}
	if baseCtx == nil || baseCtx.Done() == nil {
		return runnerCtx, func() {}
	}

	ctx, cancel := context.WithCancel(runnerCtx)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		select {
		case <-runnerCtx.Done():
		case <-baseCtx.Done():
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, func() {
		cancel()
		<-stopped
	}
}
