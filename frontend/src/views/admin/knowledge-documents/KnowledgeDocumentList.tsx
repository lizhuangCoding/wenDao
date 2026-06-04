import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Search } from 'lucide-react';
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
  if (status === 'approved') {
    return { label: '已通过', variant: 'success' };
  }
  if (status === 'rejected') {
    return { label: '已拒绝', variant: 'danger' };
  }
  if (status === 'pending_review') {
    return { label: '待审核', variant: 'warning' };
  }
  return { label: status, variant: 'neutral' };
};

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
      <PageHeader title="知识文档审核" description="审核 AI 生成的知识文档，并管理它们生成的文章。" />

      <Panel className="space-y-3">
        <form onSubmit={applySearch} className="grid gap-3 md:grid-cols-[1fr_auto_auto]">
          <TextInput
            value={keywordInput}
            onChange={(event) => setKeywordInput(event.target.value)}
            placeholder="搜索标题、摘要或正文"
            leading={<Search className="h-4 w-4" />}
          />
          <SelectInput
            value={status}
            onChange={(event) => changeStatus(event.target.value as KnowledgeDocumentStatusFilter)}
          >
            <option value="">全部状态</option>
            <option value="pending_review">待审核</option>
            <option value="approved">已通过</option>
            <option value="rejected">已拒绝</option>
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
          deleteLabel="删除知识文档"
        />
      </Panel>

      {isError ? (
        <ErrorState message={(error as any)?.message || '知识文档列表加载失败'} onRetry={() => refetch()} />
      ) : (
        <DataTable
          emptyState={
            documents.length === 0 ? (
              <EmptyState title="暂无知识文档" description="当前筛选条件下没有知识文档。" className="m-6" />
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
                      aria-label="选择当前页知识文档"
                    />
              </DataTableHeaderCell>
              <DataTableHeaderCell>标题</DataTableHeaderCell>
              <DataTableHeaderCell>状态</DataTableHeaderCell>
              <DataTableHeaderCell>创建时间</DataTableHeaderCell>
              <DataTableHeaderCell align="right">操作</DataTableHeaderCell>
            </DataTableHeadRow>
              </thead>
          <DataTableBody>
            {documents.map((doc) => {
              const statusMeta = getKnowledgeDocumentStatusMeta(doc.status);
              return (
                <DataTableRow key={doc.id}>
                  <DataTableCell>
                      <input
                        type="checkbox"
                        checked={selectedIds.includes(doc.id)}
                        onChange={() => toggleDocumentSelection(doc.id)}
                        className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                        aria-label={`选择知识文档 ${doc.title}`}
                      />
                  </DataTableCell>
                  <DataTableCell className="font-medium text-neutral-800 dark:text-neutral-200">{doc.title}</DataTableCell>
                  <DataTableCell>
                      <div className="flex flex-col gap-1">
                      <StatusBadge variant={statusMeta.variant}>{statusMeta.label}</StatusBadge>
                      {doc.article_id && (
                        <span className="text-xs text-primary-600 dark:text-primary-400">
                          已生成文章 #{doc.article_id}
                        </span>
                      )}
                      </div>
                  </DataTableCell>
                  <DataTableCell>
                      {new Date(doc.created_at).toLocaleString()}
                  </DataTableCell>
                  <DataTableCell align="right">
                    <Link
                      to={`/admin/knowledge-documents/${doc.id}`}
                      className={getButtonClassName({
                        variant: 'ghost',
                        size: 'sm',
                        className: 'text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/30',
                      })}
                    >
                        查看
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
        isConfirming={batchDeleteMutation.isPending}
        isDanger
      />
    </div>
  );
};
