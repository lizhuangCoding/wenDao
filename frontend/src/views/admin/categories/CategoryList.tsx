import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { categoryApi } from '@/api';
import { Loading, ConfirmModal } from '@/components/common';
import { formatDate } from '@/utils';
import { useUIStore } from '@/store';
import { motion, AnimatePresence } from 'framer-motion';

export const CategoryList = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingCategory, setEditingCategory] = useState<any>(null);
  const [deleteId, setDeleteId] = useState<number | null>(null);
  const [formData, setFormData] = useState({ name: '', slug: '', description: '' });

  const { data: categories, isLoading } = useQuery({
    queryKey: ['categories'],
    queryFn: categoryApi.getCategories,
  });

  const saveMutation = useMutation({
    mutationFn: (data: typeof formData) =>
      editingCategory
        ? categoryApi.updateCategory(editingCategory.id, data)
        : categoryApi.createCategory(data),
    onSuccess: () => {
      showToast(editingCategory ? '分类已更新' : '分类已创建', 'success');
      setIsModalOpen(false);
      setEditingCategory(null);
      setFormData({ name: '', slug: '', description: '' });
      queryClient.invalidateQueries({ queryKey: ['categories'] });
    },
    onError: (error: any) => {
      showToast(error.message || '操作失败', 'error');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => categoryApi.deleteCategory(id),
    onSuccess: () => {
      showToast('分类已删除', 'success');
      queryClient.invalidateQueries({ queryKey: ['categories'] });
    },
    onError: () => {
      showToast('删除失败，该分类下可能还有文章', 'error');
    },
  });

  const handleEdit = (category: any) => {
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

  if (isLoading) return <Loading />;

  return (
    <div className="space-y-6">
      {/* 标题和操作栏 */}
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex justify-between items-center"
      >
        <h1 className="text-3xl font-serif font-bold text-neutral-800 dark:text-neutral-100">
          {t('admin.categoryManagement')}
        </h1>
        <button
          onClick={() => setIsModalOpen(true)}
          className="flex items-center gap-2 px-5 py-2.5 bg-primary-500 text-white rounded-xl font-medium hover:bg-primary-600 transition-all shadow-md hover:shadow-lg"
        >
          <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          {t('admin.newCategory')}
        </button>
      </motion.div>

      {/* 分类表格 */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.1 }}
        className="bg-white dark:bg-neutral-900 rounded-2xl shadow-sm border border-neutral-100 dark:border-neutral-800 overflow-hidden"
      >
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="bg-neutral-50 dark:bg-neutral-800/50 border-b border-neutral-100 dark:border-neutral-800">
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.name')}</th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.slug')}</th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.articleCount')}</th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400">{t('admin.createdAt')}</th>
              <th className="px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400 text-right">{t('admin.actions')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-neutral-100 dark:divide-neutral-800">
            {categories?.map((category, index) => (
              <motion.tr
                key={category.id}
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ delay: index * 0.05 }}
                className="hover:bg-neutral-50 dark:hover:bg-neutral-800/50 transition-colors"
              >
                <td className="px-6 py-4 font-medium text-neutral-800 dark:text-neutral-200">{category.name}</td>
                <td className="px-6 py-4 text-sm text-neutral-500 dark:text-neutral-400">{category.slug}</td>
                <td className="px-6 py-4 text-sm text-neutral-500 dark:text-neutral-400">{category.article_count}</td>
                <td className="px-6 py-4 text-sm text-neutral-500 dark:text-neutral-400">{formatDate(category.created_at)}</td>
                <td className="px-6 py-4 text-right">
                  <div className="flex items-center justify-end gap-2">
                    <button
                      onClick={() => handleEdit(category)}
                      className="px-3 py-1.5 text-sm text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/30 rounded-lg transition-colors"
                    >
                      {t('admin.edit')}
                    </button>
                    <button
                      onClick={() => setDeleteId(category.id)}
                      className="px-3 py-1.5 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/30 rounded-lg transition-colors"
                    >
                      {t('admin.delete')}
                    </button>
                    </div>
                    </td>
                    </motion.tr>
                    ))}
                    </tbody>
                    </table>
                    </motion.div>

                    <ConfirmModal
                    isOpen={deleteId !== null}
                    title={t('admin.categories')}
                    message={t('admin.confirmDeleteCategory')}
                    onConfirm={() => {
                    if (deleteId) {
                    deleteMutation.mutate(deleteId);
                    setDeleteId(null);
                    }
                    }}
                    onCancel={() => setDeleteId(null)}
                    isDanger
                    />

                    <AnimatePresence>
                    ...
        {isModalOpen && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm"
          >
            <motion.div
              initial={{ scale: 0.95, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.95, opacity: 0 }}
              className="bg-white dark:bg-neutral-900 rounded-2xl shadow-xl w-full max-w-md overflow-hidden border border-neutral-100 dark:border-neutral-800"
            >
              <div className="px-6 py-4 border-b border-neutral-100 dark:border-neutral-800">
                <h3 className="text-xl font-bold text-neutral-800 dark:text-neutral-100">
                  {editingCategory ? t('admin.editCategory') : t('admin.newCategory')}
                </h3>
              </div>
              <div className="p-6 space-y-5">
                <div>
                  <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">{t('admin.name')}</label>
                  <input
                    type="text"
                    className="w-full px-4 py-3 bg-neutral-50 dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 rounded-xl text-neutral-800 dark:text-neutral-200 focus:outline-none focus:ring-2 focus:ring-primary-500 transition-all"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="如：Go 语言"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">{t('admin.slug')}</label>
                  <input
                    type="text"
                    className="w-full px-4 py-3 bg-neutral-50 dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 rounded-xl text-neutral-800 dark:text-neutral-200 focus:outline-none focus:ring-2 focus:ring-primary-500 transition-all"
                    value={formData.slug}
                    onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
                    placeholder="如：golang"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">{t('admin.description')}</label>
                  <textarea
                    className="w-full px-4 py-3 bg-neutral-50 dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700 rounded-xl text-neutral-800 dark:text-neutral-200 focus:outline-none focus:ring-2 focus:ring-primary-500 transition-all resize-none"
                    rows={3}
                    value={formData.description}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    placeholder="分类描述..."
                  />
                </div>
              </div>
              <div className="px-6 py-4 bg-neutral-50 dark:bg-neutral-800/50 flex justify-end gap-3">
                <button
                  onClick={handleClose}
                  className="px-5 py-2.5 bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 rounded-xl font-medium hover:bg-neutral-200 dark:hover:bg-neutral-700 transition-all"
                >
                  {t('admin.cancel')}
                </button>
                <button
                  onClick={() => {
                    if (!formData.name || !formData.slug) {
                      showToast(t('admin.pleaseFillComplete'), 'error');
                      return;
                    }
                    saveMutation.mutate(formData);
                  }}
                  disabled={saveMutation.isPending}
                  className="px-5 py-2.5 bg-primary-500 text-white rounded-xl font-medium hover:bg-primary-600 disabled:opacity-50 transition-all"
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