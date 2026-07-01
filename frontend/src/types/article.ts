import type { User } from './auth';

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

export interface Tag {
  id: number;
  name: string;
  slug: string;
  article_count: number;
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

export interface ArticleSearchResult {
  article: ArticleListItem;
  snippet: string;
  matched_fields: string[];
}
