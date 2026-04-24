import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { Comment } from '@/types';
import { formatDate } from '@/utils';
import { CommentForm } from './CommentForm';

interface CommentItemProps {
  comment: Comment;
  articleId: number;
  isReply?: boolean;
}

export const CommentItem = ({ comment, articleId, isReply = false }: CommentItemProps) => {
  const { t } = useTranslation();
  const [showReplyForm, setShowReplyForm] = useState(false);

  return (
    <div className={`${isReply ? 'py-2' : 'py-4'}`}>
      {/* 评论主体 */}
      <div className={`${isReply ? 'bg-neutral-50/50 dark:bg-neutral-800/50 rounded-lg p-3 border-l-2 border-primary-100' : 'bg-neutral-50 dark:bg-neutral-800 rounded-lg p-4'}`}>
        {/* 评论头部 */}
        <div className="flex items-center gap-3 mb-2">
          {/* 用户头像 */}
          <div className="w-8 h-8 rounded-full overflow-hidden bg-neutral-200 dark:bg-neutral-700 flex-shrink-0 border border-neutral-100 dark:border-neutral-600">
            <img
              src={comment.user.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${comment.user.username}`}
              alt={comment.user.username}
              className="w-full h-full object-cover"
            />
          </div>
          <div className="flex flex-col">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="font-medium text-neutral-700 dark:text-neutral-200">{comment.user.username}</span>
              {comment.reply_to_user && (
                <div className="flex items-center gap-1">
                  <span className="text-xs text-neutral-400 dark:text-neutral-500">{t('article.reply')}</span>
                  <span className="text-xs font-bold text-primary-600 dark:text-primary-400">@{comment.reply_to_user.username}</span>
                </div>
              )}
            </div>
            <span className="text-xs text-neutral-500 dark:text-neutral-400">{formatDate(comment.created_at)}</span>
          </div>
        </div>

        {/* 评论内容 */}
        <p className="text-neutral-800 dark:text-neutral-100 mb-2">{comment.content}</p>

        {/* 回复按钮 - 始终显示回复按钮，支持二级评论继续回复 */}
        <button
          onClick={() => setShowReplyForm(!showReplyForm)}
          className="text-xs font-bold text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300 uppercase tracking-wider"
        >
          {showReplyForm ? t('article.cancelReply') : t('article.reply')}
        </button>
      </div>

      {/* 回复表单 */}
      {showReplyForm && (
        <div className={`mt-4 ${isReply ? 'ml-4' : 'ml-8'}`}>
          <CommentForm
            articleId={articleId}
            parentId={comment.parent_id || comment.id} // 如果是回复，使用相同的 parent_id；如果是直评，使用当前 ID 作为 parent
            replyToUserId={comment.user_id} // 被回复人的 ID
            replyToUsername={comment.user.username} // 被回复人的用户名
            onSuccess={() => setShowReplyForm(false)}
          />
        </div>
      )}

      {/* 子评论 - 仅一级评论渲染其 replies 列表 */}
      {!isReply && comment.replies && comment.replies.length > 0 && (
        <div className="mt-4 ml-6 relative">
          {/* 视觉引导线 */}
          <div className="absolute left-0 top-0 bottom-0 w-0.5 bg-gradient-to-b from-primary-200 to-transparent opacity-50"></div>
          <div className="space-y-1">
            {comment.replies.map((reply) => (
              <div key={reply.id} className="relative pl-4">
                {/* 连接线 */}
                <div className="absolute left-0 top-6 w-3 h-px bg-primary-200 opacity-50"></div>
                <CommentItem comment={reply} articleId={articleId} isReply={true} />
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
