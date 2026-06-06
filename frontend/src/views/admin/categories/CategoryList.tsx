import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Plus } from 'lucide-react';
import { categoryApi } from '@/api';
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
import { motion, AnimatePresence } from 'framer-motion';
import type { Category } from '@/types';

type CategoryFormData = {
  name: string;
  slug: string;
  description: string;
};

export const CategoryList = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const [pageSize, setPageSize] = useState(10);
  const [page, setPage] = useState(1);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingCategory, setEditingCategory] = useState<Category | null>(null);
  const [deleteId, setDeleteId] = useState<number | null>(null);
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [confirmBatchDelete, setConfirmBatchDelete] = useState(false);
  const [formData, setFormData] = useState<CategoryFormData>({ name: '', slug: '', description: '' });

  const {
    data: categoriesData,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['admin-categories', page, pageSize],
    queryFn: () => categoryApi.getAdminCategories({ page, pageSize }),
  });

  const categories = categoriesData?.data ?? [];
  const totalPages = Math.max(1, categoriesData?.totalPages ?? 1);
  const currentPageIds = categories.map((category) => category.id);
  const allCurrentPageSelected =
    currentPageIds.length > 0 && currentPageIds.every((id) => selectedIds.includes(id));

  const invalidateCategories = () => {
    queryClient.invalidateQueries({ queryKey: ['admin-categories'] });
    queryClient.invalidateQueries({ queryKey: ['categories'] });
  };

  const saveMutation = useMutation({
    mutationFn: (data: CategoryFormData) =>
      editingCategory
        ? categoryApi.updateCategory(editingCategory.id, data)
        : categoryApi.createCategory(data),
    onSuccess: () => {
      showToast(editingCategory ? '分类已更新' : '分类已创建', 'success');
      handleClose();
      invalidateCategories();
    },
    onError: (err: any) => {
      showToast(err.message || '操作失败', 'error');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => categoryApi.deleteCategory(id),
    onSuccess: () => {
      showToast('分类已删除', 'success');
      setDeleteId(null);
      invalidateCategories();
      if (page > 1 && categories.length <= 1) {
        setPage((currentPage) => Math.max(1, currentPage - 1));
      }
    },
    onError: (err: any) => {
      showToast(err.message || '删除失败，该分类下可能还有文章', 'error');
    },
  });

  const batchDeleteMutation = useMutation({
    mutationFn: (ids: number[]) => categoryApi.batchDeleteCategories(ids),
    onSuccess: (result) => {
      showToast(`已删除 ${result.deleted_count} 个分类`, 'success');
      setSelectedIds([]);
      setConfirmBatchDelete(false);
      invalidateCategories();
      if (page > 1 && categories.length > 0 && selectedIds.length >= categories.length) {
        setPage((currentPage) => Math.max(1, currentPage - 1));
      }
    },
    onError: (err: any) => {
      showToast(err.message || '批量删除失败，分类下可能还有文章', 'error');
    },
  });

  const handleEdit = (category: Category) => {
    setEditingCategory(category);
    setFormData({
      name: category.name,
      slug: category.slug,
      description: category.description || '',
    });
    setIsModalOpen(true);
  };

  const handleClose = () => {
    setIsModalOpen(false);
    setEditingCategory(null);
    setFormData({ name: '', slug: '', description: '' });
  };

  const toggleCategorySelection = (id: number) => {
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
        title={t('admin.categoryManagement')}
        actions={
          <Button size="lg" onClick={() => setIsModalOpen(true)}>
            <Plus className="h-5 w-5" />
            {t('admin.newCategory')}
          </Button>
        }
      />

      <BulkActionBar
        selectedCount={selectedIds.length}
        onDelete={() => setConfirmBatchDelete(true)}
        onClear={() => setSelectedIds([])}
        isDeleting={batchDeleteMutation.isPending}
        deleteLabel="删除分类"
      />

      {isError ? (
        <ErrorState message={(error as any)?.message || '分类列表加载失败'} onRetry={() => refetch()} />
      ) : (
        <DataTable
          emptyState={
            categories.length === 0 ? (
              <EmptyState title="暂无分类" description="还没有创建分类。" className="m-6" />
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
                      aria-label="选择当前页分类"
                    />
              </DataTableHeaderCell>
              <DataTableHeaderCell>{t('admin.name')}</DataTableHeaderCell>
              <DataTableHeaderCell>{t('admin.slug')}</DataTableHeaderCell>
              <DataTableHeaderCell>{t('admin.articleCount')}</DataTableHeaderCell>
              <DataTableHeaderCell>{t('admin.createdAt')}</DataTableHeaderCell>
              <DataTableHeaderCell align="right">{t('admin.actions')}</DataTableHeaderCell>
            </DataTableHeadRow>
              </thead>
          <DataTableBody>
            {categories.map((category) => (
              <DataTableRow
                    key={category.id}
                  >
                <DataTableCell>
                      <input
                        type="checkbox"
                        checked={selectedIds.includes(category.id)}
                        onChange={() => toggleCategorySelection(category.id)}
                        className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                        aria-label={`选择分类 ${category.name}`}
                      />
                </DataTableCell>
                <DataTableCell className="font-medium text-neutral-800 dark:text-neutral-200">{category.name}</DataTableCell>
                <DataTableCell>{category.slug}</DataTableCell>
                <DataTableCell>{category.article_count}</DataTableCell>
                <DataTableCell>{formatDate(category.created_at)}</DataTableCell>
                <DataTableCell align="right">
                      <div className="flex items-center justify-end gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                          onClick={() => handleEdit(category)}
                      className="text-blue-600 hover:bg-blue-50 dark:text-blue-400 dark:hover:bg-blue-900/30"
                        >
                          {t('admin.edit')}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                          onClick={() => setDeleteId(category.id)}
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

      {categoriesData && (
        <Pagination
          page={page}
          totalPages={totalPages}
          total={categoriesData?.total}
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
        title={t('admin.categories')}
        message={t('admin.confirmDeleteCategory')}
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
        title="批量删除分类"
        message={`确定删除选中的 ${selectedIds.length} 个分类吗？有文章的分类不能删除。`}
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
                  {editingCategory ? t('admin.editCategory') : t('admin.newCategory')}
                </h3>
              </div>
              <div className="space-y-5 p-6">
                <div>
                  <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">{t('admin.name')}</label>
                  <TextInput
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="如：Go 语言"
                  />
                </div>
                <div>
                  <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">{t('admin.slug')}</label>
                  <TextInput
                    type="text"
                    value={formData.slug}
                    onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
                    placeholder="如：golang"
                  />
                </div>
                <div>
                  <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">{t('admin.description')}</label>
                  <textarea
                    className="w-full resize-none rounded-xl border border-neutral-200 bg-neutral-50 px-4 py-3 text-neutral-800 transition-all focus:outline-none focus:ring-2 focus:ring-primary-500 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
                    rows={3}
                    value={formData.description}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    placeholder="分类描述..."
                  />
                </div>
              </div>
              <div className="flex justify-end gap-3 bg-neutral-50 px-6 py-4 dark:bg-neutral-800/50">
                <Button variant="secondary" onClick={handleClose}>
                  {t('admin.cancel')}
                </Button>
                <Button
                  onClick={() => {
                    if (!formData.name || !formData.slug) {
                      showToast(t('admin.pleaseFillComplete'), 'error');
                      return;
                    }
                    saveMutation.mutate(formData);
                  }}
                  disabled={saveMutation.isPending}
                >
                  {saveMutation.isPending ? t('admin.saving') : t('admin.save')}
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
