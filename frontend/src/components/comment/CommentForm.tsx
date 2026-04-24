import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '@/hooks';
import { commentApi } from '@/api';
import { useUIStore } from '@/store';

interface CommentFormProps {
  articleId: number;
  parentId?: number;
  replyToUserId?: number;
  replyToUsername?: string;
  onSuccess?: () => void;
}

export const CommentForm = ({ 
  articleId, 
  parentId, 
  replyToUserId, 
  replyToUsername,
  onSuccess 
}: CommentFormProps) => {
  const [content, setContent] = useState('');
  const { isAuthenticated } = useAuth();
  const { showToast } = useUIStore();
  const queryClient = useQueryClient();

  const createCommentMutation = useMutation({
    mutationFn: commentApi.createComment,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['comments', articleId] });
      setContent('');
      showToast('评论成功', 'success');
      onSuccess?.();
    },
    onError: () => {
      showToast('评论失败', 'error');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!content.trim()) {
      showToast('请输入评论内容', 'error');
      return;
    }

    createCommentMutation.mutate({
      content: content.trim(),
      articleId,
      parentId,
      replyToUserId,
    });
  };

  if (!isAuthenticated) {
    return (
      <div className="text-center py-8 text-neutral-500 dark:text-neutral-400">
        请先登录后再评论
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        placeholder={replyToUsername ? `回复 @${replyToUsername}...` : '写下你的评论...'}
        className="input min-h-[100px] resize-none dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-100"
        disabled={createCommentMutation.isPending}
      />
      <div className="flex justify-end">
        <button
          type="submit"
          className="btn btn-primary"
          disabled={createCommentMutation.isPending}
        >
          {createCommentMutation.isPending ? '发送中...' : '发送'}
        </button>
      </div>
    </form>
  );
};
