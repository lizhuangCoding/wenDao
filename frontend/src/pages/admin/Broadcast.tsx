import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { notificationApi } from '@/api';
import { useUIStore } from '@/store';

export const Broadcast = () => {
  const { showToast } = useUIStore();
  const [formData, setFormData] = useState({
    title: '',
    content: '',
    link_url: '',
  });

  const broadcastMutation = useMutation({
    mutationFn: notificationApi.broadcast,
    onSuccess: () => {
      showToast('消息已成功广播给所有用户', 'success');
      setFormData({ title: '', content: '', link_url: '' });
    },
    onError: (error: any) => {
      showToast(error.message || '广播发送失败', 'error');
    },
  });

  const handleSubmit = () => {
    if (!formData.title.trim()) {
      showToast('请输入通知标题', 'error');
      return;
    }
    if (!formData.content.trim()) {
      showToast('请输入通知内容', 'error');
      return;
    }
    broadcastMutation.mutate({
      title: formData.title,
      content: formData.content,
      link_url: formData.link_url || undefined,
    });
  };

  return (
    <div className="max-w-2xl">
      <h1 className="text-2xl font-bold text-neutral-700 dark:text-neutral-100 mb-8">消息广播</h1>

      <div className="space-y-6 rounded-xl border border-neutral-100 bg-white p-8 shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
        <div>
          <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
            通知标题
          </label>
          <input
            type="text"
            className="input w-full"
            value={formData.title}
            onChange={(e) => setFormData({ ...formData, title: e.target.value })}
            placeholder="输入通知标题"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
            通知内容
          </label>
          <textarea
            className="input w-full h-32 py-2"
            value={formData.content}
            onChange={(e) => setFormData({ ...formData, content: e.target.value })}
            placeholder="输入通知内容（支持纯文本）"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
            跳转链接 <span className="text-neutral-400 font-normal">（可选）</span>
          </label>
          <input
            type="text"
            className="input w-full"
            value={formData.link_url}
            onChange={(e) => setFormData({ ...formData, link_url: e.target.value })}
            placeholder="例如：/article/your-article-slug"
          />
          <p className="mt-1 text-xs text-neutral-400 dark:text-neutral-500">
            用户点击通知后将跳转到此链接
          </p>
        </div>

        {/* Preview */}
        <div className="rounded-xl border border-primary-100 dark:border-primary-900/30 bg-primary-50/50 dark:bg-primary-900/10 p-5">
          <h3 className="text-xs font-bold text-neutral-400 dark:text-neutral-500 uppercase tracking-wider mb-3">
            通知预览
          </h3>
          <div className="flex items-start gap-4">
            <div className="flex-1">
              <h4 className="text-sm font-semibold text-neutral-700 dark:text-neutral-200">
                {formData.title || '通知标题'}
              </h4>
              <p className="text-sm text-neutral-500 dark:text-neutral-400 mt-1">
                {formData.content || '通知内容将显示在这里'}
              </p>
            </div>
            <span className="px-2 py-0.5 text-[10px] font-bold text-primary-600 dark:text-primary-400 bg-primary-100 dark:bg-primary-900/30 rounded-full">
              新
            </span>
          </div>
        </div>

        <button
          type="button"
          onClick={handleSubmit}
          disabled={broadcastMutation.isPending}
          className="btn btn-primary w-full"
        >
          {broadcastMutation.isPending ? '发送中...' : '发送广播'}
        </button>
      </div>
    </div>
  );
};
