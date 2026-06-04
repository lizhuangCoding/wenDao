import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
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
      showToast('知识文档已通过审核', 'success');
      queryClient.invalidateQueries({ queryKey: ['admin-knowledge-documents'] });
      queryClient.invalidateQueries({ queryKey: ['admin-articles'] });
      queryClient.invalidateQueries({ queryKey: ['articles'] });
      navigate('/admin/knowledge-documents');
    },
    onError: (err: any) => {
      showToast(err.message || '审核通过失败，请重试', 'error');
    },
  });

  const rejectMutation = useMutation({
    mutationFn: () => knowledgeDocumentApi.rejectKnowledgeDocument(documentId, reviewNote),
    onSuccess: () => {
      showToast('知识文档已拒绝', 'success');
      queryClient.invalidateQueries({ queryKey: ['admin-knowledge-documents'] });
      navigate('/admin/knowledge-documents');
    },
    onError: (err: any) => {
      showToast(err.message || '拒绝失败，请重试', 'error');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => knowledgeDocumentApi.deleteKnowledgeDocument(documentId),
    onSuccess: () => {
      showToast('知识文档已删除，对应文章已同步删除', 'success');
      queryClient.invalidateQueries({ queryKey: ['admin-knowledge-documents'] });
      queryClient.invalidateQueries({ queryKey: ['articles'] });
      navigate('/admin/knowledge-documents');
    },
    onError: (err: any) => {
      showToast(err.message || '删除失败，请重试', 'error');
    },
  });

  const isReviewing = approveMutation.isPending || rejectMutation.isPending;

  if (!hasValidDocumentId) {
    return <ErrorState title="文档不存在" message="知识文档 ID 无效，请返回列表重新选择。" />;
  }

  if (isLoading) {
    return <Loading />;
  }

  if (isError || !data) {
    return (
      <ErrorState
        title="知识文档加载失败"
        message={(error as any)?.message || '无法加载知识文档详情，请稍后重试。'}
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
        description={data.document.article_id ? `已生成首页文章 #${data.document.article_id}` : undefined}
        actions={
          <Button
            variant="danger"
            onClick={() => setIsDeleteConfirmOpen(true)}
            disabled={deleteMutation.isPending}
          >
            {deleteMutation.isPending ? '删除中...' : '删除知识文档'}
          </Button>
        }
      />
      <Panel padding="lg">
        <p className="mb-4 text-neutral-600 dark:text-neutral-300">{data?.document.summary}</p>
        <pre className="whitespace-pre-wrap text-sm text-neutral-700 dark:text-neutral-200">{data?.document.content}</pre>
      </Panel>
      <Panel padding="lg">
        <h2 className="mb-4 text-lg font-semibold">来源</h2>
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
        placeholder="审核备注"
      />
      <div className="flex gap-3">
        <Button
          onClick={() => approveMutation.mutate()}
          disabled={data.document.status === 'approved' || isReviewing}
        >
          {approveMutation.isPending ? '审核中...' : '审核通过'}
        </Button>
        <Button
          variant="secondary"
          onClick={() => rejectMutation.mutate()}
          disabled={data.document.status === 'approved' || isReviewing}
        >
          {rejectMutation.isPending ? '提交中...' : '拒绝'}
        </Button>
      </div>

      <ConfirmModal
        isOpen={isDeleteConfirmOpen}
        title="删除知识文档"
        message="确定删除这篇知识文档吗？如果它已生成文章，对应文章也会被删除。"
        confirmText="删除"
        onConfirm={() => deleteMutation.mutate()}
        onCancel={() => setIsDeleteConfirmOpen(false)}
        isConfirming={deleteMutation.isPending}
        isDanger
      />
    </div>
  );
};
