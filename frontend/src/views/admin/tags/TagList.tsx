import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { tagApi } from '@/api';
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
import { formatDate } from '@/utils';
import { useUIStore } from '@/store';
import { getApiErrorMessage } from '@/utils/apiError';
import { motion, AnimatePresence } from 'framer-motion';
import type { Tag } from '@/types';

type TagFormData = {
  name: string;
  slug: string;
};

export const TagList = () => {
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const [pageSize, setPageSize] = useState(10);
  const [page, setPage] = useState(1);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingTag, setEditingTag] = useState<Tag | null>(null);
  const [deleteId, setDeleteId] = useState<number | null>(null);
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [confirmBatchDelete, setConfirmBatchDelete] = useState(false);
  const [formData, setFormData] = useState<TagFormData>({ name: '', slug: '' });

  const {
    data: tagsData,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['admin-tags', page, pageSize],
    queryFn: () => tagApi.getAdminTags({ page, pageSize }),
  });

  const tags = tagsData?.data ?? [];
  const totalPages = Math.max(1, tagsData?.totalPages ?? 1);
  const currentPageIds = tags.map((tag) => tag.id);
  const allCurrentPageSelected =
    currentPageIds.length > 0 && currentPageIds.every((id) => selectedIds.includes(id));

  const invalidateTags = () => {
    queryClient.invalidateQueries({ queryKey: ['admin-tags'] });
    queryClient.invalidateQueries({ queryKey: ['tags'] });
    queryClient.invalidateQueries({ queryKey: ['articles'] });
  };

  const saveMutation = useMutation({
    mutationFn: (data: TagFormData) =>
      editingTag ? tagApi.updateTag(editingTag.id, data) : tagApi.createTag(data),
    onSuccess: () => {
      showToast(editingTag ? '标签已更新' : '标签已创建', 'success');
      handleClose();
      invalidateTags();
    },
    onError: (err) => {
      showToast(getApiErrorMessage(err, '保存标签失败'), 'error');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => tagApi.deleteTag(id),
    onSuccess: () => {
      showToast('标签已删除', 'success');
      setDeleteId(null);
      invalidateTags();
      if (page > 1 && tags.length <= 1) {
        setPage((currentPage) => Math.max(1, currentPage - 1));
      }
    },
    onError: (err) => {
      showToast(getApiErrorMessage(err, '删除标签失败'), 'error');
    },
  });

  const batchDeleteMutation = useMutation({
    mutationFn: (ids: number[]) => tagApi.batchDeleteTags(ids),
    onSuccess: (result) => {
      showToast(`已删除 ${result.deleted_count} 个标签`, 'success');
      setSelectedIds([]);
      setConfirmBatchDelete(false);
      invalidateTags();
      if (page > 1 && tags.length > 0 && selectedIds.length >= tags.length) {
        setPage((currentPage) => Math.max(1, currentPage - 1));
      }
    },
    onError: (err) => {
      showToast(getApiErrorMessage(err, '批量删除标签失败'), 'error');
    },
  });

  const handleEdit = (tag: Tag) => {
    setEditingTag(tag);
    setFormData({ name: tag.name, slug: tag.slug });
    setIsModalOpen(true);
  };

  const handleClose = () => {
    setIsModalOpen(false);
    setEditingTag(null);
    setFormData({ name: '', slug: '' });
  };

  const toggleTagSelection = (id: number) => {
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
        title="标签管理"
        tone="admin"
        actions={
          <Button size="lg" onClick={() => setIsModalOpen(true)}>
            <Plus className="h-5 w-5" />
            新建标签
          </Button>
        }
      />

      <BulkActionBar
        selectedCount={selectedIds.length}
        onDelete={() => setConfirmBatchDelete(true)}
        onClear={() => setSelectedIds([])}
        isDeleting={batchDeleteMutation.isPending}
        deleteLabel="删除标签"
      />

      {isError ? (
        <ErrorState message={getApiErrorMessage(error, '标签列表加载失败')} onRetry={() => refetch()} />
      ) : (
        <DataTable
          minWidth="760px"
          stretch={false}
          emptyState={tags.length === 0 ? <EmptyState title="暂无标签" description="新建标签后即可在文章编辑器中选择" className="m-6" /> : null}
        >
          <thead>
            <DataTableHeadRow>
              <DataTableHeaderCell width="select">
                <input
                  type="checkbox"
                  checked={allCurrentPageSelected}
                  onChange={toggleCurrentPageSelection}
                  className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                  aria-label="选择当前页标签"
                />
              </DataTableHeaderCell>
              <DataTableHeaderCell width="medium">名称</DataTableHeaderCell>
              <DataTableHeaderCell width="medium">Slug</DataTableHeaderCell>
              <DataTableHeaderCell width="compact">文章数</DataTableHeaderCell>
              <DataTableHeaderCell width="compact">创建时间</DataTableHeaderCell>
              <DataTableHeaderCell width="actions">操作</DataTableHeaderCell>
            </DataTableHeadRow>
          </thead>
          <DataTableBody>
            {tags.map((tag) => (
              <DataTableRow key={tag.id}>
                <DataTableCell width="select" nowrap>
                  <input
                    type="checkbox"
                    checked={selectedIds.includes(tag.id)}
                    onChange={() => toggleTagSelection(tag.id)}
                    className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                    aria-label={`选择标签 ${tag.name}`}
                  />
                </DataTableCell>
                <DataTableCell truncate className="font-medium text-neutral-800 dark:text-neutral-200" title={tag.name}>
                  {tag.name}
                </DataTableCell>
                <DataTableCell truncate title={tag.slug}>{tag.slug}</DataTableCell>
                <DataTableCell nowrap>{tag.article_count}</DataTableCell>
                <DataTableCell nowrap>{formatDate(tag.created_at)}</DataTableCell>
                <DataTableCell nowrap>
                  <div className="inline-flex items-center gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleEdit(tag)}
                      className="text-blue-600 hover:bg-blue-50 dark:text-blue-400 dark:hover:bg-blue-900/30"
                    >
                      编辑
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setDeleteId(tag.id)}
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

      {tagsData && (
        <Pagination
          page={page}
          totalPages={totalPages}
          total={tagsData.total}
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
        title="删除标签"
        message="确定删除这个标签吗？已有文章使用的标签不能删除。"
        onConfirm={() => {
          if (deleteId) deleteMutation.mutate(deleteId);
        }}
        onCancel={() => setDeleteId(null)}
        isConfirming={deleteMutation.isPending}
        isDanger
      />

      <ConfirmModal
        isOpen={confirmBatchDelete}
        title="删除标签"
        message="确定删除选中的标签吗？已有文章使用的标签不能删除。"
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
                <div className="border-b border-neutral-200 px-6 py-4 dark:border-neutral-700">
                  <h3 className="text-xl font-bold text-neutral-800 dark:text-neutral-100">
                    {editingTag ? '编辑标签' : '新建标签'}
                  </h3>
                </div>
                <div className="space-y-5 p-6">
                  <div>
                    <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">名称</label>
                    <TextInput
                      type="text"
                      value={formData.name}
                      onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                      placeholder="AI / 产品 / Go"
                    />
                  </div>
                  <div>
                    <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">Slug</label>
                    <TextInput
                      type="text"
                      value={formData.slug}
                      onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
                      placeholder="ai"
                    />
                  </div>
                </div>
                <div className="flex justify-end gap-3 bg-neutral-50 px-6 py-4 dark:bg-neutral-800/50">
                  <Button variant="secondary" onClick={handleClose}>
                    取消
                  </Button>
                  <Button
                    onClick={() => {
                      if (!formData.name || !formData.slug) {
                        showToast('请填写完整信息', 'error');
                        return;
                      }
                      saveMutation.mutate(formData);
                    }}
                    disabled={saveMutation.isPending}
                  >
                    {saveMutation.isPending ? '保存中...' : '保存'}
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
