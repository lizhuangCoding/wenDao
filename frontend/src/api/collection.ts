import { request } from './client';
import type { Collection, PaginatedResponse, PaginationParams } from '@/types';
import { toPaginationQuery } from './pagination';

type CollectionFormPayload = {
  name: string;
  slug: string;
  description?: string;
  sort_order: number;
  status: 'active' | 'hidden';
};

export const collectionApi = {
  getCollections: () => {
    return request.get<Collection[]>('/collections');
  },

  getAdminCollections: (params: PaginationParams) => {
    return request.get<PaginatedResponse<Collection>>('/admin/collections', {
      params: toPaginationQuery(params),
    });
  },

  createCollection: (data: CollectionFormPayload) => {
    return request.post<Collection>('/admin/collections', data);
  },

  updateCollection: (id: number, data: CollectionFormPayload) => {
    return request.put<Collection>(`/admin/collections/${id}`, data);
  },

  deleteCollection: (id: number) => {
    return request.delete(`/admin/collections/${id}`);
  },

  batchDeleteCollections: (ids: number[]) => {
    return request.post<{ message: string; deleted_count: number }>('/admin/collections/batch-delete', { ids });
  },
};
