import type { AIObservabilityRun, PaginatedResponse, PaginationParams } from '@/types';
import { request } from './client';
import { toPaginationQuery } from './pagination';

export const aiObservabilityApi = {
  listRuns: (params: PaginationParams & { status?: string; keyword?: string }) => {
    return request.get<PaginatedResponse<AIObservabilityRun>>('/admin/ai-observability/runs', {
      params: toPaginationQuery(params),
    });
  },

  batchDeleteRuns: (ids: number[]) => {
    return request.post<{ message: string; deleted_count: number }>('/admin/ai-observability/runs/batch-delete', { ids });
  },
};
