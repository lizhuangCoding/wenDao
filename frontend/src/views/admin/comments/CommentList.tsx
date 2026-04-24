import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { commentApi } from '@/api';
import { Loading, ConfirmModal } from '@/components/common';
import { formatDate } from '@/utils';
import { useUIStore } from '@/store';
import { motion } from 'framer-motion';

export const CommentList = () => {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
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

  const { data: commentsData, isLoading } = useQuery({
    queryKey: ['admin-comments', page],
    queryFn: () => commentApi.getAdminComments({ page, pageSize: 15 }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => commentApi.adminDeleteComment(id),
    onSuccess: () => {
      showToast('评论已删除', 'success');
      queryClient.invalidateQueries({ queryKey: ['admin-comments'] });
    },
    onError: (error: any) => {
      showToast(error.message || '删除失败', 'error');
    },
  });

  const restoreMutation = useMutation({
    mutationFn: (id: number) => commentApi.adminRestoreComment(id),
    onSuccess: () => {
      showToast('评论已恢复', 'success');
      queryClient.invalidateQueries({ queryKey: ['admin-comments'] });
    },
    onError: (error: any) => {
      showToast(error.message || '恢复失败', 'error');
    },
  });

  if (isLoading) return <Loading />;

  return (
    <div className="space-y-6">
      {/* 标题 */}
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex justify-between items-center"
      >
        <h1 className="text-3xl font-serif font-bold text-neutral-800 dark:text-neutral-100">
          {t('admin.commentManagement')}
        </h1>
      </motion.div>

      {/* 评论表格 */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.1 }}
        className="bg-white dark:bg-neutral-900 rounded-2xl shadow-sm border border-neutral-100 dark:border-neutral-800 overflow-hidden"
      >
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="bg-neutral-50 dark:bg-neutral-800/50 border-b border-neutral-100 dark:border-neutral-800">
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.commentContent')}</th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.author')}</th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.article')}</th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.status')}</th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.createdAt')}</th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400 text-right">{t('admin.actions')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-neutral-100 dark:divide-neutral-800">
            {commentsData?.data?.map((comment, index) => (
              <motion.tr
                key={comment.id}
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ delay: index * 0.03 }}
                className="hover:bg-neutral-50 dark:hover:bg-neutral-800/50 transition-colors"
              >
                <td className="px-6 py-4">
                  <div className="text-sm text-neutral-700 dark:text-neutral-300 max-w-md line-clamp-2">
                    {comment.content}
                  </div>
                </td>
                <td className="px-6 py-4 text-sm text-neutral-500 dark:text-neutral-400">
                  {comment.user?.username}
                </td>
                <td className="px-6 py-4 text-sm text-neutral-500 dark:text-neutral-400">
                  <div className="max-w-xs truncate">{comment.article?.title}</div>
                </td>
                <td className="px-6 py-4">
                  <span
                    className={`px-3 py-1.5 rounded-full text-xs font-medium ${
                      comment.status === 'normal'
                        ? 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400'
                        : 'bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400'
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
                      onClick={() => setConfirmConfig({ isOpen: true, id: comment.id, type: 'delete' })}
                      className="px-3 py-1.5 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/30 rounded-lg transition-colors"
                    >
                      {t('admin.delete')}
                    </button>
                  ) : (
                    <button
                      onClick={() => setConfirmConfig({ isOpen: true, id: comment.id, type: 'restore' })}
                      className="px-3 py-1.5 text-sm text-green-600 dark:text-green-400 hover:bg-green-50 dark:hover:bg-green-900/30 rounded-lg transition-colors"
                    >
                      {t('admin.restore')}
                    </button>
                  )}
                </td>
              </motion.tr>
            ))}
          </tbody>
        </table>
      </motion.div>

      {/* 分页 */}
      {commentsData && commentsData.totalPages > 1 && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.2 }}
          className="flex justify-center items-center gap-3"
        >
          <button
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page === 1}
            className="flex items-center gap-2 px-4 py-2 bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 rounded-xl font-medium hover:bg-neutral-200 dark:hover:bg-neutral-700 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
            {t('admin.previous')}
          </button>
          <span className="px-4 py-2 text-neutral-600 dark:text-neutral-400 font-medium">
            {page} / {commentsData.totalPages}
          </span>
          <button
            onClick={() => setPage((p) => Math.min(commentsData.totalPages, p + 1))}
            disabled={page === commentsData.totalPages}
            className="flex items-center gap-2 px-4 py-2 bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 rounded-xl font-medium hover:bg-neutral-200 dark:hover:bg-neutral-700 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
          >
            {t('admin.next')}
            <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </button>
        </motion.div>
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
          setConfirmConfig({ ...confirmConfig, isOpen: false });
        }}
        onCancel={() => setConfirmConfig({ ...confirmConfig, isOpen: false })}
        isDanger={confirmConfig.type === 'delete'}
      />
    </div>
  );
};