import { request } from './client';
import type { User, PaginatedResponse, PaginationParams } from '@/types';
import { toPaginationQuery } from './pagination';

// 用户管理 API（管理员）
export const userApi = {
  // 获取用户列表
  listUsers: (params: PaginationParams & { role?: string; status?: string; search?: string }) => {
    return request.get<PaginatedResponse<User>>('/admin/users', {
      params: toPaginationQuery(params),
    });
  },

  // 更新用户角色
  updateUserRole: (id: number, role: string) => {
    return request.put(`/admin/users/${id}/role`, { role });
  },

  // 更新用户状态
  updateUserStatus: (id: number, status: string) => {
    return request.put(`/admin/users/${id}/status`, { status });
  },
};
