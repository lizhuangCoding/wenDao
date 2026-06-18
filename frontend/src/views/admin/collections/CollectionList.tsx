import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AnimatePresence, motion } from 'framer-motion';
import { Plus } from 'lucide-react';
import { collectionApi } from '@/api';
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
  TextInput,
} from '@/components/common';
import { useUIStore } from '@/store';
import { formatDate } from '@/utils';
import type { Collection } from '@/types';

type CollectionFormData = {
  name: string;
  slug: string;
  description: string;
  sort_order: number;
  status: 'active' | 'hidden';
};

const emptyForm: CollectionFormData = {
  name: '',
  slug: '',
  description: '',
  sort_order: 0,
  status: 'active',
};

export const CollectionList = () => {
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingCollection, setEditingCollection] = useState<Collection | null>(null);
  const [deleteId, setDeleteId] = useState<number | null>(null);
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [confirmBatchDelete, setConfirmBatchDelete] = useState(false);
  const [formData, setFormData] = useState<CollectionFormData>(emptyForm);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['admin-collections', page, pageSize],
    queryFn: () => collectionApi.getAdminCollections({ page, pageSize }),
  });

  const collections = data?.data ?? [];
  const totalPages = Math.max(1, data?.totalPages ?? 1);
  const currentPageIds = collections.map((collection) => collection.id);
  const allCurrentPageSelected =
    currentPageIds.length > 0 && currentPageIds.every((id) => selectedIds.includes(id));

  const invalidateCollections = () => {
    queryClient.invalidateQueries({ queryKey: ['admin-collections'] });
    queryClient.invalidateQueries({ queryKey: ['collections'] });
  };

  const saveMutation = useMutation({
    mutationFn: (payload: CollectionFormData) =>
      editingCollection
        ? collectionApi.updateCollection(editingCollection.id, payload)
        : collectionApi.createCollection(payload),
    onSuccess: () => {
      showToast(editingCollection ? '合集已更新' : '合集已创建', 'success');
      closeModal();
      invalidateCollections();
    },
    onError: (err: any) => showToast(err.message || '保存合集失败', 'error'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => collectionApi.deleteCollection(id),
    onSuccess: () => {
      showToast('合集已删除', 'success');
      setDeleteId(null);
      invalidateCollections();
      if (page > 1 && collections.length <= 1) {
        setPage((currentPage) => Math.max(1, currentPage - 1));
      }
    },
    onError: (err: any) => showToast(err.message || '删除合集失败', 'error'),
  });

  const batchDeleteMutation = useMutation({
    mutationFn: (ids: number[]) => collectionApi.batchDeleteCollections(ids),
    onSuccess: (result) => {
      showToast(`已删除 ${result.deleted_count} 个合集`, 'success');
      setSelectedIds([]);
      setConfirmBatchDelete(false);
      invalidateCollections();
      if (page > 1 && collections.length > 0 && selectedIds.length >= collections.length) {
        setPage((currentPage) => Math.max(1, currentPage - 1));
      }
    },
    onError: (err: any) => showToast(err.message || '批量删除合集失败', 'error'),
  });

  const openCreateModal = () => {
    setEditingCollection(null);
    setFormData(emptyForm);
    setIsModalOpen(true);
  };

  const openEditModal = (collection: Collection) => {
    setEditingCollection(collection);
    setFormData({
      name: collection.name,
      slug: collection.slug,
      description: collection.description || '',
      sort_order: collection.sort_order,
      status: collection.status,
    });
    setIsModalOpen(true);
  };

  const closeModal = () => {
    setIsModalOpen(false);
    setEditingCollection(null);
    setFormData(emptyForm);
  };

  const toggleCollectionSelection = (id: number) => {
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
        title="合集管理"
        actions={
          <Button size="lg" onClick={openCreateModal}>
            <Plus className="h-5 w-5" />
            新建合集
          </Button>
        }
      />

      <BulkActionBar
        selectedCount={selectedIds.length}
        onDelete={() => setConfirmBatchDelete(true)}
        onClear={() => setSelectedIds([])}
        isDeleting={batchDeleteMutation.isPending}
        deleteLabel="删除合集"
      />

      {isError ? (
        <ErrorState message={(error as any)?.message || '合集列表加载失败'} onRetry={() => refetch()} />
      ) : (
        <DataTable
          minWidth="920px"
          stretch={false}
          emptyState={
            collections.length === 0 ? (
              <EmptyState title="暂无合集" description="新建合集后，就可以把文章组织成连续阅读路径。" className="m-6" />
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
                  aria-label="选择当前页合集"
                />
              </DataTableHeaderCell>
              <DataTableHeaderCell width="medium">名称</DataTableHeaderCell>
              <DataTableHeaderCell width="medium">Slug</DataTableHeaderCell>
              <DataTableHeaderCell width="compact">排序</DataTableHeaderCell>
              <DataTableHeaderCell width="compact">文章数</DataTableHeaderCell>
              <DataTableHeaderCell width="compact">状态</DataTableHeaderCell>
              <DataTableHeaderCell width="compact">创建时间</DataTableHeaderCell>
              <DataTableHeaderCell width="actions">操作</DataTableHeaderCell>
            </DataTableHeadRow>
          </thead>
          <DataTableBody>
            {collections.map((collection) => (
              <DataTableRow key={collection.id}>
                <DataTableCell width="select" nowrap>
                  <input
                    type="checkbox"
                    checked={selectedIds.includes(collection.id)}
                    onChange={() => toggleCollectionSelection(collection.id)}
                    className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                    aria-label={`选择合集 ${collection.name}`}
                  />
                </DataTableCell>
                <DataTableCell truncate className="font-medium text-neutral-800 dark:text-neutral-200" title={collection.name}>
                  {collection.name}
                </DataTableCell>
                <DataTableCell truncate title={collection.slug}>{collection.slug}</DataTableCell>
                <DataTableCell nowrap className="tabular-nums text-neutral-500 dark:text-neutral-400">
                  {collection.sort_order}
                </DataTableCell>
                <DataTableCell nowrap>{collection.article_count}</DataTableCell>
                <DataTableCell nowrap>
                  <span className={collection.status === 'active' ? 'text-emerald-600' : 'text-neutral-500'}>
                    {collection.status === 'active' ? '启用' : '隐藏'}
                  </span>
                </DataTableCell>
                <DataTableCell nowrap>{formatDate(collection.created_at)}</DataTableCell>
                <DataTableCell nowrap>
                  <div className="inline-flex items-center gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => openEditModal(collection)}
                      className="text-blue-600 hover:bg-blue-50 dark:text-blue-400 dark:hover:bg-blue-900/30"
                    >
                      编辑
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setDeleteId(collection.id)}
                      className="text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/30"
                    >
                      删除
                    </Button>
                  </div>
                </DataTableCell>
              </DataTableRow>
            ))}
          </DataTableBody>
        </DataTable>
      )}

      {data && (
        <Pagination
          page={page}
          totalPages={totalPages}
          total={data.total}
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
          previousLabel="上一页"
          nextLabel="下一页"
        />
      )}

      <ConfirmModal
        isOpen={deleteId !== null}
        title="删除合集"
        message="确认删除这个合集吗？包含文章的合集不能删除。"
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
        title="删除合集"
        message="确认删除选中的合集吗？包含文章的合集不能删除。"
        confirmText="删除"
        onConfirm={() => batchDeleteMutation.mutate(selectedIds)}
        onCancel={() => setConfirmBatchDelete(false)}
        isConfirming={batchDeleteMutation.isPending}
        isDanger
      />

      <AnimatePresence>
        {isModalOpen && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm"
          >
            <motion.div
              initial={{ scale: 0.95, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.95, opacity: 0 }}
              className="w-full max-w-md"
            >
              <Panel padding="none" className="overflow-hidden shadow-xl">
                <div className="border-b border-neutral-100 px-6 py-4 dark:border-neutral-800">
                  <h3 className="text-xl font-bold text-neutral-800 dark:text-neutral-100">
                    {editingCollection ? '编辑合集' : '新建合集'}
                  </h3>
                </div>
                <div className="space-y-5 p-6">
                  <div>
                    <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">名称</label>
                    <TextInput
                      type="text"
                      value={formData.name}
                      onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                      placeholder="Go 语言入门系列"
                    />
                  </div>
                  <div>
                    <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">Slug</label>
                    <TextInput
                      type="text"
                      value={formData.slug}
                      onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
                      placeholder="go-intro"
                    />
                  </div>
                  <div>
                    <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">排序</label>
                    <TextInput
                      type="number"
                      min={0}
                      step={1}
                      value={formData.sort_order}
                      onChange={(e) => setFormData({ ...formData, sort_order: Number(e.target.value) })}
                      placeholder="0"
                    />
                  </div>
                  <div>
                    <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">状态</label>
                    <select
                      className="w-full rounded-xl border border-neutral-200 bg-neutral-50 px-4 py-3 text-neutral-800 transition-all focus:outline-none focus:ring-2 focus:ring-primary-500 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
                      value={formData.status}
                      onChange={(e) => setFormData({ ...formData, status: e.target.value as CollectionFormData['status'] })}
                    >
                      <option value="active">启用</option>
                      <option value="hidden">隐藏</option>
                    </select>
                  </div>
                  <div>
                    <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">描述</label>
                    <textarea
                      className="w-full resize-none rounded-xl border border-neutral-200 bg-neutral-50 px-4 py-3 text-neutral-800 transition-all focus:outline-none focus:ring-2 focus:ring-primary-500 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
                      rows={3}
                      value={formData.description}
                      onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                      placeholder="这个合集的阅读目标和范围"
                    />
                  </div>
                </div>
                <div className="flex justify-end gap-3 bg-neutral-50 px-6 py-4 dark:bg-neutral-800/50">
                  <Button variant="secondary" onClick={closeModal}>
                    取消
                  </Button>
                  <Button
                    onClick={() => {
                      if (!formData.name.trim() || !formData.slug.trim()) {
                        showToast('请填写名称和 Slug', 'error');
                        return;
                      }
                      saveMutation.mutate({
                        ...formData,
                        sort_order: Number.isFinite(formData.sort_order) ? formData.sort_order : 0,
                      });
                    }}
                    disabled={saveMutation.isPending}
                  >
                    {saveMutation.isPending ? '保存中' : '保存'}
                  </Button>
                </div>
              </Panel>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};
