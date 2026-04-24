// 用户相关类型
export interface User {
  id: number;
  username: string;
  email: string;
  avatar_url?: string;
  bio?: string;
  role: 'user' | 'admin';
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
}

export interface AuthResponse {
  user: User;
  access_token: string;
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
  tags?: string[];
  created_at: string;
  updated_at: string;
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
  created_at: string;
}

export interface CreateArticleRequest {
  title: string;
  summary: string;
  content: string;
  cover_image?: string;
  category_id: number | undefined;
  status: 'draft' | 'published';
  tags?: string[];
}

// 分类相关类型
export interface Category {
  id: number;
  name: string;
  slug: string;
  description?: string;
  article_count: number;
  created_at: string;
  updated_at: string;
}

// 评论相关类型
export interface Comment {
  id: number;
  content: string;
  article_id: number;
  user_id: number;
  user: User;
  parent_id?: number;
  reply_to_user_id?: number;
  reply_to_user?: User;
  replies?: Comment[];
  status: string;
  article?: Article;
  created_at: string;
  updated_at: string;
}

export interface CreateCommentRequest {
  content: string;
  articleId: number;
  parentId?: number;
  replyToUserId?: number;
}

// AI 聊天相关类型
export type ChatStage =
  | 'analyzing'
  | 'clarifying'
  | 'adk_event'
  | 'local_search'
  | 'web_research'
  | 'integration'
  | 'synthesizing'
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
  step_id: number;
  agent_name: string;
  status: 'running' | 'completed' | 'failed';
  summary: string;
  detail: string;
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
}

export interface ChatRequest {
  message: string;
  article_id?: number;
  conversation_id?: number;
}

export interface ChatResponse {
  message: string;
  sources?: string[];
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
