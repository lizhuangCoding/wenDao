import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Search } from 'lucide-react';
import { knowledgeDocumentApi } from '@/api/knowledgeDocument';
import { ConfirmModal, Loading, Pagination, EmptyState, ErrorState, BulkActionBar } from '@/components/common';
import { useUIStore } from '@/store';

type KnowledgeDocumentStatusFilter = 'pending_review' | 'approved' | 'rejected' | '';

export const KnowledgeDocumentList = () => {
  const pageSize = 10;
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState<KnowledgeDocumentStatusFilter>('');
  const [keyword, setKeyword] = useState('');
  const [keywordInput, setKeywordInput] = useState('');
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [confirmBatchDelete, setConfirmBatchDelete] = useState(false);
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();

  const {
    data: documentsData,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['admin-knowledge-documents', page, status, keyword],
    queryFn: () =>
      knowledgeDocumentApi.getKnowledgeDocuments({
        page,
        pageSize,
        status: status || undefined,
        keyword: keyword || undefined,
      }),
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
    onError: (err: any) => {
      showToast(err.message || '批量删除失败', 'error');
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

  const changeStatus = (value: KnowledgeDocumentStatusFilter) => {
    setStatus(value);
    setPage(1);
    setSelectedIds([]);
  };

  const applySearch = (event: React.FormEvent) => {
    event.preventDefault();
    setKeyword(keywordInput.trim());
    setPage(1);
    setSelectedIds([]);
  };

  const resetFilters = () => {
    setStatus('');
    setKeyword('');
    setKeywordInput('');
    setPage(1);
    setSelectedIds([]);
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <h1 className="text-3xl font-serif font-bold text-neutral-800 dark:text-neutral-100">知识文档审核</h1>
      </div>

      <div className="space-y-3 rounded-2xl border border-neutral-100 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
        <form onSubmit={applySearch} className="grid gap-3 md:grid-cols-[1fr_auto_auto]">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
            <input
              value={keywordInput}
              onChange={(event) => setKeywordInput(event.target.value)}
              placeholder="搜索标题、摘要或正文"
              className="w-full rounded-xl border border-neutral-200 bg-white py-2.5 pl-9 pr-3 text-sm text-neutral-800 outline-none transition-colors focus:border-primary-400 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100"
            />
          </div>
          <select
            value={status}
            onChange={(event) => changeStatus(event.target.value as KnowledgeDocumentStatusFilter)}
            className="rounded-xl border border-neutral-200 bg-white px-3 py-2.5 text-sm text-neutral-700 outline-none focus:border-primary-400 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100"
          >
            <option value="">全部状态</option>
            <option value="pending_review">待审核</option>
            <option value="approved">已通过</option>
            <option value="rejected">已拒绝</option>
          </select>
          <div className="flex gap-2">
            <button
              type="submit"
              className="rounded-xl bg-primary-600 px-4 py-2.5 text-sm font-bold text-white transition-colors hover:bg-primary-700"
            >
              搜索
            </button>
            <button
              type="button"
              onClick={resetFilters}
              className="rounded-xl bg-neutral-100 px-4 py-2.5 text-sm font-bold text-neutral-600 transition-colors hover:bg-neutral-200 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
            >
              重置
            </button>
          </div>
        </form>
        <BulkActionBar
          selectedCount={selectedIds.length}
          onDelete={() => setConfirmBatchDelete(true)}
          onClear={() => setSelectedIds([])}
          isDeleting={batchDeleteMutation.isPending}
          deleteLabel="删除知识文档"
        />
      </div>

      {isError ? (
        <ErrorState message={(error as any)?.message || '知识文档列表加载失败'} onRetry={() => refetch()} />
      ) : (
        <div className="overflow-hidden rounded-2xl border border-neutral-100 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
          <div className="overflow-x-auto">
            <table className="w-full border-collapse text-left">
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
                    <td className="px-6 py-4 text-sm text-neutral-500 dark:text-neutral-400">
                      {new Date(doc.created_at).toLocaleString()}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <Link to={`/admin/knowledge-documents/${doc.id}`} className="text-primary-600 hover:underline">
                        查看
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {documents.length === 0 && (
            <EmptyState title="暂无知识文档" description="当前筛选条件下没有知识文档。" className="m-6" />
          )}
        </div>
      )}

      {documentsData && (
        <Pagination
          page={page}
          totalPages={totalPages}
          onChange={(nextPage) => {
            setPage(nextPage);
            setSelectedIds([]);
          }}
        />
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
