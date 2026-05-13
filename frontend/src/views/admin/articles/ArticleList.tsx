import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Plus, Search } from 'lucide-react';
import { articleApi, categoryApi, siteApi } from '@/api';
import { Loading, ConfirmModal, Pagination, EmptyState, ErrorState, BulkActionBar } from '@/components/common';
import { formatDate } from '@/utils';
import { useUIStore } from '@/store';
import { motion } from 'framer-motion';

type ArticleStatusFilter = '' | 'published' | 'draft';

export const ArticleList = () => {
  const { t } = useTranslation();
  const pageSize = 10;
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState<ArticleStatusFilter>('');
  const [categoryID, setCategoryID] = useState('');
  const [keyword, setKeyword] = useState('');
  const [keywordInput, setKeywordInput] = useState('');
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [deleteId, setDeleteId] = useState<number | null>(null);
  const [confirmBatchDelete, setConfirmBatchDelete] = useState(false);
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();

  const { data: sortMode } = useQuery({
    queryKey: ['site-sort-mode'],
    queryFn: () => siteApi.getSortMode(),
  });

  const { data: categories } = useQuery({
    queryKey: ['categories'],
    queryFn: categoryApi.getCategories,
  });

  const {
    data: articlesData,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['admin-articles', page, status, categoryID, keyword, sortMode?.enabled],
    queryFn: () =>
      articleApi.getAdminArticles({
        page,
        pageSize,
        status: status || undefined,
        category_id: categoryID ? Number(categoryID) : undefined,
        keyword: keyword || undefined,
        sort_by_popularity: sortMode?.enabled,
      }),
  });

  const articles = articlesData?.data ?? [];
  const totalPages = Math.max(1, articlesData?.totalPages ?? 1);
  const currentPageIds = articles.map((article) => article.id);
  const allCurrentPageSelected =
    currentPageIds.length > 0 && currentPageIds.every((id) => selectedIds.includes(id));

  const invalidateArticles = () => {
    queryClient.invalidateQueries({ queryKey: ['admin-articles'] });
    queryClient.invalidateQueries({ queryKey: ['categories'] });
  };

  const sortModeMutation = useMutation({
    mutationFn: (enabled: boolean) => siteApi.setSortMode(enabled),
    onSuccess: () => {
      showToast('排序模式已更新', 'success');
      queryClient.invalidateQueries({ queryKey: ['site-sort-mode'] });
      invalidateArticles();
    },
    onError: (err: any) => {
      showToast(err.message || '切换失败，请重试', 'error');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => articleApi.deleteArticle(id),
    onSuccess: () => {
      showToast('文章已删除', 'success');
      setDeleteId(null);
      setSelectedIds((ids) => ids.filter((id) => id !== deleteId));
      invalidateArticles();
    },
    onError: (err: any) => {
      showToast(err.message || '删除失败', 'error');
    },
  });

  const batchDeleteMutation = useMutation({
    mutationFn: (ids: number[]) => articleApi.batchDeleteArticles(ids),
    onSuccess: (result) => {
      showToast(`已删除 ${result.deleted_count} 篇文章`, 'success');
      setSelectedIds([]);
      setConfirmBatchDelete(false);
      invalidateArticles();
      if (page > 1 && articles.length > 0 && selectedIds.length >= articles.length) {
        setPage((currentPage) => Math.max(1, currentPage - 1));
      }
    },
    onError: (err: any) => {
      showToast(err.message || '批量删除失败', 'error');
    },
  });

  const statusMutation = useMutation({
    mutationFn: ({ id, status: articleStatus }: { id: number; status: string }) =>
      articleStatus === 'published' ? articleApi.draftArticle(id) : articleApi.publishArticle(id),
    onSuccess: () => {
      showToast('文章状态已更新', 'success');
      invalidateArticles();
    },
    onError: (err: any) => {
      showToast(err.message || '状态更新失败', 'error');
    },
  });

  const topMutation = useMutation({
    mutationFn: (id: number) => articleApi.toggleTop(id),
    onSuccess: () => {
      showToast('置顶状态已更新', 'success');
      invalidateArticles();
    },
    onError: (err: any) => {
      showToast(err.message || '置顶状态更新失败', 'error');
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
    setCategoryID('');
    setKeyword('');
    setKeywordInput('');
    setPage(1);
    setSelectedIds([]);
  };

  const toggleArticleSelection = (id: number) => {
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
        className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between"
      >
        <h1 className="text-3xl font-serif font-bold text-neutral-800 dark:text-neutral-100">
          {t('admin.articleManagement')}
        </h1>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
          <div className="flex flex-col gap-3 rounded-2xl border border-neutral-200 bg-white px-3 py-3 shadow-sm dark:border-neutral-700 dark:bg-neutral-900 sm:flex-row sm:items-center">
            <div className="min-w-0 sm:pr-1">
              <div className="text-xs font-semibold text-neutral-700 dark:text-neutral-200">排序方式</div>
              <div className="text-[11px] text-neutral-400 dark:text-neutral-500">控制文章列表与首页展示顺序</div>
            </div>
            <div className="grid grid-cols-2 gap-1 rounded-xl bg-neutral-100 p-1 dark:bg-neutral-800">
              <button
                type="button"
                onClick={() => sortModeMutation.mutate(false)}
                disabled={sortModeMutation.isPending}
                className={`min-w-20 rounded-lg px-3 py-1.5 text-xs font-semibold transition-all ${
                  !sortMode?.enabled
                    ? 'bg-white text-primary-600 shadow-sm dark:bg-neutral-700 dark:text-primary-300'
                    : 'text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-neutral-100'
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
                    ? 'bg-white text-primary-600 shadow-sm dark:bg-neutral-700 dark:text-primary-300'
                    : 'text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-neutral-100'
                } ${sortModeMutation.isPending ? 'cursor-not-allowed opacity-70' : ''}`}
              >
                活跃度
              </button>
            </div>
          </div>
          <Link
            to="/admin/articles/new"
            className="inline-flex items-center justify-center gap-2 rounded-xl bg-primary-500 px-5 py-2.5 font-medium text-white shadow-md transition-all hover:bg-primary-600 hover:shadow-lg"
          >
            <Plus className="h-5 w-5" />
            {t('admin.newArticle')}
          </Link>
        </div>
      </motion.div>

      <div className="space-y-3 rounded-2xl border border-neutral-100 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
        <form onSubmit={applySearch} className="grid gap-3 md:grid-cols-[1fr_auto_auto_auto]">
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
            onChange={(event) => {
              setStatus(event.target.value as ArticleStatusFilter);
              setPage(1);
              setSelectedIds([]);
            }}
            className="rounded-xl border border-neutral-200 bg-white px-3 py-2.5 text-sm text-neutral-700 outline-none focus:border-primary-400 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100"
          >
            <option value="">全部状态</option>
            <option value="published">已发布</option>
            <option value="draft">草稿</option>
          </select>
          <select
            value={categoryID}
            onChange={(event) => {
              setCategoryID(event.target.value);
              setPage(1);
              setSelectedIds([]);
            }}
            className="rounded-xl border border-neutral-200 bg-white px-3 py-2.5 text-sm text-neutral-700 outline-none focus:border-primary-400 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100"
          >
            <option value="">全部分类</option>
            {categories?.map((category) => (
              <option key={category.id} value={category.id}>
                {category.name}
              </option>
            ))}
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
          deleteLabel="删除文章"
        />
      </div>

      {isError ? (
        <ErrorState message={(error as any)?.message || '文章列表加载失败'} onRetry={() => refetch()} />
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
                      aria-label="选择当前页文章"
                    />
                  </th>
                  <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.title')}</th>
                  <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">置顶</th>
                  <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.status')}</th>
                  <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.createdAt')}</th>
                  <th className="px-6 py-4 text-right text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100 dark:divide-neutral-800">
                {articles.map((article, index) => (
                  <motion.tr
                    key={article.id}
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    transition={{ delay: index * 0.03 }}
                    className="transition-colors hover:bg-neutral-50 dark:hover:bg-neutral-800/50"
                  >
                    <td className="px-6 py-4">
                      <input
                        type="checkbox"
                        checked={selectedIds.includes(article.id)}
                        onChange={() => toggleArticleSelection(article.id)}
                        className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                        aria-label={`选择文章 ${article.title}`}
                      />
                    </td>
                    <td className="px-6 py-4">
                      <div className="font-medium text-neutral-800 dark:text-neutral-200">{article.title}</div>
                      <div className="mt-1 text-xs text-neutral-400 dark:text-neutral-500">{article.category.name}</div>
                    </td>
                    <td className="px-6 py-4">
                      <button
                        type="button"
                        onClick={() => topMutation.mutate(article.id)}
                        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none ${
                          article.is_top ? 'bg-primary-500' : 'bg-neutral-200 dark:bg-neutral-700'
                        }`}
                        aria-label="切换置顶"
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
                        className={`rounded-full px-3 py-1.5 text-xs font-medium ${
                          article.status === 'published'
                            ? 'bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400'
                            : 'bg-yellow-100 text-yellow-600 dark:bg-yellow-900/30 dark:text-yellow-400'
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
                          type="button"
                          onClick={() => statusMutation.mutate({ id: article.id, status: article.status })}
                          className="rounded-lg px-3 py-1.5 text-sm text-primary-600 transition-colors hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/30"
                        >
                          {article.status === 'published' ? t('admin.toDraft') : t('admin.publish')}
                        </button>
                        <Link
                          to={`/admin/articles/edit/${article.id}`}
                          className="rounded-lg px-3 py-1.5 text-sm text-blue-600 transition-colors hover:bg-blue-50 dark:text-blue-400 dark:hover:bg-blue-900/30"
                        >
                          {t('admin.edit')}
                        </Link>
                        <button
                          type="button"
                          onClick={() => setDeleteId(article.id)}
                          className="rounded-lg px-3 py-1.5 text-sm text-red-600 transition-colors hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/30"
                        >
                          {t('admin.delete')}
                        </button>
                      </div>
                    </td>
                  </motion.tr>
                ))}
              </tbody>
            </table>
          </div>
          {articles.length === 0 && (
            <EmptyState title="暂无文章" description="当前筛选条件下没有文章。" className="m-6" />
          )}
        </motion.div>
      )}

      {articlesData && (
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
        isOpen={deleteId !== null}
        title={t('admin.delete')}
        message={t('admin.confirmDelete')}
        onConfirm={() => {
          if (deleteId) {
            deleteMutation.mutate(deleteId);
          }
        }}
        onCancel={() => setDeleteId(null)}
        isDanger
      />

      <ConfirmModal
        isOpen={confirmBatchDelete}
        title="批量删除文章"
        message={`确定删除选中的 ${selectedIds.length} 篇文章吗？关联评论、统计和向量数据也会同步清理。`}
        confirmText="删除"
        onConfirm={() => batchDeleteMutation.mutate(selectedIds)}
        onCancel={() => setConfirmBatchDelete(false)}
        isDanger
      />
    </div>
  );
};
