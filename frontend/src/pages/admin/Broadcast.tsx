import { useMemo, useRef, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Send } from 'lucide-react';
import { notificationApi } from '@/api';
import { PageHeader } from '@/components/common';
import { useNotificationStore, useUIStore } from '@/store';
import { MarkdownWritingStudio } from '@/views/admin/articles/components/MarkdownWritingStudio';

const getContentStats = (content: string) => {
  const trimmed = content.trim();
  const lineCount = content ? content.split('\n').length : 0;
  const cjkCount = (trimmed.match(/[\u4e00-\u9fff]/g) || []).length;
  const wordCount =
    trimmed
      .replace(/[\u4e00-\u9fff]/g, '')
      .split(/\s+/)
      .filter(Boolean).length + cjkCount;

  return {
    characters: content.length,
    lines: lineCount,
    words: wordCount,
    readingMinutes: Math.max(1, Math.ceil(wordCount / 450)),
  };
};

export const Broadcast = () => {
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const { fetchUnreadCount } = useNotificationStore();
  const contentInputRef = useRef<HTMLTextAreaElement>(null);
  const [isWritingFocused, setIsWritingFocused] = useState(false);
  const [formData, setFormData] = useState({
    title: '',
    content: '',
  });
  const contentStats = useMemo(() => getContentStats(formData.content), [formData.content]);

  const broadcastMutation = useMutation({
    mutationFn: notificationApi.broadcast,
    onSuccess: () => {
      showToast('消息已成功广播给所有用户', 'success');
      setFormData({ title: '', content: '' });
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      void fetchUnreadCount();
    },
    onError: (error: any) => {
      showToast(error.message || '广播发送失败', 'error');
    },
  });

  const handleContentPaste = () => {
    // 广播站内信暂不处理图片上传；MarkdownWritingStudio 在此页面隐藏图片按钮。
  };

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
      title: formData.title.trim(),
      content: formData.content.trim(),
    });
  };

  return (
    <div
      className={`${isWritingFocused ? 'max-w-display' : 'max-w-6xl'} mx-auto space-y-6 pb-12 transition-[max-width] duration-300`}
    >
      <PageHeader
        title="编辑站内信"
        description="像写文章一样编写 Markdown 内容，发送后会以站内信形式广播给所有活跃用户。"
        actions={
          <button
            type="button"
            onClick={handleSubmit}
            disabled={broadcastMutation.isPending}
            className="btn btn-primary"
          >
            <Send className="h-4 w-4" />
            {broadcastMutation.isPending ? '发送中...' : '发送广播'}
          </button>
        }
      />

      <section className="rounded-2xl border border-neutral-100 bg-white p-5 shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
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
      </section>

      <MarkdownWritingStudio
        content={formData.content}
        onContentChange={(content) => setFormData({ ...formData, content })}
        textareaRef={contentInputRef}
        onPaste={handleContentPaste}
        onImageUploadClick={() => undefined}
        allowImageUpload={false}
        helperText="支持标题、列表、引用、代码块等 Markdown 内容；发送后用户将在站内信页面阅读。"
        placeholder="编写要发送给用户的站内信内容..."
        contentStats={contentStats}
        lastSavedTime={null}
        isAutoSaving={false}
        isImmersive={isWritingFocused}
        onImmersiveChange={setIsWritingFocused}
      />
    </div>
  );
};
