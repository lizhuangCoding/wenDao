import { request } from './client';
import type {
  Article,
  ArticleListItem,
  CreateArticleRequest,
  PaginatedResponse,
  PaginationParams,
} from '@/types';

// 文章 API
export const articleApi = {
  // 获取文章列表（公开）
  getArticles: (params: PaginationParams & { category_id?: number; keyword?: string }) => {
    return request.get<PaginatedResponse<ArticleListItem>>('/articles', { params });
  },

  // 获取文章详情（通过 slug）
  getArticleBySlug: (slug: string) => {
    return request.get<Article>(`/articles/slug/${slug}`);
  },

  // --- 管理员接口 ---

  // 获取所有文章列表（管理员）
  getAdminArticles: (params: PaginationParams & { status?: string; category_id?: number; keyword?: string; sort_by_popularity?: boolean }) => {
    return request.get<PaginatedResponse<ArticleListItem>>('/admin/articles', { params });
  },

  // 获取单个文章详情（管理员）
  getAdminArticleById: (id: number) => {
    return request.get<Article>(`/admin/articles/${id}`);
  },

  // 创建文章
  createArticle: (data: CreateArticleRequest) => {
    return request.post<Article>('/admin/articles', data);
  },

  // 更新文章
  updateArticle: (id: number, data: Partial<CreateArticleRequest>) => {
    return request.put<Article>(`/admin/articles/${id}`, data);
  },

  // 删除文章
  deleteArticle: (id: number) => {
    return request.delete(`/admin/articles/${id}`);
  },

  // 发布文章
  publishArticle: (id: number) => {
    return request.patch(`/admin/articles/${id}/publish`);
  },

  // 转为草稿
  draftArticle: (id: number) => {
    return request.patch(`/admin/articles/${id}/draft`);
  },

  // 自动保存草稿
  autoSave: (id: number, data: { title: string; content: string; summary: string }) => {
    return request.put(`/admin/articles/${id}/autosave`, data);
  },

  // 切换置顶
  toggleTop: (id: number) => {
    return request.patch(`/admin/articles/${id}/top`);
  },

  // 刷新活跃度分数
  refreshScores: () => {
    return request.post('/admin/articles/refresh-scores');
  },
};
