import { request } from './client';
import type { Category, PaginatedResponse, PaginationParams } from '@/types';
import { toPaginationQuery } from './pagination';

type CategoryFormPayload = {
  name: string;
  slug: string;
  description?: string;
  sort_order: number;
};

// 分类 API
export const categoryApi = {
  // 获取所有分类
  getCategories: () => {
    return request.get<Category[]>('/categories');
  },

  // 获取分类分页列表（管理员）
  getAdminCategories: (params: PaginationParams) => {
    return request.get<PaginatedResponse<Category>>('/admin/categories', { params: toPaginationQuery(params) });
  },

  // 获取分类详情
  getCategory: (id: number) => {
    return request.get<Category>(`/categories/${id}`);
  },

  // 创建分类（管理员）
  createCategory: (data: CategoryFormPayload) => {
    return request.post<Category>('/admin/categories', data);
  },

  // 更新分类（管理员）
  updateCategory: (id: number, data: CategoryFormPayload) => {
    return request.put<Category>(`/admin/categories/${id}`, data);
  },

  // 删除分类（管理员）
  deleteCategory: (id: number) => {
    return request.delete(`/admin/categories/${id}`);
  },

  // 批量删除分类（管理员）
  batchDeleteCategories: (ids: number[]) => {
    return request.post<{ message: string; deleted_count: number }>('/admin/categories/batch-delete', { ids });
  },
};
