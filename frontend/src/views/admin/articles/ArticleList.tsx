import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { articleApi, siteApi } from '@/api';
import { Loading, ConfirmModal } from '@/components/common';
import { formatDate } from '@/utils';
import { useUIStore } from '@/store';
import { motion } from 'framer-motion';

export const ArticleList = () => {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const [deleteId, setDeleteId] = useState<number | null>(null);

  const { data: sortMode } = useQuery({
    queryKey: ['site-sort-mode'],
    queryFn: () => siteApi.getSortMode(),
  });

  const { data: articlesData, isLoading } = useQuery({
    queryKey: ['admin-articles', page, sortMode?.enabled],
    queryFn: () => articleApi.getAdminArticles({ 
      page, 
      pageSize: 10,
      sort_by_popularity: sortMode?.enabled
    }),
  });

  const sortModeMutation = useMutation({
    mutationFn: (enabled: boolean) => siteApi.setSortMode(enabled),
    onSuccess: () => {
      showToast('排序模式已更新', 'success');
      queryClient.invalidateQueries({ queryKey: ['site-sort-mode'] });
      queryClient.invalidateQueries({ queryKey: ['admin-articles'] });
    },
    onError: (error: any) => {
      showToast(error.message || '切换失败，请重试', 'error');
    }
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => articleApi.deleteArticle(id),
    onSuccess: () => {
      showToast(t('admin.delete') + t('admin.success') || '文章已删除', 'success');
      queryClient.invalidateQueries({ queryKey: ['admin-articles'] });
    },
  });

  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: number; status: string }) =>
      status === 'published' ? articleApi.draftArticle(id) : articleApi.publishArticle(id),
    onSuccess: () => {
      showToast(t('admin.status') + ' updated', 'success');
      queryClient.invalidateQueries({ queryKey: ['admin-articles'] });
    },
  });

  const topMutation = useMutation({
    mutationFn: (id: number) => articleApi.toggleTop(id),
    onSuccess: () => {
      showToast('置顶状态已更新', 'success');
      queryClient.invalidateQueries({ queryKey: ['admin-articles'] });
    },
  });

  if (isLoading) return <Loading />;

  return (
    <div className="space-y-6">
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4"
      >
        <h1 className="text-3xl font-serif font-bold text-neutral-800 dark:text-neutral-100">
          {t('admin.articleManagement')}
        </h1>
        <div className="flex flex-col sm:flex-row sm:items-center gap-3 w-full sm:w-auto">
          <div className="flex flex-col sm:flex-row sm:items-center gap-3 rounded-2xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 px-3 py-3 shadow-sm">
            <div className="min-w-0 sm:pr-1">
              <div className="text-xs font-semibold text-neutral-700 dark:text-neutral-200">排序方式</div>
              <div className="text-[11px] text-neutral-400 dark:text-neutral-500">控制文章列表与首页展示顺序</div>
            </div>
            <div className="grid grid-cols-2 gap-1 rounded-xl bg-neutral-100 dark:bg-neutral-800 p-1">
              <button
                type="button"
                onClick={() => sortModeMutation.mutate(false)}
                disabled={sortModeMutation.isPending}
                className={`min-w-20 rounded-lg px-3 py-1.5 text-xs font-semibold transition-all ${
                  !sortMode?.enabled
                    ? 'bg-white dark:bg-neutral-700 text-primary-600 dark:text-primary-300 shadow-sm'
                    : 'text-neutral-500 dark:text-neutral-400 hover:text-neutral-800 dark:hover:text-neutral-100'
                } ${sortModeMutation.isPending ? 'cursor-not-allowed opacity-70' : ''}`}
              >
                发布时间
              </button>
              <button
                type="button"
                onClick={() => sortModeMutation.mutate(true)}
                disabled={sortModeMutation.isPending}
                className={`min-w-20 rounded-lg px-3 py-1.5 text-xs font-semibold transition-all ${
                  sortMode?.enabled
                    ? 'bg-white dark:bg-neutral-700 text-primary-600 dark:text-primary-300 shadow-sm'
                    : 'text-neutral-500 dark:text-neutral-400 hover:text-neutral-800 dark:hover:text-neutral-100'
                } ${sortModeMutation.isPending ? 'cursor-not-allowed opacity-70' : ''}`}
              >
                活跃度
              </button>
            </div>
          </div>
          <Link
            to="/admin/articles/new"
            className="flex items-center justify-center gap-2 px-5 py-2.5 bg-primary-500 text-white rounded-xl font-medium hover:bg-primary-600 transition-all shadow-md hover:shadow-lg"
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            {t('admin.newArticle')}
          </Link>
        </div>
      </motion.div>

      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.1 }}
        className="bg-white dark:bg-neutral-900 rounded-2xl shadow-sm border border-neutral-100 dark:border-neutral-800 overflow-hidden"
      >
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="bg-neutral-50 dark:bg-neutral-800/50 border-b border-neutral-100 dark:border-neutral-800">
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.title')}</th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">置顶</th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.status')}</th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.createdAt')}</th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400 text-right">{t('admin.actions')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-neutral-100 dark:divide-neutral-800">
            {articlesData?.data?.map((article, index) => (
              <motion.tr
                key={article.id}
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ delay: index * 0.05 }}
                className="hover:bg-neutral-50 dark:hover:bg-neutral-800/50 transition-colors"
              >
                <td className="px-6 py-4">
                  <div className="font-medium text-neutral-800 dark:text-neutral-200">{article.title}</div>
                  <div className="text-xs text-neutral-400 dark:text-neutral-500 mt-1">{article.category.name}</div>
                </td>
                <td className="px-6 py-4">
                  <button
                    onClick={() => topMutation.mutate(article.id)}
                    className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none ${
                      article.is_top ? 'bg-primary-500' : 'bg-neutral-200 dark:bg-neutral-700'
                    }`}
                  >
                    <span
                      className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                        article.is_top ? 'translate-x-6' : 'translate-x-1'
                      }`}
                    />
                  </button>
                </td>
                <td className="px-6 py-4">
                  <span
                    className={`px-3 py-1.5 rounded-full text-xs font-medium ${
                      article.status === 'published'
                        ? 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400'
                        : 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600 dark:text-yellow-400'
                    }`}
                  >
                    {article.status === 'published' ? t('admin.published') : t('admin.draft')}
                  </span>
                </td>
                <td className="px-6 py-4 text-sm text-neutral-500 dark:text-neutral-400">
                  {formatDate(article.created_at)}
                </td>
                <td className="px-6 py-4 text-right">
                  <div className="flex items-center justify-end gap-2">
                    <button
                      onClick={() => statusMutation.mutate({ id: article.id, status: article.status })}
                      className="px-3 py-1.5 text-sm text-primary-600 dark:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-900/30 rounded-lg transition-colors"
                    >
                      {article.status === 'published' ? t('admin.toDraft') : t('admin.publish')}
                    </button>
                    <Link
                      to={`/admin/articles/edit/${article.id}`}
                      className="px-3 py-1.5 text-sm text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/30 rounded-lg transition-colors"
                    >
                      {t('admin.edit')}
                    </Link>
                    <button
                      onClick={() => setDeleteId(article.id)}
                      className="px-3 py-1.5 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/30 rounded-lg transition-colors"
                    >
                      {t('admin.delete')}
                    </button>
                  </div>
                </td>
              </motion.tr>
            ))}
          </tbody>
        </table>
      </motion.div>

      {articlesData && articlesData.totalPages > 1 && (
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
            {page} / {articlesData.totalPages}
          </span>
          <button
            onClick={() => setPage((p) => Math.min(articlesData.totalPages, p + 1))}
            disabled={page === articlesData.totalPages}
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
        isOpen={deleteId !== null}
        title={t('admin.delete')}
        message={t('admin.confirmDelete')}
        onConfirm={() => {
          if (deleteId) {
            deleteMutation.mutate(deleteId);
            setDeleteId(null);
          }
        }}
        onCancel={() => setDeleteId(null)}
        isDanger
      />
    </div>
  );
};
