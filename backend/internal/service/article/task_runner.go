package article

import "wenDao/internal/pkg/async"

func WithTaskRunner(runner async.Runner) ArticleServiceOption {
	return func(s *articleService) {
		s.taskRunner = runner
	}
}
