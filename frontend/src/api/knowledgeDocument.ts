import { request } from './client';
import type { KnowledgeDocument, KnowledgeDocumentSource, PaginatedResponse } from '@/types';
import { toPaginationQuery } from './pagination';

export const knowledgeDocumentApi = {
  getKnowledgeDocuments: (params: { page: number; pageSize: number; status?: string; keyword?: string }) =>
    request.get<PaginatedResponse<KnowledgeDocument>>('/admin/knowledge-documents', { params: toPaginationQuery(params) }),
  getKnowledgeDocument: (id: number) =>
    request.get<{ document: KnowledgeDocument; sources: KnowledgeDocumentSource[] }>(`/admin/knowledge-documents/${id}`),
  approveKnowledgeDocument: (id: number, review_note: string) =>
    request.post<KnowledgeDocument>(`/admin/knowledge-documents/${id}/approve`, { review_note }),
  rejectKnowledgeDocument: (id: number, review_note: string) =>
    request.post<KnowledgeDocument>(`/admin/knowledge-documents/${id}/reject`, { review_note }),
  deleteKnowledgeDocument: (id: number) =>
    request.delete(`/admin/knowledge-documents/${id}`),
  batchDeleteKnowledgeDocuments: (ids: number[]) =>
    request.post<{ message: string; deleted_count: number }>('/admin/knowledge-documents/batch-delete', { ids }),
};
