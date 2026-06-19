import { request } from './client';
import type { PaginatedResponse, PaginationParams, Tag } from '@/types';
import { toPaginationQuery } from './pagination';

type TagFormPayload = {
  name: string;
  slug: string;
};

export const tagApi = {
  getTags: () => {
    return request.get<Tag[]>('/tags');
  },

  getAdminTags: (params: PaginationParams) => {
    return request.get<PaginatedResponse<Tag>>('/admin/tags', { params: toPaginationQuery(params) });
  },

  createTag: (data: TagFormPayload) => {
    return request.post<Tag>('/admin/tags', data);
  },

  updateTag: (id: number, data: TagFormPayload) => {
    return request.put<Tag>(`/admin/tags/${id}`, data);
  },

  deleteTag: (id: number) => {
    return request.delete(`/admin/tags/${id}`);
  },

  batchDeleteTags: (ids: number[]) => {
    return request.post<{ message: string; deleted_count: number }>('/admin/tags/batch-delete', { ids });
  },
};
