import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Plus, Search } from 'lucide-react';
import { articleApi, categoryApi, siteApi } from '@/api';
import {
  Button,
  BulkActionBar,
  ConfirmModal,
  DataTable,
  DataTableBody,
  DataTableCell,
  DataTableHeaderCell,
  DataTableHeadRow,
  DataTableRow,
  EmptyState,
  ErrorState,
  Loading,
  PageHeader,
  Pagination,
  Panel,
  SegmentedControl,
  SelectInput,
  StatusBadge,
  TextInput,
  ToggleSwitch,
  getButtonClassName,
} from '@/components/common';
import { formatDate } from '@/utils';
import { useUIStore } from '@/store';

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
      <PageHeader
        title={t('admin.articleManagement')}
        actions={
          <>
            <Panel padding="sm" className="flex flex-col gap-3 border-neutral-200 dark:border-neutral-700 sm:flex-row sm:items-center">
            <div className="min-w-0 sm:pr-1">
              <div className="text-xs font-semibold text-neutral-700 dark:text-neutral-200">排序方式</div>
              <div className="text-[11px] text-neutral-400 dark:text-neutral-500">控制文章列表与首页展示顺序</div>
            </div>
              <SegmentedControl
                value={sortMode?.enabled ? 'popularity' : 'created'}
                items={[
                  { label: '发布时间', value: 'created', disabled: sortModeMutation.isPending },
                  { label: '活跃度', value: 'popularity', disabled: sortModeMutation.isPending },
                ]}
                onChange={(value) => sortModeMutation.mutate(value === 'popularity')}
              />
            </Panel>
          <Link
            to="/admin/articles/new"
              className={getButtonClassName({ variant: 'primary', size: 'lg' })}
          >
            <Plus className="h-5 w-5" />
            {t('admin.newArticle')}
          </Link>
          </>
        }
      />

      <Panel className="space-y-3">
        <form onSubmit={applySearch} className="grid gap-3 md:grid-cols-[1fr_auto_auto_auto]">
          <TextInput
            value={keywordInput}
            onChange={(event) => setKeywordInput(event.target.value)}
            placeholder="搜索标题、摘要或正文"
            leading={<Search className="h-4 w-4" />}
          />
          <SelectInput
            value={status}
            onChange={(event) => {
              setStatus(event.target.value as ArticleStatusFilter);
              setPage(1);
              setSelectedIds([]);
            }}
          >
            <option value="">全部状态</option>
            <option value="published">已发布</option>
            <option value="draft">草稿</option>
          </SelectInput>
          <SelectInput
            value={categoryID}
            onChange={(event) => {
              setCategoryID(event.target.value);
              setPage(1);
              setSelectedIds([]);
            }}
          >
            <option value="">全部分类</option>
            {categories?.map((category) => (
              <option key={category.id} value={category.id}>
                {category.name}
              </option>
            ))}
          </SelectInput>
          <div className="flex gap-2">
            <Button type="submit">
              搜索
            </Button>
            <Button variant="secondary" onClick={resetFilters}>
              重置
            </Button>
          </div>
        </form>
        <BulkActionBar
          selectedCount={selectedIds.length}
          onDelete={() => setConfirmBatchDelete(true)}
          onClear={() => setSelectedIds([])}
          isDeleting={batchDeleteMutation.isPending}
          deleteLabel="删除文章"
        />
      </Panel>

      {isError ? (
        <ErrorState message={(error as any)?.message || '文章列表加载失败'} onRetry={() => refetch()} />
      ) : (
        <DataTable
          emptyState={
            articles.length === 0 ? (
              <EmptyState title="暂无文章" description="当前筛选条件下没有文章。" className="m-6" />
            ) : null
          }
        >
          <thead>
            <DataTableHeadRow>
              <DataTableHeaderCell>
                    <input
                      type="checkbox"
                      checked={allCurrentPageSelected}
                      onChange={toggleCurrentPageSelection}
                      className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                      aria-label="选择当前页文章"
                    />
              </DataTableHeaderCell>
              <DataTableHeaderCell>{t('admin.title')}</DataTableHeaderCell>
              <DataTableHeaderCell>置顶</DataTableHeaderCell>
              <DataTableHeaderCell>{t('admin.status')}</DataTableHeaderCell>
              <DataTableHeaderCell>{t('admin.createdAt')}</DataTableHeaderCell>
              <DataTableHeaderCell align="right">{t('admin.actions')}</DataTableHeaderCell>
            </DataTableHeadRow>
          </thead>
          <DataTableBody>
            {articles.map((article) => (
              <DataTableRow
                    key={article.id}
                  >
                <DataTableCell>
                      <input
                        type="checkbox"
                        checked={selectedIds.includes(article.id)}
                        onChange={() => toggleArticleSelection(article.id)}
                        className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                        aria-label={`选择文章 ${article.title}`}
                      />
                </DataTableCell>
                <DataTableCell>
                      <div className="font-medium text-neutral-800 dark:text-neutral-200">{article.title}</div>
                      <div className="mt-1 text-xs text-neutral-400 dark:text-neutral-500">{article.category.name}</div>
                </DataTableCell>
                <DataTableCell>
                  <ToggleSwitch
                    checked={article.is_top}
                        onClick={() => topMutation.mutate(article.id)}
                        aria-label="切换置顶"
                  />
                </DataTableCell>
                <DataTableCell>
                  <StatusBadge variant={article.status === 'published' ? 'success' : 'warning'}>
                        {article.status === 'published' ? t('admin.published') : t('admin.draft')}
                  </StatusBadge>
                </DataTableCell>
                <DataTableCell>
                      {formatDate(article.created_at)}
                </DataTableCell>
                <DataTableCell align="right">
                      <div className="flex items-center justify-end gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                          onClick={() => statusMutation.mutate({ id: article.id, status: article.status })}
                      className="text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/30"
                        >
                          {article.status === 'published' ? t('admin.toDraft') : t('admin.publish')}
                    </Button>
                        <Link
                          to={`/admin/articles/edit/${article.id}`}
                      className={getButtonClassName({
                        variant: 'ghost',
                        size: 'sm',
                        className: 'text-blue-600 hover:bg-blue-50 dark:text-blue-400 dark:hover:bg-blue-900/30',
                      })}
                        >
                          {t('admin.edit')}
                        </Link>
                    <Button
                      variant="ghost"
                      size="sm"
                          onClick={() => setDeleteId(article.id)}
                      className="text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/30"
                        >
                          {t('admin.delete')}
                    </Button>
                      </div>
                </DataTableCell>
              </DataTableRow>
                ))}
          </DataTableBody>
        </DataTable>
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
        isConfirming={deleteMutation.isPending}
        isDanger
      />

      <ConfirmModal
        isOpen={confirmBatchDelete}
        title="批量删除文章"
        message={`确定删除选中的 ${selectedIds.length} 篇文章吗？关联评论、统计和向量数据也会同步清理。`}
        confirmText="删除"
        onConfirm={() => batchDeleteMutation.mutate(selectedIds)}
        onCancel={() => setConfirmBatchDelete(false)}
        isConfirming={batchDeleteMutation.isPending}
        isDanger
      />
    </div>
  );
};
