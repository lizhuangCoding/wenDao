export interface KnowledgeDocument {
  id: number;
  title: string;
  summary: string;
  content: string;
  status: 'pending_review' | 'approved' | 'rejected';
  source_type: 'research' | 'manual';
  created_by_user_id: number;
  reviewed_by_user_id?: number;
  reviewed_at?: string;
  review_note: string;
  article_id?: number;
  vectorized_at?: string;
  created_at: string;
  updated_at: string;
}

export interface KnowledgeDocumentSource {
  id: number;
  knowledge_document_id: number;
  source_url: string;
  source_title: string;
  source_domain: string;
  source_snippet: string;
  sort_order: number;
}
