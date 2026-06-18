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
  const [pageSize, setPageSize] = useState(10);
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
    queryKey: ['admin-articles', page, pageSize, status, categoryID, keyword, sortMode?.enabled],
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
      showToast(t('admin.sortModeUpdated'), 'success');
      queryClient.invalidateQueries({ queryKey: ['site-sort-mode'] });
      invalidateArticles();
    },
    onError: (err: any) => {
      showToast(err.message || t('admin.switchFailed'), 'error');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => articleApi.deleteArticle(id),
    onSuccess: () => {
      showToast(t('common.success'), 'success');
      setDeleteId(null);
      setSelectedIds((ids) => ids.filter((id) => id !== deleteId));
      invalidateArticles();
    },
    onError: (err: any) => {
      showToast(err.message || t('admin.deleteFailed'), 'error');
    },
  });

  const batchDeleteMutation = useMutation({
    mutationFn: (ids: number[]) => articleApi.batchDeleteArticles(ids),
    onSuccess: (result) => {
      showToast(t('admin.batchDeleteArticlesSuccess', { count: result.deleted_count }), 'success');
      setSelectedIds([]);
      setConfirmBatchDelete(false);
      invalidateArticles();
      if (page > 1 && articles.length > 0 && selectedIds.length >= articles.length) {
        setPage((currentPage) => Math.max(1, currentPage - 1));
      }
    },
    onError: (err: any) => {
      showToast(err.message || t('admin.batchDeleteFailed'), 'error');
    },
  });

  const statusMutation = useMutation({
    mutationFn: ({ id, status: articleStatus }: { id: number; status: string }) =>
      articleStatus === 'published' ? articleApi.draftArticle(id) : articleApi.publishArticle(id),
    onSuccess: () => {
      showToast(t('admin.statusUpdated'), 'success');
      invalidateArticles();
    },
    onError: (err: any) => {
      showToast(err.message || t('admin.switchFailed'), 'error');
    },
  });

  const topMutation = useMutation({
    mutationFn: (id: number) => articleApi.toggleTop(id),
    onSuccess: () => {
      showToast(t('admin.topUpdated'), 'success');
      invalidateArticles();
    },
    onError: (err: any) => {
      showToast(err.message || t('admin.switchFailed'), 'error');
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
              <div className="text-xs font-semibold text-neutral-700 dark:text-neutral-200">{t('admin.sortBy')}</div>
              <div className="text-[11px] text-neutral-400 dark:text-neutral-500">{t('admin.sortHint')}</div>
            </div>
              <SegmentedControl
                value={sortMode?.enabled ? 'popularity' : 'created'}
                items={[
                  { label: t('admin.sortByCreated'), value: 'created', disabled: sortModeMutation.isPending },
                  { label: t('admin.sortByPopularity'), value: 'popularity', disabled: sortModeMutation.isPending },
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
            placeholder={t('admin.searchArticles')}
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
            <option value="">{t('admin.allStatus')}</option>
            <option value="published">{t('admin.published')}</option>
            <option value="draft">{t('admin.draft')}</option>
          </SelectInput>
          <SelectInput
            value={categoryID}
            onChange={(event) => {
              setCategoryID(event.target.value);
              setPage(1);
              setSelectedIds([]);
            }}
          >
            <option value="">{t('admin.allCategories')}</option>
            {categories?.map((category) => (
              <option key={category.id} value={category.id}>
                {category.name}
              </option>
            ))}
          </SelectInput>
          <div className="flex gap-2">
            <Button type="submit">
              {t('common.search')}
            </Button>
            <Button variant="secondary" onClick={resetFilters}>
              {t('common.reset')}
            </Button>
          </div>
        </form>
        <BulkActionBar
          selectedCount={selectedIds.length}
          onDelete={() => setConfirmBatchDelete(true)}
          onClear={() => setSelectedIds([])}
          isDeleting={batchDeleteMutation.isPending}
          deleteLabel={t('admin.deleteSelectedArticles')}
        />
      </Panel>

      {isError ? (
        <ErrorState message={(error as any)?.message || t('admin.articleListLoadingFailed')} onRetry={() => refetch()} />
      ) : (
        <DataTable
          emptyState={
            articles.length === 0 ? (
              <EmptyState title={t('admin.noArticles')} description={t('admin.noArticlesDescription')} className="m-6" />
            ) : null
          }
        >
          <thead>
            <DataTableHeadRow>
              <DataTableHeaderCell width="select">
                    <input
                      type="checkbox"
                      checked={allCurrentPageSelected}
                      onChange={toggleCurrentPageSelection}
                      className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                      aria-label={t('admin.selectCurrentPageArticles')}
                    />
              </DataTableHeaderCell>
              <DataTableHeaderCell width="wide">{t('admin.title')}</DataTableHeaderCell>
              <DataTableHeaderCell width="compact" align="center">{t('article.pinned')}</DataTableHeaderCell>
              <DataTableHeaderCell width="compact">{t('admin.status')}</DataTableHeaderCell>
              <DataTableHeaderCell width="medium">{t('admin.createdAt')}</DataTableHeaderCell>
              <DataTableHeaderCell width="actionsWide" align="right">{t('admin.actions')}</DataTableHeaderCell>
            </DataTableHeadRow>
          </thead>
          <DataTableBody>
            {articles.map((article) => (
              <DataTableRow
                    key={article.id}
                  >
                <DataTableCell width="select" nowrap>
                      <input
                        type="checkbox"
                        checked={selectedIds.includes(article.id)}
                        onChange={() => toggleArticleSelection(article.id)}
                        className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                        aria-label={t('admin.selectArticle', { title: article.title })}
                      />
                </DataTableCell>
                <DataTableCell className="min-w-0">
                      <div className="truncate font-medium text-neutral-800 dark:text-neutral-200" title={article.title}>
                        {article.title}
                      </div>
                      <div className="mt-1 truncate text-xs text-neutral-400 dark:text-neutral-500" title={article.category.name}>
                        {article.category.name}
                      </div>
                </DataTableCell>
                <DataTableCell align="center" nowrap>
                  <ToggleSwitch
                    checked={article.is_top}
                        onClick={() => topMutation.mutate(article.id)}
                        aria-label={t('article.pinned')}
                  />
                </DataTableCell>
                <DataTableCell nowrap>
                  <StatusBadge variant={article.status === 'published' ? 'success' : 'warning'}>
                        {article.status === 'published' ? t('admin.published') : t('admin.draft')}
                  </StatusBadge>
                </DataTableCell>
                <DataTableCell nowrap>
                      {formatDate(article.created_at)}
                </DataTableCell>
                <DataTableCell align="right" nowrap>
                      <div className="inline-flex items-center justify-end gap-2">
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
          total={articlesData?.total}
          pageSize={pageSize}
          onChange={(nextPage) => {
            setPage(nextPage);
            setSelectedIds([]);
          }}
          onPageSizeChange={(newSize) => {
            setPageSize(newSize);
            setPage(1);
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
        title={t('admin.deleteSelectedArticles')}
        message={t('admin.confirmDeleteSelectedArticles', { count: selectedIds.length })}
        confirmText={t('common.delete')}
        onConfirm={() => batchDeleteMutation.mutate(selectedIds)}
        onCancel={() => setConfirmBatchDelete(false)}
        isConfirming={batchDeleteMutation.isPending}
        isDanger
      />
    </div>
  );
};
