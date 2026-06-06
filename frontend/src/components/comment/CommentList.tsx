import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { commentApi } from '@/api';
import { CommentForm } from './CommentForm';
import { CommentItem } from './CommentItem';
import { ErrorState, Loading } from '@/components/common';

interface CommentListProps {
  articleId: number;
  totalCommentCount?: number;
}

export const CommentList = ({ articleId, totalCommentCount }: CommentListProps) => {
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
          评论 {displayCount}
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
            最新
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
            最热
          </button>
        </div>
      </div>

      {/* 评论表单 */}
      <CommentForm articleId={articleId} />

      {isError ? (
        <ErrorState
          title="评论加载失败"
          message={(error as any)?.message || '无法加载评论，请稍后重试。'}
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
          暂无评论，快来发表第一条评论吧
        </div>
      )}
    </div>
  );
};
