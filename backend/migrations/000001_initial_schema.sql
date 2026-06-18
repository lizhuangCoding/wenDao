CREATE TABLE IF NOT EXISTS users (
  id BIGINT NOT NULL AUTO_INCREMENT,
  username VARCHAR(50) NOT NULL,
  email VARCHAR(100) NOT NULL,
  password_hash VARCHAR(255) NULL,
  role VARCHAR(10) NOT NULL DEFAULT 'user',
  o_auth_provider VARCHAR(20) NULL,
  o_auth_id VARCHAR(100) NULL,
  avatar_url VARCHAR(500) NULL,
  avatar_source VARCHAR(20) NOT NULL DEFAULT 'default',
  email_verified BOOLEAN DEFAULT FALSE,
  comment_reply_email_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  status VARCHAR(10) NOT NULL DEFAULT 'active',
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS categories (
  id BIGINT NOT NULL AUTO_INCREMENT,
  name VARCHAR(50) NOT NULL,
  slug VARCHAR(50) NOT NULL,
  description VARCHAR(200) NULL,
  article_count BIGINT DEFAULT 0,
  sort_order BIGINT DEFAULT 0,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_categories_name (name),
  UNIQUE KEY idx_categories_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS collections (
  id BIGINT NOT NULL AUTO_INCREMENT,
  name VARCHAR(100) NOT NULL,
  slug VARCHAR(100) NOT NULL,
  description VARCHAR(500) NULL,
  article_count BIGINT DEFAULT 0,
  sort_order BIGINT DEFAULT 0,
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_collections_name (name),
  UNIQUE KEY idx_collections_slug (slug),
  KEY idx_collections_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS articles (
  id BIGINT NOT NULL AUTO_INCREMENT,
  title VARCHAR(200) NOT NULL,
  slug VARCHAR(200) NOT NULL,
  summary TEXT NULL,
  content LONGTEXT NOT NULL,
  content_html LONGTEXT NULL,
  category_id BIGINT NOT NULL,
  author_id BIGINT NOT NULL,
  cover_image VARCHAR(500) NULL,
  status VARCHAR(10) NOT NULL DEFAULT 'draft',
  ai_index_status VARCHAR(20) NOT NULL DEFAULT 'pending',
  source_type VARCHAR(32) NOT NULL DEFAULT 'manual',
  source_id BIGINT NULL,
  view_count BIGINT DEFAULT 0,
  comment_count BIGINT DEFAULT 0,
  like_count BIGINT DEFAULT 0,
  is_top BOOLEAN DEFAULT FALSE,
  popularity DOUBLE DEFAULT 0,
  published_at DATETIME(3) NULL,
  scheduled_publish_at DATETIME(3) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_articles_slug (slug),
  KEY idx_category (category_id),
  KEY idx_status_published (status, published_at),
  KEY idx_article_source (source_type, source_id),
  KEY idx_popularity (popularity),
  KEY idx_articles_scheduled_publish_at (scheduled_publish_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS article_semantic_profiles (
  article_id BIGINT NOT NULL,
  embedding_json LONGTEXT NOT NULL,
  content_hash VARCHAR(64) NOT NULL,
  map_x DOUBLE NOT NULL DEFAULT 0,
  map_y DOUBLE NOT NULL DEFAULT 0,
  map_z DOUBLE NOT NULL DEFAULT 0,
  neighbor_json MEDIUMTEXT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (article_id),
  KEY idx_article_semantic_profiles_content_hash (content_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS article_collections (
  id BIGINT NOT NULL AUTO_INCREMENT,
  collection_id BIGINT NOT NULL,
  article_id BIGINT NOT NULL,
  position BIGINT NOT NULL DEFAULT 0,
  is_primary BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_collection_article (collection_id, article_id),
  KEY idx_article_collections_article_id (article_id),
  KEY idx_article_collections_is_primary (is_primary),
  KEY idx_article_collection_collection_position (collection_id, position)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS article_interactions (
  id BIGINT NOT NULL AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  article_id BIGINT NOT NULL,
  interaction_type VARCHAR(20) NOT NULL,
  created_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_article_interaction_unique (user_id, article_id, interaction_type),
  KEY idx_article_interaction_user_type (user_id, interaction_type, created_at),
  KEY idx_article_interaction_article_type (article_id, interaction_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS comments (
  id BIGINT NOT NULL AUTO_INCREMENT,
  article_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  parent_id BIGINT NULL,
  content TEXT NOT NULL,
  root_id BIGINT NULL,
  reply_to_user_id BIGINT NULL,
  like_count BIGINT DEFAULT 0,
  dislike_count BIGINT DEFAULT 0,
  status VARCHAR(10) NOT NULL DEFAULT 'normal',
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_article (article_id, created_at),
  KEY idx_parent (parent_id),
  KEY idx_comments_root_id (root_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS uploads (
  id BIGINT NOT NULL AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  filename VARCHAR(255) NOT NULL,
  file_path VARCHAR(500) NOT NULL,
  file_size BIGINT NOT NULL,
  mime_type VARCHAR(100) NULL,
  file_type VARCHAR(10) DEFAULT 'image',
  created_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS settings (
  id BIGINT NOT NULL AUTO_INCREMENT,
  `key` VARCHAR(100) NOT NULL,
  value TEXT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_settings_key (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS daily_stats (
  id BIGINT NOT NULL AUTO_INCREMENT,
  date VARCHAR(10) NOT NULL,
  pv BIGINT DEFAULT 0,
  uv BIGINT DEFAULT 0,
  comment_count BIGINT DEFAULT 0,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_daily_stats_date (date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS article_stats (
  id BIGINT NOT NULL AUTO_INCREMENT,
  article_id BIGINT NOT NULL,
  date VARCHAR(10) NOT NULL,
  pv BIGINT DEFAULT 0,
  created_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_article_date (article_id, date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS conversations (
  id BIGINT NOT NULL AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  title VARCHAR(255) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  model_provider VARCHAR(50) NULL,
  model_name VARCHAR(100) NULL,
  share_token VARCHAR(64) NULL,
  is_shared BOOLEAN DEFAULT FALSE,
  PRIMARY KEY (id),
  UNIQUE KEY idx_conversations_share_token (share_token),
  KEY idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS chat_messages (
  id BIGINT NOT NULL AUTO_INCREMENT,
  conversation_id BIGINT NOT NULL,
  run_id BIGINT NULL,
  role VARCHAR(20) NOT NULL,
  content TEXT NULL,
  created_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_conversation (conversation_id),
  KEY idx_chat_message_run (run_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS conversation_memories (
  id BIGINT NOT NULL AUTO_INCREMENT,
  conversation_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  scope VARCHAR(32) NOT NULL,
  content TEXT NOT NULL,
  source_message_id_start BIGINT NOT NULL DEFAULT 0,
  source_message_id_end BIGINT NOT NULL DEFAULT 0,
  importance BIGINT NOT NULL DEFAULT 1,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_conversation_memory_conversation (conversation_id),
  KEY idx_conversation_memory_user (user_id),
  KEY idx_conversation_memory_scope (scope)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS conversation_runs (
  id BIGINT NOT NULL AUTO_INCREMENT,
  conversation_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  status VARCHAR(32) NOT NULL,
  current_stage VARCHAR(32) NOT NULL,
  original_question TEXT NOT NULL,
  normalized_question TEXT NULL,
  pending_question TEXT NULL,
  pending_context LONGTEXT NULL,
  last_answer LONGTEXT NULL,
  last_plan LONGTEXT NULL,
  last_error TEXT NULL,
  heartbeat_at DATETIME(3) NULL,
  completed_at DATETIME(3) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_conversation_run_conversation (conversation_id),
  KEY idx_conversation_run_user (user_id),
  KEY idx_conversation_run_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS conversation_run_steps (
  id BIGINT NOT NULL AUTO_INCREMENT,
  conversation_id BIGINT NOT NULL,
  run_id BIGINT NOT NULL,
  agent_name VARCHAR(64) NOT NULL,
  type VARCHAR(32) NOT NULL,
  summary TEXT NOT NULL,
  detail LONGTEXT NULL,
  status VARCHAR(32) NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_conversation_run_step_conversation (conversation_id),
  KEY idx_conversation_run_step_run (run_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS knowledge_documents (
  id BIGINT NOT NULL AUTO_INCREMENT,
  title VARCHAR(255) NOT NULL,
  summary TEXT NULL,
  content LONGTEXT NOT NULL,
  status VARCHAR(32) NOT NULL,
  source_type VARCHAR(32) NOT NULL,
  created_by_user_id BIGINT NOT NULL,
  reviewed_by_user_id BIGINT NULL,
  reviewed_at DATETIME(3) NULL,
  review_note TEXT NULL,
  vectorized_at DATETIME(3) NULL,
  article_id BIGINT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_knowledge_document_status (status),
  KEY idx_knowledge_document_created_by (created_by_user_id),
  KEY idx_knowledge_document_reviewed_by (reviewed_by_user_id),
  KEY idx_knowledge_document_article (article_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS knowledge_document_sources (
  id BIGINT NOT NULL AUTO_INCREMENT,
  knowledge_document_id BIGINT NOT NULL,
  source_url TEXT NOT NULL,
  source_title VARCHAR(500) NULL,
  source_domain VARCHAR(255) NULL,
  source_snippet TEXT NULL,
  sort_order BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_kd_source_document (knowledge_document_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS notifications (
  id BIGINT NOT NULL AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  type VARCHAR(30) NOT NULL,
  title VARCHAR(200) NOT NULL,
  content TEXT NOT NULL,
  link_url VARCHAR(500) NULL,
  is_read BOOLEAN DEFAULT FALSE,
  created_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_notifications_type (type),
  KEY idx_user_unread (user_id, is_read)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
