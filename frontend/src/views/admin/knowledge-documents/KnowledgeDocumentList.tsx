import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Search } from 'lucide-react';
import i18n from '@/i18n';
import { knowledgeDocumentApi } from '@/api/knowledgeDocument';
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
  getButtonClassName,
} from '@/components/common';
import { useUIStore } from '@/store';

type KnowledgeDocumentStatusFilter = 'pending_review' | 'approved' | 'rejected' | '';

const getKnowledgeDocumentStatusMeta = (
  status: string
): { label: string; variant: 'success' | 'warning' | 'danger' | 'neutral' } => {
  const isEnglish = (i18n.resolvedLanguage || i18n.language || 'zh').startsWith('en');
  if (status === 'approved') {
    return { label: isEnglish ? 'Approved' : '已通过', variant: 'success' };
  }
  if (status === 'rejected') {
    return { label: isEnglish ? 'Rejected' : '已拒绝', variant: 'danger' };
  }
  if (status === 'pending_review') {
    return { label: isEnglish ? 'Pending Review' : '待审核', variant: 'warning' };
  }
  return { label: status, variant: 'neutral' };
};

export const KnowledgeDocumentList = () => {
  const { t } = useTranslation();
  const [pageSize, setPageSize] = useState(10);
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
    queryKey: ['admin-knowledge-documents', page, pageSize, status, keyword],
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
    onSuccess: () => {
      showToast(t('knowledgeDocument.deleteSuccess'), 'success');
      setSelectedIds([]);
      setConfirmBatchDelete(false);
      queryClient.invalidateQueries({ queryKey: ['admin-knowledge-documents'] });
      if (page > 1 && documents.length > 0 && selectedIds.length >= documents.length) {
        setPage((currentPage) => Math.max(1, currentPage - 1));
      }
    },
    onError: (err: any) => {
      showToast(err.message || t('knowledgeDocument.deleteFailed'), 'error');
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
      <PageHeader title={t('knowledgeDocument.reviewTitle')} description={t('knowledgeDocument.reviewDescription')} />

      <Panel className="space-y-3">
        <form onSubmit={applySearch} className="grid gap-3 md:grid-cols-[1fr_auto_auto]">
          <TextInput
            value={keywordInput}
            onChange={(event) => setKeywordInput(event.target.value)}
            placeholder={t('knowledgeDocument.searchPlaceholder')}
            leading={<Search className="h-4 w-4" />}
          />
          <SelectInput
            value={status}
            onChange={(event) => changeStatus(event.target.value as KnowledgeDocumentStatusFilter)}
          >
            <option value="">{t('knowledgeDocument.allStatus')}</option>
            <option value="pending_review">{t('knowledgeDocument.statusPendingReview')}</option>
            <option value="approved">{t('knowledgeDocument.statusApprovedOption')}</option>
            <option value="rejected">{t('knowledgeDocument.statusRejectedOption')}</option>
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
          deleteLabel={t('knowledgeDocument.delete')}
        />
      </Panel>

      {isError ? (
        <ErrorState message={(error as any)?.message || t('knowledgeDocument.listLoadFailed')} onRetry={() => refetch()} />
      ) : (
        <DataTable
          emptyState={
            documents.length === 0 ? (
              <EmptyState title={t('knowledgeDocument.noDocuments')} description={t('knowledgeDocument.noDocumentsDescription')} className="m-6" />
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
                      aria-label={t('knowledgeDocument.selectCurrentPage')}
                    />
              </DataTableHeaderCell>
              <DataTableHeaderCell width="wide">{t('admin.title')}</DataTableHeaderCell>
              <DataTableHeaderCell width="medium">{t('admin.status')}</DataTableHeaderCell>
              <DataTableHeaderCell width="medium">{t('knowledgeDocument.createdAt')}</DataTableHeaderCell>
              <DataTableHeaderCell width="actions" align="right">{t('knowledgeDocument.actionView')}</DataTableHeaderCell>
            </DataTableHeadRow>
              </thead>
          <DataTableBody>
            {documents.map((doc) => {
              const statusMeta = getKnowledgeDocumentStatusMeta(doc.status);
              return (
                <DataTableRow key={doc.id}>
                  <DataTableCell width="select" nowrap>
                      <input
                        type="checkbox"
                        checked={selectedIds.includes(doc.id)}
                        onChange={() => toggleDocumentSelection(doc.id)}
                        className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                        aria-label={t('knowledgeDocument.selectDocument', { title: doc.title })}
                      />
                  </DataTableCell>
                  <DataTableCell truncate className="font-medium text-neutral-800 dark:text-neutral-200" title={doc.title}>
                    {doc.title}
                  </DataTableCell>
                  <DataTableCell nowrap>
                      <div className="flex flex-col gap-1">
                      <StatusBadge variant={statusMeta.variant}>{statusMeta.label}</StatusBadge>
                      {doc.article_id && (
                        <span className="text-xs text-primary-600 dark:text-primary-400">
                          {t('knowledgeDocument.articleGenerated', { id: doc.article_id })}
                        </span>
                      )}
                      </div>
                  </DataTableCell>
                  <DataTableCell nowrap>
                      {new Date(doc.created_at).toLocaleString()}
                  </DataTableCell>
                  <DataTableCell align="right" nowrap>
                    <Link
                      to={`/admin/knowledge-documents/${doc.id}`}
                      className={getButtonClassName({
                        variant: 'ghost',
                        size: 'sm',
                        className: 'text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/30',
                      })}
                    >
                        {t('knowledgeDocument.actionView')}
                      </Link>
                  </DataTableCell>
                </DataTableRow>
              );
            })}
          </DataTableBody>
        </DataTable>
      )}

      {documentsData && (
        <Pagination
          page={page}
          totalPages={totalPages}
          total={documentsData?.total}
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
        />
      )}

      <ConfirmModal
        isOpen={confirmBatchDelete}
        title={t('knowledgeDocument.delete')}
        message={t('knowledgeDocument.deleteConfirmMessage')}
        confirmText={t('common.delete')}
        onConfirm={() => batchDeleteMutation.mutate(selectedIds)}
        onCancel={() => setConfirmBatchDelete(false)}
        isConfirming={batchDeleteMutation.isPending}
        isDanger
      />
    </div>
  );
};
