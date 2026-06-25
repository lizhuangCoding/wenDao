import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { knowledgeDocumentApi } from '@/api/knowledgeDocument';
import {
  Button,
  ConfirmModal,
  ErrorState,
  Loading,
  PageHeader,
  Panel,
  StatusBadge,
  TextArea,
} from '@/components/common';
import { useUIStore } from '@/store';

const getKnowledgeDocumentStatusMeta = (
  status: string
): { label: string; variant: 'success' | 'warning' | 'danger' | 'neutral' } => {
  if (status === 'approved') {
    return { label: '已通过', variant: 'success' };
  }
  if (status === 'rejected') {
    return { label: '已拒绝', variant: 'danger' };
  }
  if (status === 'pending_review') {
    return { label: '待审核', variant: 'warning' };
  }
  return { label: status, variant: 'neutral' };
};

export const KnowledgeDocumentDetail = () => {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const [reviewNote, setReviewNote] = useState('');
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const documentId = Number(id);
  const hasValidDocumentId = Boolean(id) && Number.isFinite(documentId);

  const {
    data,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['admin-knowledge-document', id],
    queryFn: () => knowledgeDocumentApi.getKnowledgeDocument(documentId),
    enabled: hasValidDocumentId,
  });

  const approveMutation = useMutation({
    mutationFn: () => knowledgeDocumentApi.approveKnowledgeDocument(documentId, reviewNote),
    onSuccess: () => {
      showToast(t('knowledgeDocument.approved'), 'success');
      queryClient.invalidateQueries({ queryKey: ['admin-knowledge-documents'] });
      queryClient.invalidateQueries({ queryKey: ['admin-articles'] });
      queryClient.invalidateQueries({ queryKey: ['articles'] });
      navigate('/admin/knowledge-documents');
    },
    onError: (err: any) => {
      showToast(err.message || t('knowledgeDocument.rejectFailed'), 'error');
    },
  });

  const rejectMutation = useMutation({
    mutationFn: () => knowledgeDocumentApi.rejectKnowledgeDocument(documentId, reviewNote),
    onSuccess: () => {
      showToast(t('knowledgeDocument.rejected'), 'success');
      queryClient.invalidateQueries({ queryKey: ['admin-knowledge-documents'] });
      navigate('/admin/knowledge-documents');
    },
    onError: (err: any) => {
      showToast(err.message || t('knowledgeDocument.rejectFailed'), 'error');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => knowledgeDocumentApi.deleteKnowledgeDocument(documentId),
    onSuccess: () => {
      showToast(t('knowledgeDocument.deleteSuccess'), 'success');
      queryClient.invalidateQueries({ queryKey: ['admin-knowledge-documents'] });
      queryClient.invalidateQueries({ queryKey: ['articles'] });
      navigate('/admin/knowledge-documents');
    },
    onError: (err: any) => {
      showToast(err.message || t('common.failed'), 'error');
    },
  });

  const isReviewing = approveMutation.isPending || rejectMutation.isPending;

  if (!hasValidDocumentId) {
    return <ErrorState title={t('knowledgeDocument.documentNotFound')} message={t('knowledgeDocument.invalidDocumentId')} />;
  }

  if (isLoading) {
    return <Loading />;
  }

  if (isError || !data) {
    return (
      <ErrorState
        title={t('knowledgeDocument.loadFailed')}
        message={(error as any)?.message || t('knowledgeDocument.loadFailedDescription')}
        onRetry={() => refetch()}
      />
    );
  }

  const statusMeta = getKnowledgeDocumentStatusMeta(data.document.status);

  return (
    <div className="space-y-6">
      <PageHeader
        title={
          <span className="inline-flex flex-wrap items-center gap-3">
            {data.document.title}
            <StatusBadge variant={statusMeta.variant}>{statusMeta.label}</StatusBadge>
          </span>
        }
        description={data.document.article_id ? t('knowledgeDocument.generatedArticle', { id: data.document.article_id }) : undefined}
        tone="admin"
        actions={
          <Button
            variant="danger"
            onClick={() => setIsDeleteConfirmOpen(true)}
            disabled={deleteMutation.isPending}
          >
            {deleteMutation.isPending ? t('common.loading') : t('knowledgeDocument.delete')}
          </Button>
        }
      />
      <Panel padding="lg">
        <p className="mb-4 text-neutral-600 dark:text-neutral-300">{data?.document.summary}</p>
        <pre className="whitespace-pre-wrap text-sm text-neutral-700 dark:text-neutral-200">{data?.document.content}</pre>
      </Panel>
      <Panel padding="lg">
        <h2 className="mb-4 text-lg font-semibold">{t('knowledgeDocument.source')}</h2>
        <ul className="space-y-3">
          {data?.sources.map((source) => (
            <li key={source.id}>
              <a
                href={source.source_url}
                target="_blank"
                rel="noreferrer"
                className="font-medium text-primary-600 hover:underline dark:text-primary-400"
              >
                {source.source_title || source.source_url}
              </a>
              <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">{source.source_snippet}</p>
            </li>
          ))}
        </ul>
      </Panel>
        <TextArea
          value={reviewNote}
          onChange={(e) => setReviewNote(e.target.value)}
        placeholder={t('knowledgeDocument.reviewNote')}
        />
      <div className="flex gap-3">
        <Button
          onClick={() => approveMutation.mutate()}
          disabled={data.document.status === 'approved' || isReviewing}
        >
          {approveMutation.isPending ? t('knowledgeDocument.reviewing') : t('knowledgeDocument.approve')}
        </Button>
        <Button
          variant="secondary"
          onClick={() => rejectMutation.mutate()}
          disabled={data.document.status === 'approved' || isReviewing}
        >
          {rejectMutation.isPending ? t('common.sending') : t('knowledgeDocument.rejecting')}
        </Button>
      </div>

      <ConfirmModal
        isOpen={isDeleteConfirmOpen}
        title={t('knowledgeDocument.deleteConfirmTitle')}
        message={t('knowledgeDocument.deleteConfirmMessage')}
        confirmText={t('common.delete')}
        onConfirm={() => deleteMutation.mutate()}
        onCancel={() => setIsDeleteConfirmOpen(false)}
        isConfirming={deleteMutation.isPending}
        isDanger
      />
    </div>
  );
};
