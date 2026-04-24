import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { knowledgeDocumentApi } from '@/api/knowledgeDocument';
import { useUIStore } from '@/store';

export const KnowledgeDocumentDetail = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const [reviewNote, setReviewNote] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['admin-knowledge-document', id],
    queryFn: () => knowledgeDocumentApi.getKnowledgeDocument(Number(id)),
    enabled: Boolean(id),
  });

  const approveMutation = useMutation({
    mutationFn: () => knowledgeDocumentApi.approveKnowledgeDocument(Number(id), reviewNote),
    onSuccess: () => {
      showToast('知识文档已通过审核', 'success');
      queryClient.invalidateQueries({ queryKey: ['admin-knowledge-documents'] });
      queryClient.invalidateQueries({ queryKey: ['admin-articles'] });
      queryClient.invalidateQueries({ queryKey: ['articles'] });
      navigate('/admin/knowledge-documents');
    },
  });

  const rejectMutation = useMutation({
    mutationFn: () => knowledgeDocumentApi.rejectKnowledgeDocument(Number(id), reviewNote),
    onSuccess: () => {
      showToast('知识文档已拒绝', 'success');
      queryClient.invalidateQueries({ queryKey: ['admin-knowledge-documents'] });
      navigate('/admin/knowledge-documents');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => knowledgeDocumentApi.deleteKnowledgeDocument(Number(id)),
    onSuccess: () => {
      showToast('知识文档已删除，对应文章已同步删除', 'success');
      queryClient.invalidateQueries({ queryKey: ['admin-knowledge-documents'] });
      queryClient.invalidateQueries({ queryKey: ['articles'] });
      navigate('/admin/knowledge-documents');
    },
  });

  if (isLoading) {
    return <div className="p-6 text-sm text-neutral-500">加载中...</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-neutral-700 dark:text-neutral-100">{data?.document.title}</h1>
          {data?.document.article_id && (
            <p className="mt-2 text-sm text-primary-600">已生成首页文章 #{data.document.article_id}</p>
          )}
        </div>
        <button
          onClick={() => {
            if (window.confirm('确定删除这篇知识文档吗？如果它已生成文章，对应文章也会被删除。')) {
              deleteMutation.mutate();
            }
          }}
          className="rounded-xl bg-red-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
          disabled={deleteMutation.isPending}
        >
          删除知识文档
        </button>
      </div>
      <div className="rounded-2xl border border-neutral-100 bg-white p-6 dark:border-neutral-800 dark:bg-neutral-900">
        <p className="mb-4 text-neutral-600 dark:text-neutral-300">{data?.document.summary}</p>
        <pre className="whitespace-pre-wrap text-sm text-neutral-700 dark:text-neutral-200">{data?.document.content}</pre>
      </div>
      <div className="rounded-2xl border border-neutral-100 bg-white p-6 dark:border-neutral-800 dark:bg-neutral-900">
        <h2 className="mb-4 text-lg font-semibold">来源</h2>
        <ul className="space-y-3">
          {data?.sources.map((source) => (
            <li key={source.id}>
              <a href={source.source_url} target="_blank" rel="noreferrer" className="text-primary-600 hover:underline">
                {source.source_title || source.source_url}
              </a>
              <p className="mt-1 text-sm text-neutral-500">{source.source_snippet}</p>
            </li>
          ))}
        </ul>
      </div>
      <textarea className="w-full rounded-xl border border-neutral-200 bg-white px-4 py-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" value={reviewNote} onChange={(e) => setReviewNote(e.target.value)} placeholder="审核备注" />
      <div className="flex gap-3">
        <button onClick={() => approveMutation.mutate()} className="rounded-xl bg-primary-600 px-4 py-2 text-sm font-semibold text-white" disabled={data?.document.status === 'approved'}>
          审核通过
        </button>
        <button onClick={() => rejectMutation.mutate()} className="rounded-xl bg-neutral-200 px-4 py-2 text-sm font-semibold text-neutral-800 dark:bg-neutral-700 dark:text-neutral-100" disabled={data?.document.status === 'approved'}>
          拒绝
        </button>
      </div>
    </div>
  );
};
