import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Plus } from 'lucide-react';
import { categoryApi } from '@/api';
import { Loading, ConfirmModal, Pagination, EmptyState, ErrorState, BulkActionBar } from '@/components/common';
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
  const pageSize = 10;
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
    queryKey: ['admin-categories', page],
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
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex items-center justify-between gap-4"
      >
        <h1 className="text-3xl font-serif font-bold text-neutral-800 dark:text-neutral-100">
          {t('admin.categoryManagement')}
        </h1>
        <button
          type="button"
          onClick={() => setIsModalOpen(true)}
          className="inline-flex items-center gap-2 rounded-xl bg-primary-500 px-5 py-2.5 font-medium text-white shadow-md transition-all hover:bg-primary-600 hover:shadow-lg"
        >
          <Plus className="h-5 w-5" />
          {t('admin.newCategory')}
        </button>
      </motion.div>

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
                      aria-label="选择当前页分类"
                    />
                  </th>
                  <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.name')}</th>
                  <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.slug')}</th>
                  <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.articleCount')}</th>
                  <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.createdAt')}</th>
                  <th className="px-6 py-4 text-right text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100 dark:divide-neutral-800">
                {categories.map((category, index) => (
                  <motion.tr
                    key={category.id}
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    transition={{ delay: index * 0.03 }}
                    className="transition-colors hover:bg-neutral-50 dark:hover:bg-neutral-800/50"
                  >
                    <td className="px-6 py-4">
                      <input
                        type="checkbox"
                        checked={selectedIds.includes(category.id)}
                        onChange={() => toggleCategorySelection(category.id)}
                        className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                        aria-label={`选择分类 ${category.name}`}
                      />
                    </td>
                    <td className="px-6 py-4 font-medium text-neutral-800 dark:text-neutral-200">{category.name}</td>
                    <td className="px-6 py-4 text-sm text-neutral-500 dark:text-neutral-400">{category.slug}</td>
                    <td className="px-6 py-4 text-sm text-neutral-500 dark:text-neutral-400">{category.article_count}</td>
                    <td className="px-6 py-4 text-sm text-neutral-500 dark:text-neutral-400">{formatDate(category.created_at)}</td>
                    <td className="px-6 py-4 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <button
                          type="button"
                          onClick={() => handleEdit(category)}
                          className="rounded-lg px-3 py-1.5 text-sm text-blue-600 transition-colors hover:bg-blue-50 dark:text-blue-400 dark:hover:bg-blue-900/30"
                        >
                          {t('admin.edit')}
                        </button>
                        <button
                          type="button"
                          onClick={() => setDeleteId(category.id)}
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
          {categories.length === 0 && (
            <EmptyState title="暂无分类" description="还没有创建分类。" className="m-6" />
          )}
        </motion.div>
      )}

      {categoriesData && (
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
        title={t('admin.categories')}
        message={t('admin.confirmDeleteCategory')}
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
        title="批量删除分类"
        message={`确定删除选中的 ${selectedIds.length} 个分类吗？有文章的分类不能删除。`}
        confirmText="删除"
        onConfirm={() => batchDeleteMutation.mutate(selectedIds)}
        onCancel={() => setConfirmBatchDelete(false)}
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
              className="w-full max-w-md overflow-hidden rounded-2xl border border-neutral-100 bg-white shadow-xl dark:border-neutral-800 dark:bg-neutral-900"
            >
              <div className="border-b border-neutral-100 px-6 py-4 dark:border-neutral-800">
                <h3 className="text-xl font-bold text-neutral-800 dark:text-neutral-100">
                  {editingCategory ? t('admin.editCategory') : t('admin.newCategory')}
                </h3>
              </div>
              <div className="space-y-5 p-6">
                <div>
                  <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">{t('admin.name')}</label>
                  <input
                    type="text"
                    className="w-full rounded-xl border border-neutral-200 bg-neutral-50 px-4 py-3 text-neutral-800 transition-all focus:outline-none focus:ring-2 focus:ring-primary-500 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="如：Go 语言"
                  />
                </div>
                <div>
                  <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">{t('admin.slug')}</label>
                  <input
                    type="text"
                    className="w-full rounded-xl border border-neutral-200 bg-neutral-50 px-4 py-3 text-neutral-800 transition-all focus:outline-none focus:ring-2 focus:ring-primary-500 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
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
                <button
                  type="button"
                  onClick={handleClose}
                  className="rounded-xl bg-neutral-100 px-5 py-2.5 font-medium text-neutral-700 transition-all hover:bg-neutral-200 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
                >
                  {t('admin.cancel')}
                </button>
                <button
                  type="button"
                  onClick={() => {
                    if (!formData.name || !formData.slug) {
                      showToast(t('admin.pleaseFillComplete'), 'error');
                      return;
                    }
                    saveMutation.mutate(formData);
                  }}
                  disabled={saveMutation.isPending}
                  className="rounded-xl bg-primary-500 px-5 py-2.5 font-medium text-white transition-all hover:bg-primary-600 disabled:opacity-50"
                >
                  {saveMutation.isPending ? t('admin.saving') : t('admin.save')}
                </button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};
