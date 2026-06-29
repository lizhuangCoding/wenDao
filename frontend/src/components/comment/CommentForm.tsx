import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { useAuth } from '@/hooks';
import { commentApi } from '@/api';
import { useUIStore } from '@/store';
import { getApiErrorMessage } from '@/utils/apiError';

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
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const createCommentMutation = useMutation({
    mutationFn: commentApi.createComment,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['comments', articleId] });
      setContent('');
      showToast(t('common.success'), 'success');
      onSuccess?.();
    },
    onError: (error) => {
      showToast(getApiErrorMessage(error, t('common.failed')), 'error');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!content.trim()) {
      showToast(t('comment.enterContent'), 'error');
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
        {t('comment.loginToComment')}
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        placeholder={replyToUsername ? t('comment.replyTo', { username: replyToUsername }) : t('comment.placeholder')}
        className="input min-h-[100px] resize-none dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-100"
        disabled={createCommentMutation.isPending}
      />
      <div className="flex justify-end">
        <button
          type="submit"
          className="btn btn-primary"
          disabled={createCommentMutation.isPending}
        >
          {createCommentMutation.isPending ? t('common.sending') : t('common.submit')}
        </button>
      </div>
    </form>
  );
};
