import { request } from './client';
import type { KnowledgeDocument, KnowledgeDocumentSource, PaginatedResponse } from '@/types';

export const knowledgeDocumentApi = {
  getKnowledgeDocuments: (params: { page: number; pageSize: number; status?: string; keyword?: string }) =>
    request.get<PaginatedResponse<KnowledgeDocument>>('/admin/knowledge-documents', { params }),
  getKnowledgeDocument: (id: number) =>
    request.get<{ document: KnowledgeDocument; sources: KnowledgeDocumentSource[] }>(`/admin/knowledge-documents/${id}`),
  approveKnowledgeDocument: (id: number, review_note: string) =>
    request.post<KnowledgeDocument>(`/admin/knowledge-documents/${id}/approve`, { review_note }),
  rejectKnowledgeDocument: (id: number, review_note: string) =>
    request.post<KnowledgeDocument>(`/admin/knowledge-documents/${id}/reject`, { review_note }),
  deleteKnowledgeDocument: (id: number) =>
    request.delete(`/admin/knowledge-documents/${id}`),
};
