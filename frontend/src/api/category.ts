import { request } from './client';
import type { Category } from '@/types';

// 分类 API
export const categoryApi = {
  // 获取所有分类
  getCategories: () => {
    return request.get<Category[]>('/categories');
  },

  // 获取分类详情
  getCategory: (id: number) => {
    return request.get<Category>(`/categories/${id}`);
  },

  // 创建分类（管理员）
  createCategory: (data: { name: string; slug: string; description?: string }) => {
    return request.post<Category>('/admin/categories', data);
  },

  // 更新分类（管理员）
  updateCategory: (id: number, data: { name?: string; slug?: string; description?: string }) => {
    return request.put<Category>(`/admin/categories/${id}`, data);
  },

  // 删除分类（管理员）
  deleteCategory: (id: number) => {
    return request.delete(`/admin/categories/${id}`);
  },
};
