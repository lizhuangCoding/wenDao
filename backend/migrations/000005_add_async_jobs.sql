CREATE TABLE IF NOT EXISTS async_jobs (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    job_type VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL,
    payload JSON NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    run_after DATETIME(3) NOT NULL,
    locked_at DATETIME(3) NULL,
    last_error TEXT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    INDEX idx_async_jobs_dispatch (job_type, status, run_after)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

