import { useQuery } from '@tanstack/react-query';
import { commentApi } from '@/api';
import { CommentForm } from './CommentForm';
import { CommentItem } from './CommentItem';
import { Loading } from '@/components/common';

interface CommentListProps {
  articleId: number;
  totalCommentCount?: number;
}

export const CommentList = ({ articleId, totalCommentCount }: CommentListProps) => {
  const { data: comments, isLoading } = useQuery({
    queryKey: ['comments', articleId],
    queryFn: () => commentApi.getComments(articleId),
  });

  if (isLoading) {
    return <Loading />;
  }

  // 使用传入的总评论数，如果没有则使用API返回的长度
  const displayCount = totalCommentCount ?? comments?.length ?? 0;

  return (
    <div className="space-y-6">
      <h3 className="text-2xl font-semibold text-neutral-700 dark:text-neutral-200 mb-6">
        评论 {displayCount}
      </h3>

      {/* 评论表单 */}
      <CommentForm articleId={articleId} />

      {/* 评论列表 */}
      <div className="divide-y divide-neutral-200 dark:divide-neutral-700">
        {comments?.map((comment) => (
          <CommentItem key={comment.id} comment={comment} articleId={articleId} />
        ))}
      </div>

      {comments?.length === 0 && (
        <div className="text-center py-8 text-neutral-500 dark:text-neutral-400">
          暂无评论，快来发表第一条评论吧
        </div>
      )}
    </div>
  );
};
