import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { commentApi } from '@/api';
import { CommentForm } from './CommentForm';
import { CommentItem } from './CommentItem';
import { ErrorState, Loading } from '@/components/common';
import { getApiErrorMessage } from '@/utils/apiError';

interface CommentListProps {
  articleId: number;
  totalCommentCount?: number;
}

export const CommentList = ({ articleId, totalCommentCount }: CommentListProps) => {
  const { t } = useTranslation();
  const [sort, setSort] = useState<'newest' | 'hottest'>('newest');
  const {
    data: comments,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['comments', articleId, sort],
    queryFn: () => commentApi.getComments(articleId, sort),
  });

  if (isLoading) {
    return <Loading />;
  }

  // 使用传入的总评论数，如果没有则使用API返回的长度
  const displayCount = totalCommentCount ?? comments?.length ?? 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-2xl font-semibold text-neutral-700 dark:text-neutral-200">
          {t('comment.title', { count: displayCount })}
        </h3>
        <div className="flex items-center bg-neutral-100 dark:bg-neutral-800 rounded-lg p-1">
          <button
            type="button"
            onClick={() => setSort('newest')}
            className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
              sort === 'newest'
                ? 'bg-white dark:bg-neutral-700 text-neutral-900 dark:text-neutral-100 shadow-sm'
                : 'text-neutral-500 dark:text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-300'
            }`}
          >
            {t('comment.newest')}
          </button>
          <button
            type="button"
            onClick={() => setSort('hottest')}
            className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
              sort === 'hottest'
                ? 'bg-white dark:bg-neutral-700 text-neutral-900 dark:text-neutral-100 shadow-sm'
                : 'text-neutral-500 dark:text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-300'
            }`}
          >
            {t('comment.hottest')}
          </button>
        </div>
      </div>

      {/* 评论表单 */}
      <CommentForm articleId={articleId} />

      {isError ? (
        <ErrorState
          title={t('comment.loadFailed')}
          message={getApiErrorMessage(error, t('comment.loadFailedDescription'))}
          onRetry={() => refetch()}
        />
      ) : (
        <div className="divide-y divide-neutral-200 dark:divide-neutral-700">
          {comments?.map((comment) => (
            <CommentItem key={comment.id} comment={comment} articleId={articleId} />
          ))}
        </div>
      )}

      {!isError && comments?.length === 0 && (
        <div className="text-center py-8 text-neutral-500 dark:text-neutral-400">
          {t('comment.noCommentsYet')}
        </div>
      )}
    </div>
  );
};
