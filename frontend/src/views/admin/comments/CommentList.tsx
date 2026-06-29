import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Search } from 'lucide-react';
import { commentApi } from '@/api';
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
  SelectInput,
  StatusBadge,
  TextInput,
} from '@/components/common';
import { formatDate } from '@/utils';
import { useUIStore } from '@/store';
import { getApiErrorMessage } from '@/utils/apiError';

type CommentStatusFilter = '' | 'normal' | 'deleted';

export const CommentList = () => {
  const { t } = useTranslation();
  const [pageSize, setPageSize] = useState(15);
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState<CommentStatusFilter>('normal');
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
    queryKey: ['admin-comments', page, pageSize, status, keyword],
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
      showToast(t('admin.commentDeleted'), 'success');
      setConfirmConfig({ isOpen: false, id: null, type: 'delete' });
      invalidateComments();
    },
    onError: (err) => {
      showToast(getApiErrorMessage(err, t('admin.deleteCommentFailed')), 'error');
    },
  });

  const batchDeleteMutation = useMutation({
    mutationFn: (ids: number[]) => commentApi.batchDeleteComments(ids),
    onSuccess: (result) => {
      showToast(t('admin.batchDeleteCommentsSuccess', { count: result.deleted_count }), 'success');
      setSelectedIds([]);
      setConfirmBatchDelete(false);
      invalidateComments();
      if (page > 1 && comments.length > 0 && selectedIds.length >= comments.length) {
        setPage((currentPage) => Math.max(1, currentPage - 1));
      }
    },
    onError: (err) => {
      showToast(getApiErrorMessage(err, t('admin.batchDeleteCommentFailed')), 'error');
    },
  });

  const restoreMutation = useMutation({
    mutationFn: (id: number) => commentApi.adminRestoreComment(id),
    onSuccess: () => {
      showToast(t('admin.commentRestored'), 'success');
      setConfirmConfig({ isOpen: false, id: null, type: 'restore' });
      invalidateComments();
    },
    onError: (err) => {
      showToast(getApiErrorMessage(err, t('admin.restoreCommentFailed')), 'error');
    },
  });

  const applySearch = (event: React.FormEvent) => {
    event.preventDefault();
    setKeyword(keywordInput.trim());
    setPage(1);
    setSelectedIds([]);
  };

  const resetFilters = () => {
    setStatus('normal');
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
      <PageHeader title={t('admin.commentManagement')} tone="admin" />

      <Panel className="space-y-3">
        <form onSubmit={applySearch} className="grid gap-3 md:grid-cols-[1fr_auto_auto]">
          <TextInput
            value={keywordInput}
            onChange={(event) => setKeywordInput(event.target.value)}
            placeholder={t('admin.searchComments')}
            leading={<Search className="h-4 w-4" />}
          />
          <SelectInput
            value={status}
            onChange={(event) => {
              setStatus(event.target.value as CommentStatusFilter);
              setPage(1);
              setSelectedIds([]);
            }}
          >
            <option value="">{t('admin.allStatus')}</option>
            <option value="normal">{t('admin.normal')}</option>
            <option value="deleted">{t('admin.deleted')}</option>
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
          deleteLabel={t('admin.deleteComment')}
        />
      </Panel>

      {isError ? (
        <ErrorState message={getApiErrorMessage(error, t('admin.commentListLoadingFailed'))} onRetry={() => refetch()} />
      ) : (
        <DataTable
          emptyState={
            comments.length === 0 ? (
              <EmptyState title={t('admin.noComments')} description={t('admin.noCommentsDescription')} className="m-6" />
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
                      aria-label={t('admin.selectCurrentPageComments')}
                    />
              </DataTableHeaderCell>
              <DataTableHeaderCell width="wide">{t('admin.commentContent')}</DataTableHeaderCell>
              <DataTableHeaderCell width="medium">{t('admin.author')}</DataTableHeaderCell>
              <DataTableHeaderCell width="medium">{t('admin.article')}</DataTableHeaderCell>
              <DataTableHeaderCell width="compact">{t('admin.status')}</DataTableHeaderCell>
              <DataTableHeaderCell width="medium">{t('admin.createdAt')}</DataTableHeaderCell>
              <DataTableHeaderCell width="actionsCompact" align="right">{t('admin.actions')}</DataTableHeaderCell>
            </DataTableHeadRow>
              </thead>
          <DataTableBody>
            {comments.map((comment) => (
              <DataTableRow
                    key={comment.id}
                  >
                <DataTableCell width="select" nowrap>
                      <input
                        type="checkbox"
                        checked={selectedIds.includes(comment.id)}
                        onChange={() => toggleCommentSelection(comment.id)}
                        className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                        aria-label={t('admin.selectComment', { id: comment.id })}
                      />
                </DataTableCell>
                <DataTableCell className="min-w-0">
                      <div className="line-clamp-2 text-sm text-neutral-700 dark:text-neutral-300" title={comment.content}>
                        {comment.content}
                      </div>
                </DataTableCell>
                <DataTableCell truncate title={comment.user?.username || '-'}>
                      {comment.user?.username || '-'}
                </DataTableCell>
                <DataTableCell className="min-w-0">
                      <div className="truncate" title={comment.article?.title || '-'}>
                        {comment.article?.title || '-'}
                      </div>
                </DataTableCell>
                <DataTableCell nowrap>
                  <StatusBadge variant={comment.status === 'normal' ? 'success' : 'danger'}>
                        {comment.status === 'normal' ? t('admin.normal') : t('admin.deleted')}
                  </StatusBadge>
                </DataTableCell>
                <DataTableCell nowrap>
                      {formatDate(comment.created_at)}
                </DataTableCell>
                <DataTableCell align="right" nowrap>
                      {comment.status === 'normal' ? (
                    <Button
                      variant="ghost"
                      size="sm"
                          onClick={() => setConfirmConfig({ isOpen: true, id: comment.id, type: 'delete' })}
                      className="text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/30"
                        >
                          {t('admin.delete')}
                    </Button>
                      ) : (
                    <Button
                      variant="ghost"
                      size="sm"
                          onClick={() => setConfirmConfig({ isOpen: true, id: comment.id, type: 'restore' })}
                      className="text-green-600 hover:bg-green-50 dark:text-green-400 dark:hover:bg-green-900/30"
                        >
                          {t('admin.restore')}
                    </Button>
                      )}
                </DataTableCell>
              </DataTableRow>
                ))}
          </DataTableBody>
        </DataTable>
      )}

      {commentsData && (
        <Pagination
          page={page}
          totalPages={totalPages}
          total={commentsData?.total}
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
        isConfirming={deleteMutation.isPending || restoreMutation.isPending}
        isDanger={confirmConfig.type === 'delete'}
      />

      <ConfirmModal
        isOpen={confirmBatchDelete}
        title={t('admin.deleteComment')}
        message={t('admin.confirmDeleteComment')}
        confirmText={t('common.delete')}
        onConfirm={() => batchDeleteMutation.mutate(selectedIds)}
        onCancel={() => setConfirmBatchDelete(false)}
        isConfirming={batchDeleteMutation.isPending}
        isDanger
      />
    </div>
  );
};
