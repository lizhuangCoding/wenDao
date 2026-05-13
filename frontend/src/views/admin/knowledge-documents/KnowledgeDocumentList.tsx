import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { knowledgeDocumentApi } from '@/api/knowledgeDocument';
import { ConfirmModal, Loading } from '@/components/common';
import { useUIStore } from '@/store';

export const KnowledgeDocumentList = () => {
  const pageSize = 10;
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState<'pending_review' | 'approved' | 'rejected' | ''>('');
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [confirmBatchDelete, setConfirmBatchDelete] = useState(false);
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();

  const { data: documentsData, isLoading } = useQuery({
    queryKey: ['admin-knowledge-documents', page, status],
    queryFn: () => knowledgeDocumentApi.getKnowledgeDocuments({ page, pageSize, status: status || undefined }),
  });

  const documents = documentsData?.data ?? [];
  const totalPages = Math.max(1, documentsData?.totalPages ?? 1);
  const currentPageIds = documents.map((doc) => doc.id);
  const allCurrentPageSelected =
    currentPageIds.length > 0 && currentPageIds.every((id) => selectedIds.includes(id));

  const batchDeleteMutation = useMutation({
    mutationFn: (ids: number[]) => knowledgeDocumentApi.batchDeleteKnowledgeDocuments(ids),
    onSuccess: (result) => {
      showToast(`已删除 ${result.deleted_count} 个知识文档`, 'success');
      setSelectedIds([]);
      setConfirmBatchDelete(false);
      queryClient.invalidateQueries({ queryKey: ['admin-knowledge-documents'] });
      if (page > 1 && documents.length > 0 && selectedIds.length >= documents.length) {
        setPage((currentPage) => Math.max(1, currentPage - 1));
      }
    },
    onError: (error: any) => {
      showToast(error.message || '批量删除失败', 'error');
    },
  });

  if (isLoading) {
    return <Loading />;
  }

  const toggleDocumentSelection = (id: number) => {
    setSelectedIds((current) =>
      current.includes(id) ? current.filter((selectedId) => selectedId !== id) : [...current, id]
    );
  };

  const toggleCurrentPageSelection = () => {
    setSelectedIds((current) => {
      if (allCurrentPageSelected) {
        return current.filter((id) => !currentPageIds.includes(id));
      }
      return Array.from(new Set([...current, ...currentPageIds]));
    });
  };

  const changeStatus = (value: typeof status) => {
    setStatus(value);
    setPage(1);
    setSelectedIds([]);
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <h1 className="text-3xl font-serif font-bold text-neutral-800 dark:text-neutral-100">知识文档审核</h1>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
          {selectedIds.length > 0 && (
            <button
              type="button"
              onClick={() => setConfirmBatchDelete(true)}
              disabled={batchDeleteMutation.isPending}
              className="rounded-xl bg-red-500 px-4 py-2 text-xs font-bold text-white transition-colors hover:bg-red-600 disabled:cursor-not-allowed disabled:opacity-60"
            >
              批量删除（{selectedIds.length}）
            </button>
          )}
          <div className="flex flex-wrap gap-2">
            {[
              ['', '全部'],
              ['pending_review', '待审核'],
              ['approved', '已通过'],
              ['rejected', '已拒绝'],
            ].map(([value, label]) => (
              <button
                key={value}
                type="button"
                onClick={() => changeStatus(value as typeof status)}
                className={`rounded-xl px-3 py-2 text-xs font-bold ${
                  status === value
                    ? 'bg-primary-600 text-white'
                    : 'bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300'
                }`}
              >
                {label}
              </button>
            ))}
          </div>
        </div>
      </div>
      <div className="overflow-hidden rounded-2xl border border-neutral-100 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
        <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="border-b border-neutral-100 bg-neutral-50 dark:border-neutral-800 dark:bg-neutral-800/50">
              <th className="px-6 py-4">
                <input
                  type="checkbox"
                  checked={allCurrentPageSelected}
                  onChange={toggleCurrentPageSelection}
                  className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                  aria-label="选择当前页知识文档"
                />
              </th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">标题</th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">状态</th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">创建时间</th>
              <th className="px-6 py-4 text-right text-sm font-semibold text-neutral-600 dark:text-neutral-400">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-neutral-100 dark:divide-neutral-800">
            {documents.map((doc) => (
              <tr key={doc.id} className="transition-colors hover:bg-neutral-50 dark:hover:bg-neutral-800/50">
                <td className="px-6 py-4">
                  <input
                    type="checkbox"
                    checked={selectedIds.includes(doc.id)}
                    onChange={() => toggleDocumentSelection(doc.id)}
                    className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                    aria-label={`选择知识文档 ${doc.title}`}
                  />
                </td>
                <td className="px-6 py-4 font-medium text-neutral-800 dark:text-neutral-200">{doc.title}</td>
                <td className="px-6 py-4">
                  <div className="flex flex-col gap-1">
                    <span className="text-sm text-neutral-600 dark:text-neutral-300">{doc.status}</span>
                    {doc.article_id && <span className="text-xs text-primary-600">已生成文章 #{doc.article_id}</span>}
                  </div>
                </td>
                <td className="px-6 py-4 text-sm text-neutral-500 dark:text-neutral-400">{new Date(doc.created_at).toLocaleString()}</td>
                <td className="px-6 py-4 text-right">
                  <Link to={`/admin/knowledge-documents/${doc.id}`} className="text-primary-600 hover:underline">
                    查看
                  </Link>
                </td>
              </tr>
            ))}
            {documents.length === 0 && (
              <tr>
                <td colSpan={5} className="px-6 py-12 text-center text-sm text-neutral-500 dark:text-neutral-400">
                  暂无知识文档
                </td>
              </tr>
            )}
          </tbody>
        </table>
        </div>
      </div>

      {documentsData && (
        <div className="flex items-center justify-center gap-3">
          <button
            type="button"
            onClick={() => {
              setPage((currentPage) => Math.max(1, currentPage - 1));
              setSelectedIds([]);
            }}
            disabled={page === 1}
            className="rounded-xl bg-neutral-100 px-4 py-2 text-sm font-medium text-neutral-700 transition-all hover:bg-neutral-200 disabled:cursor-not-allowed disabled:opacity-40 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
          >
            上一页
          </button>
          <span className="px-4 py-2 text-sm font-medium text-neutral-600 dark:text-neutral-400">
            {page} / {totalPages}
          </span>
          <button
            type="button"
            onClick={() => {
              setPage((currentPage) => Math.min(totalPages, currentPage + 1));
              setSelectedIds([]);
            }}
            disabled={page === totalPages}
            className="rounded-xl bg-neutral-100 px-4 py-2 text-sm font-medium text-neutral-700 transition-all hover:bg-neutral-200 disabled:cursor-not-allowed disabled:opacity-40 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
          >
            下一页
          </button>
        </div>
      )}

      <ConfirmModal
        isOpen={confirmBatchDelete}
        title="批量删除知识文档"
        message={`确定删除选中的 ${selectedIds.length} 个知识文档吗？已通过审核生成的文章也会同步删除。`}
        confirmText="删除"
        onConfirm={() => batchDeleteMutation.mutate(selectedIds)}
        onCancel={() => setConfirmBatchDelete(false)}
        isDanger
      />
    </div>
  );
};
