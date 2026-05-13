import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Search } from 'lucide-react';
import { commentApi } from '@/api';
import { Loading, ConfirmModal, Pagination, EmptyState, ErrorState, BulkActionBar } from '@/components/common';
import { formatDate } from '@/utils';
import { useUIStore } from '@/store';
import { motion } from 'framer-motion';

type CommentStatusFilter = '' | 'normal' | 'deleted';

export const CommentList = () => {
  const { t } = useTranslation();
  const pageSize = 15;
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState<CommentStatusFilter>('');
  const [keyword, setKeyword] = useState('');
  const [keywordInput, setKeywordInput] = useState('');
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const [confirmConfig, setConfirmConfig] = useState<{
    isOpen: boolean;
    id: number | null;
    type: 'delete' | 'restore';
  }>({
    isOpen: false,
    id: null,
    type: 'delete',
  });
  const [confirmBatchDelete, setConfirmBatchDelete] = useState(false);

  const {
    data: commentsData,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['admin-comments', page, status, keyword],
    queryFn: () =>
      commentApi.getAdminComments({
        page,
        pageSize,
        status: status || undefined,
        keyword: keyword || undefined,
      }),
  });
  const comments = commentsData?.data ?? [];
  const totalPages = Math.max(1, commentsData?.totalPages ?? 1);
  const currentPageIds = comments.map((comment) => comment.id);
  const allCurrentPageSelected =
    currentPageIds.length > 0 && currentPageIds.every((id) => selectedIds.includes(id));

  const invalidateComments = () => {
    queryClient.invalidateQueries({ queryKey: ['admin-comments'] });
  };

  const deleteMutation = useMutation({
    mutationFn: (id: number) => commentApi.adminDeleteComment(id),
    onSuccess: () => {
      showToast('评论已删除', 'success');
      setConfirmConfig({ isOpen: false, id: null, type: 'delete' });
      invalidateComments();
    },
    onError: (err: any) => {
      showToast(err.message || '删除失败', 'error');
    },
  });

  const batchDeleteMutation = useMutation({
    mutationFn: (ids: number[]) => commentApi.batchDeleteComments(ids),
    onSuccess: (result) => {
      showToast(`已删除 ${result.deleted_count} 条评论`, 'success');
      setSelectedIds([]);
      setConfirmBatchDelete(false);
      invalidateComments();
      if (page > 1 && comments.length > 0 && selectedIds.length >= comments.length) {
        setPage((currentPage) => Math.max(1, currentPage - 1));
      }
    },
    onError: (err: any) => {
      showToast(err.message || '批量删除失败', 'error');
    },
  });

  const restoreMutation = useMutation({
    mutationFn: (id: number) => commentApi.adminRestoreComment(id),
    onSuccess: () => {
      showToast('评论已恢复', 'success');
      setConfirmConfig({ isOpen: false, id: null, type: 'restore' });
      invalidateComments();
    },
    onError: (err: any) => {
      showToast(err.message || '恢复失败', 'error');
    },
  });

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

  const toggleCommentSelection = (id: number) => {
    setSelectedIds((ids) => (ids.includes(id) ? ids.filter((item) => item !== id) : [...ids, id]));
  };

  const toggleCurrentPageSelection = () => {
    setSelectedIds((ids) => {
      if (allCurrentPageSelected) {
        return ids.filter((id) => !currentPageIds.includes(id));
      }
      return Array.from(new Set([...ids, ...currentPageIds]));
    });
  };

  if (isLoading) return <Loading />;

  return (
    <div className="space-y-6">
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex items-center justify-between"
      >
        <h1 className="text-3xl font-serif font-bold text-neutral-800 dark:text-neutral-100">
          {t('admin.commentManagement')}
        </h1>
      </motion.div>

      <div className="space-y-3 rounded-2xl border border-neutral-100 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
        <form onSubmit={applySearch} className="grid gap-3 md:grid-cols-[1fr_auto_auto]">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
            <input
              value={keywordInput}
              onChange={(event) => setKeywordInput(event.target.value)}
              placeholder="搜索评论内容、作者或文章标题"
              className="w-full rounded-xl border border-neutral-200 bg-white py-2.5 pl-9 pr-3 text-sm text-neutral-800 outline-none transition-colors focus:border-primary-400 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100"
            />
          </div>
          <select
            value={status}
            onChange={(event) => {
              setStatus(event.target.value as CommentStatusFilter);
              setPage(1);
              setSelectedIds([]);
            }}
            className="rounded-xl border border-neutral-200 bg-white px-3 py-2.5 text-sm text-neutral-700 outline-none focus:border-primary-400 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100"
          >
            <option value="">全部状态</option>
            <option value="normal">正常</option>
            <option value="deleted">已删除</option>
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
          deleteLabel="删除评论"
        />
      </div>

      {isError ? (
        <ErrorState message={(error as any)?.message || '评论列表加载失败'} onRetry={() => refetch()} />
      ) : (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="overflow-hidden rounded-2xl border border-neutral-100 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900"
        >
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
                      aria-label="选择当前页评论"
                    />
                  </th>
                  <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.commentContent')}</th>
                  <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.author')}</th>
                  <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.article')}</th>
                  <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.status')}</th>
                  <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.createdAt')}</th>
                  <th className="px-6 py-4 text-right text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100 dark:divide-neutral-800">
                {comments.map((comment, index) => (
                  <motion.tr
                    key={comment.id}
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    transition={{ delay: index * 0.03 }}
                    className="transition-colors hover:bg-neutral-50 dark:hover:bg-neutral-800/50"
                  >
                    <td className="px-6 py-4">
                      <input
                        type="checkbox"
                        checked={selectedIds.includes(comment.id)}
                        onChange={() => toggleCommentSelection(comment.id)}
                        className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                        aria-label={`选择评论 ${comment.id}`}
                      />
                    </td>
                    <td className="px-6 py-4">
                      <div className="max-w-md line-clamp-2 text-sm text-neutral-700 dark:text-neutral-300">
                        {comment.content}
                      </div>
                    </td>
                    <td className="px-6 py-4 text-sm text-neutral-500 dark:text-neutral-400">
                      {comment.user?.username || '-'}
                    </td>
                    <td className="px-6 py-4 text-sm text-neutral-500 dark:text-neutral-400">
                      <div className="max-w-xs truncate">{comment.article?.title || '-'}</div>
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={`rounded-full px-3 py-1.5 text-xs font-medium ${
                          comment.status === 'normal'
                            ? 'bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400'
                            : 'bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400'
                        }`}
                      >
                        {comment.status === 'normal' ? t('admin.normal') : t('admin.deleted')}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-neutral-500 dark:text-neutral-400">
                      {formatDate(comment.created_at)}
                    </td>
                    <td className="px-6 py-4 text-right">
                      {comment.status === 'normal' ? (
                        <button
                          type="button"
                          onClick={() => setConfirmConfig({ isOpen: true, id: comment.id, type: 'delete' })}
                          className="rounded-lg px-3 py-1.5 text-sm text-red-600 transition-colors hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/30"
                        >
                          {t('admin.delete')}
                        </button>
                      ) : (
                        <button
                          type="button"
                          onClick={() => setConfirmConfig({ isOpen: true, id: comment.id, type: 'restore' })}
                          className="rounded-lg px-3 py-1.5 text-sm text-green-600 transition-colors hover:bg-green-50 dark:text-green-400 dark:hover:bg-green-900/30"
                        >
                          {t('admin.restore')}
                        </button>
                      )}
                    </td>
                  </motion.tr>
                ))}
              </tbody>
            </table>
          </div>
          {comments.length === 0 && (
            <EmptyState title="暂无评论" description="当前筛选条件下没有评论。" className="m-6" />
          )}
        </motion.div>
      )}

      {commentsData && (
        <Pagination
          page={page}
          totalPages={totalPages}
          onChange={(nextPage) => {
            setPage(nextPage);
            setSelectedIds([]);
          }}
          previousLabel={t('admin.previous')}
          nextLabel={t('admin.next')}
        />
      )}

      <ConfirmModal
        isOpen={confirmConfig.isOpen}
        title={confirmConfig.type === 'delete' ? t('admin.delete') : t('admin.restore')}
        message={
          confirmConfig.type === 'delete'
            ? t('admin.confirmDeleteComment')
            : t('admin.confirmRestoreComment')
        }
        onConfirm={() => {
          if (confirmConfig.id) {
            if (confirmConfig.type === 'delete') {
              deleteMutation.mutate(confirmConfig.id);
            } else {
              restoreMutation.mutate(confirmConfig.id);
            }
          }
        }}
        onCancel={() => setConfirmConfig({ ...confirmConfig, isOpen: false })}
        isDanger={confirmConfig.type === 'delete'}
      />

      <ConfirmModal
        isOpen={confirmBatchDelete}
        title="批量删除评论"
        message={`确定删除选中的 ${selectedIds.length} 条评论吗？该操作会将评论标记为已删除。`}
        confirmText="删除"
        onConfirm={() => batchDeleteMutation.mutate(selectedIds)}
        onCancel={() => setConfirmBatchDelete(false)}
        isDanger
      />
    </div>
  );
};
