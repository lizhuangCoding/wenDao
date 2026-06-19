ALTER TABLE conversation_runs
  ADD COLUMN provider VARCHAR(64) NULL AFTER last_error,
  ADD COLUMN model_name VARCHAR(128) NULL AFTER provider,
  ADD COLUMN prompt_tokens BIGINT NOT NULL DEFAULT 0 AFTER model_name,
  ADD COLUMN completion_tokens BIGINT NOT NULL DEFAULT 0 AFTER prompt_tokens,
  ADD COLUMN estimated_cost DOUBLE NOT NULL DEFAULT 0 AFTER completion_tokens,
  ADD COLUMN cost_currency VARCHAR(16) NOT NULL DEFAULT 'USD' AFTER estimated_cost,
  ADD COLUMN cost_status VARCHAR(32) NOT NULL DEFAULT 'not_collected' AFTER cost_currency,
  ADD COLUMN source_quality_score BIGINT NOT NULL DEFAULT 0 AFTER cost_status,
  ADD COLUMN failure_category VARCHAR(64) NULL AFTER source_quality_score,
  ADD COLUMN failure_fingerprint VARCHAR(128) NULL AFTER failure_category,
  ADD KEY idx_conversation_run_user_created (user_id, created_at),
  ADD KEY idx_conversation_run_failure_category (failure_category);
