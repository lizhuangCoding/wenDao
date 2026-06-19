CREATE TABLE IF NOT EXISTS tags (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(50) NOT NULL,
  slug VARCHAR(50) NOT NULL,
  article_count INT NOT NULL DEFAULT 0,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE KEY idx_tags_name (name),
  UNIQUE KEY idx_tags_slug (slug)
);

CREATE TABLE IF NOT EXISTS article_tags (
  article_id BIGINT NOT NULL,
  tag_id BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  PRIMARY KEY (article_id, tag_id),
  KEY idx_article_tags_article_id (article_id),
  KEY idx_article_tags_tag_id (tag_id)
);
