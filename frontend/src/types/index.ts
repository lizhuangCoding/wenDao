// 用户相关类型
export interface User {
  id: number;
  username: string;
  email: string;
  avatar_url?: string;
  bio?: string;
  role: 'user' | 'admin';
  status: 'active' | 'banned';
  comment_reply_email_enabled?: boolean;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
  verification_code: string;
}

export interface VerificationCodeRequest {
  email: string;
}

export interface PasswordResetConfirmRequest {
  email: string;
  password: string;
  verification_code: string;
}

export interface UpdatePreferencesRequest {
  comment_reply_email_enabled: boolean;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface CurrentUserResponse {
  user: User;
  expires_in: number;
}

// 文章相关类型
export interface Article {
  id: number;
  title: string;
  slug: string;
  summary: string;
  content: string;
  cover_image?: string;
  status: 'draft' | 'published';
  is_top: boolean;
  ai_index_status: 'pending' | 'success' | 'failed';
  source_type: 'manual' | 'knowledge_document';
  source_id?: number;
  view_count: number;
  like_count: number;
  comment_count: number;
  category_id: number;
  category: Category;
  author_id: number;
  author: User;
  tags?: Tag[];
  collection_id?: number;
  collection_position?: number;
  collection_membership?: ArticleCollectionMembership;
  collection_navigation?: ArticleCollectionNavigation;
  published_at?: string;
  scheduled_publish_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Collection {
  id: number;
  name: string;
  slug: string;
  description?: string;
  article_count: number;
  sort_order: number;
  status: 'active' | 'hidden';
  created_at: string;
  updated_at: string;
}

export interface ArticleCollectionMembership {
  collection_id: number;
  name: string;
  slug: string;
  position: number;
}

export interface CollectionNavigationArticle {
  id: number;
  title: string;
  slug: string;
}

export interface ArticleCollectionNavigation {
  collection_id: number;
  collection_name: string;
  collection_slug: string;
  position: number;
  total: number;
  previous?: CollectionNavigationArticle;
  next?: CollectionNavigationArticle;
}

export interface ArticleListItem {
  id: number;
  title: string;
  slug: string;
  summary: string;
  cover_image?: string;
  view_count: number;
  like_count: number;
  comment_count: number;
  status: 'draft' | 'published';
  is_top: boolean;
  ai_index_status: 'pending' | 'success' | 'failed';
  source_type: 'manual' | 'knowledge_document';
  source_id?: number;
  category: Category;
  author: User;
  tags?: Tag[];
  created_at: string;
}

export interface ArticleOrbitCategory {
  id: number;
  name: string;
  slug: string;
}

export interface ArticleOrbitCollection {
  id: number;
  name: string;
  slug: string;
  position: number;
}

export interface ArticleOrbitSemanticPosition {
  x: number;
  y: number;
  z: number;
}

export interface ArticleOrbitSemanticNeighbor {
  article_id: number;
  score: number;
}

export interface ArticleOrbitItem {
  id: number;
  title: string;
  slug: string;
  summary: string;
  cover_image?: string;
  view_count: number;
  comment_count: number;
  is_top: boolean;
  source_type: 'manual' | 'knowledge_document';
  category?: ArticleOrbitCategory;
  collection?: ArticleOrbitCollection;
  semantic_position?: ArticleOrbitSemanticPosition;
  semantic_neighbors?: ArticleOrbitSemanticNeighbor[];
  created_at: string;
  published_at: string;
}

export interface ArticleOrbitResponse {
  data: ArticleOrbitItem[];
  total: number;
}

export interface ArticleInteractionState {
  liked: boolean;
  favorited: boolean;
}

export interface CreateArticleRequest {
  title: string;
  summary: string;
  content: string;
  cover_image?: string;
  category_id: number | undefined;
  status: 'draft' | 'published';
  tag_ids?: number[];
  scheduled_publish_at?: string;
  collection_id?: number;
  collection_position?: number;
}

// 分类相关类型
export interface Category {
  id: number;
  name: string;
  slug: string;
  description?: string;
  article_count: number;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

// 标签相关类型
export interface Tag {
  id: number;
  name: string;
  slug: string;
  article_count: number;
  created_at: string;
  updated_at: string;
}

export interface ContactLink {
  type: string;
  label: string;
  value: string;
  url?: string;
  enabled: boolean;
  sort_order: number;
}

// 评论相关类型
export interface Comment {
  id: number;
  content: string;
  article_id: number;
  user_id: number;
  user?: User;
  parent_id?: number;
  reply_to_user_id?: number;
  reply_to_user?: User;
  replies?: Comment[];
  status: string;
  article?: Article;
  like_count: number;
  dislike_count: number;
  created_at: string;
  updated_at: string;
}

export interface CreateCommentRequest {
  content: string;
  articleId: number;
  parentId?: number;
  replyToUserId?: number;
}

// 通知相关类型
export type NotificationType =
  | 'comment_reply'
  | 'comment_like'
  | 'admin_broadcast'
  | 'system_notice';

export interface Notification {
  id: number;
  user_id: number;
  type: NotificationType;
  title: string;
  content: string;
  link_url: string;
  is_read: boolean;
  created_at: string;
}

// AI 聊天相关类型
export type ChatStage =
  | 'analyzing'
  | 'clarifying_intent'
  | 'clarifying'
  | 'adk_event'
  | 'local_search'
  | 'web_research'
  | 'integration'
  | 'synthesizing'
  | 'reviewing'
  | 'revising'
  | 'streaming'
  | 'completed'
  | 'failed';

export interface ChatStageEvent {
  stage: ChatStage;
  label?: string;
}

export interface ChatStep {
  id: number;
  run_id?: number;
  agent_name: string;
  type: string;
  summary: string;
  detail: string;
  status: 'running' | 'completed' | 'failed';
  created_at: string;
}

export interface ChatStepEvent {
  run_id?: number;
  step_id: number;
  agent_name: string;
  status: 'running' | 'completed' | 'failed';
  summary: string;
  detail: string;
}

export interface ChatActiveRun {
  id: number;
  status: 'idle' | 'running' | 'waiting_user' | 'completed' | 'failed';
  current_stage: ChatStage | 'streaming';
  pending_question?: string;
  last_answer: string;
  heartbeat_at?: string;
  can_resume: boolean;
}

export interface ChatResumeEvent {
  run_id: number;
  stage?: ChatStage | 'streaming';
  status?: ChatActiveRun['status'];
}

export interface ChatSnapshotEvent {
  run_id: number;
  stage?: ChatStage | 'streaming';
  status?: ChatActiveRun['status'];
  message?: string;
}

export interface ChatHeartbeatEvent {
  run_id: number;
  stage?: ChatStage | 'streaming';
  status?: ChatActiveRun['status'];
}

export interface ChatArticleReference {
  title: string;
  url: string;
}

export interface ChatReferenceGroups {
  blog: ChatArticleReference[];
  external: ChatArticleReference[];
}

export interface ChatQuestionEvent {
  run_id?: number;
  stage: 'clarifying';
  message: string;
  requires_user_input: true;
}

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: number;
  processSteps?: ChatStep[];
  runId?: number;
}

export interface ChatRequest {
  message: string;
  article_id?: number;
  conversation_id?: number;
  model_provider?: string;
  model_name?: string;
}

export interface ModelInfo {
  provider: string;
  model_name: string;
  display_name: string;
}

export interface ChatResponse {
  message: string;
  sources?: string[];
}

export interface SharedConversationData {
  conversation: {
    id: number;
    user_id: number;
    title: string;
    is_shared: boolean;
    share_token: string;
    created_at: string;
    updated_at: string;
  };
  messages: Array<{
    id: number;
    conversation_id: number;
    run_id?: number;
    role: 'user' | 'assistant';
    content: string;
    created_at: string;
    process_steps?: ChatStep[];
  }>;
  steps?: ChatStep[];
  shared_by: {
    username: string;
    avatar_url?: string;
  };
}

export interface ChatConversationDetailResponse {
  conversation: {
    id: number;
    title: string;
    user_id: number;
    is_shared: boolean;
    share_token?: string;
    created_at: string;
    updated_at: string;
  };
  messages: Array<{
    id: number;
    conversation_id: number;
    run_id?: number;
    role: 'user' | 'assistant';
    content: string;
    created_at: string;
    process_steps?: ChatStep[];
  }>;
  steps?: ChatStep[];
  active_run?: ChatActiveRun;
  active_steps?: ChatStep[];
}

export interface KnowledgeDocument {
  id: number;
  title: string;
  summary: string;
  content: string;
  status: 'pending_review' | 'approved' | 'rejected';
  source_type: 'research' | 'manual';
  created_by_user_id: number;
  reviewed_by_user_id?: number;
  reviewed_at?: string;
  review_note: string;
  article_id?: number;
  vectorized_at?: string;
  created_at: string;
  updated_at: string;
}

export interface KnowledgeDocumentSource {
  id: number;
  knowledge_document_id: number;
  source_url: string;
  source_title: string;
  source_domain: string;
  source_snippet: string;
  sort_order: number;
}

export interface AIObservabilityToolUsage {
  local_search: number;
  web_search: number;
  web_fetch: number;
  doc_writer: number;
  other: number;
}

export interface AIObservabilitySources {
  local_hits: number;
  web_hits: number;
  quality_score: number;
  external_urls: Array<{
    url: string;
    quality_score: number;
  }>;
}

export interface AIObservabilityCost {
  status: 'not_collected' | 'tokens_only' | 'estimated' | 'collected';
  prompt_tokens: number;
  completion_tokens: number;
  estimated_cost: number;
  currency: string;
}

export interface AIObservabilityFeedback {
  status: 'not_collected' | 'collected';
  score?: number;
}

export interface AIObservabilityFailedStep {
  id: number;
  agent_name: string;
  type: string;
  summary: string;
  detail: string;
  category: string;
  created_at: string;
}

export interface AIObservabilityFailureCluster {
  category: string;
  count: number;
}

export interface AIObservabilityStep {
  id: number;
  agent_name: string;
  type: string;
  summary: string;
  status: string;
  created_at: string;
}

export interface AIObservabilityRun {
  id: number;
  conversation_id: number;
  user_id: number;
  status: string;
  current_stage: string;
  original_question: string;
  normalized_question: string;
  last_error?: string;
  duration_seconds: number;
  step_count: number;
  failed_step_count: number;
  tool_usage: AIObservabilityToolUsage;
  sources: AIObservabilitySources;
  cost: AIObservabilityCost;
  feedback: AIObservabilityFeedback;
  failure_category?: string;
  failure_fingerprint?: string;
  failed_steps: AIObservabilityFailedStep[];
  failure_clusters: AIObservabilityFailureCluster[];
  steps: AIObservabilityStep[];
  created_at: string;
  updated_at: string;
  completed_at?: string;
  heartbeat_at?: string;
}

// 分页相关类型
export interface PaginationParams {
  page: number;
  pageSize: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

// API 响应类型
export interface ApiResponse<T = any> {
  code: number;
  message: string;
  data: T;
}

export interface ApiError {
  code: number;
  message: string;
}
