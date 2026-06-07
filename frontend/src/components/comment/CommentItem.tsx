import { useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import type { Comment } from '@/types';
import { formatDate } from '@/utils';
import { commentApi } from '@/api';
import { CommentForm } from './CommentForm';

interface CommentItemProps {
  comment: Comment;
  articleId: number;
  isReply?: boolean;
}

type CommentVote = 'like' | 'dislike' | null;

const getStoredCommentVote = (key: string): CommentVote => {
  const storedVote = localStorage.getItem(key);
  if (storedVote === 'like' || storedVote === 'dislike') return storedVote;
  if (storedVote === '1') return 'like';
  return null;
};

const DefaultDeletedUserAvatar = () => (
  <div aria-hidden="true" className="relative h-full w-full bg-neutral-300 dark:bg-neutral-600">
    <div className="absolute left-1/2 top-[6px] h-[9px] w-[9px] -translate-x-1/2 rounded-full bg-neutral-50 dark:bg-neutral-300" />
    <div className="absolute left-1/2 bottom-[3px] h-[14px] w-[20px] -translate-x-1/2 rounded-t-full bg-neutral-50 dark:bg-neutral-300" />
  </div>
);

export const CommentItem = ({ comment, articleId, isReply = false }: CommentItemProps) => {
  const { t } = useTranslation();
  const [showReplyForm, setShowReplyForm] = useState(false);
  const votedKey = `comment_vote_${comment.id}`;
  const [vote, setVote] = useState<CommentVote>(() => getStoredCommentVote(votedKey));
  const [isVotePending, setIsVotePending] = useState(false);
  const [optLikeCount, setOptLikeCount] = useState(comment.like_count || 0);
  const [optDislikeCount, setOptDislikeCount] = useState(comment.dislike_count || 0);

  const applyVoteChange = useCallback((previousVote: CommentVote, nextVote: CommentVote) => {
    if (previousVote === nextVote) return;

    if (previousVote === 'like') setOptLikeCount((count) => Math.max(0, count - 1));
    if (previousVote === 'dislike') setOptDislikeCount((count) => Math.max(0, count - 1));
    if (nextVote === 'like') setOptLikeCount((count) => count + 1);
    if (nextVote === 'dislike') setOptDislikeCount((count) => count + 1);

    setVote(nextVote);
    if (nextVote) {
      localStorage.setItem(votedKey, nextVote);
    } else {
      localStorage.removeItem(votedKey);
    }
  }, [votedKey]);

  const handleLike = useCallback(async () => {
    if (isVotePending || vote === 'dislike') return;

    const previousVote = vote;
    const nextVote: CommentVote = vote === 'like' ? null : 'like';
    applyVoteChange(previousVote, nextVote);
    setIsVotePending(true);
    try {
      if (nextVote === 'like') {
        await commentApi.likeComment(comment.id);
      } else {
        await commentApi.unlikeComment(comment.id);
      }
    } catch {
      applyVoteChange(nextVote, previousVote);
    } finally {
      setIsVotePending(false);
    }
  }, [applyVoteChange, comment.id, isVotePending, vote]);

  const handleDislike = useCallback(async () => {
    if (isVotePending || vote === 'like') return;

    const previousVote = vote;
    const nextVote: CommentVote = vote === 'dislike' ? null : 'dislike';
    applyVoteChange(previousVote, nextVote);
    setIsVotePending(true);
    try {
      if (nextVote === 'dislike') {
        await commentApi.dislikeComment(comment.id);
      } else {
        await commentApi.undislikeComment(comment.id);
      }
    } catch {
      applyVoteChange(nextVote, previousVote);
    } finally {
      setIsVotePending(false);
    }
  }, [applyVoteChange, comment.id, isVotePending, vote]);

  const user = comment.user;
  const isDeletedUser = !user;
  const username = user?.username || t('common.deletedUser');
  const commentUser = {
    username,
    avatarUrl: user?.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${encodeURIComponent(username)}`,
  };

  return (
    <div className={`${isReply ? 'py-2' : 'py-4'}`}>
      {/* 评论主体 */}
      <div className={`${isReply ? 'bg-neutral-50/50 dark:bg-neutral-800/50 rounded-lg p-3 border-l-2 border-primary-100' : 'bg-neutral-50 dark:bg-neutral-800 rounded-lg p-4'}`}>
        {/* 评论头部 */}
        <div className="flex items-center gap-3 mb-2">
          {/* 用户头像 */}
          <div className="w-8 h-8 rounded-full overflow-hidden bg-neutral-200 dark:bg-neutral-700 flex items-center justify-center flex-shrink-0 border border-neutral-100 dark:border-neutral-600">
            {isDeletedUser ? (
              <DefaultDeletedUserAvatar />
            ) : (
              <img
                src={commentUser.avatarUrl}
                alt={commentUser.username}
                className="w-full h-full object-cover"
              />
            )}
          </div>
          <div className="flex flex-col">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="font-medium text-neutral-700 dark:text-neutral-200">{commentUser.username}</span>
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

        {/* 操作按钮 */}
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={handleLike}
            disabled={isVotePending || vote === 'dislike'}
            className={`text-xs font-medium flex items-center gap-1 transition-colors ${
              vote === 'like'
                ? 'text-primary-600 dark:text-primary-400'
                : vote === 'dislike' || isVotePending
                ? 'text-neutral-400 dark:text-neutral-500 cursor-not-allowed'
                : 'text-neutral-500 dark:text-neutral-400 hover:text-primary-600 dark:hover:text-primary-400'
            }`}
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5" viewBox="0 0 20 20" fill="currentColor">
              <path d="M2 10.5a1.5 1.5 0 113 0v6a1.5 1.5 0 01-3 0v-6zM6 10.333v5.43a2 2 0 001.106 1.79l.05.025A4 4 0 008.943 18h5.416a2 2 0 001.962-1.608l1.2-6A2 2 0 0015.56 8H12V4a2 2 0 00-2-2 1 1 0 00-1 1v.667a4 4 0 01-.8 2.4L6.8 7.933a4 4 0 00-.8 2.4z" />
            </svg>
            <span>{optLikeCount}</span>
          </button>
          <button
            type="button"
            onClick={handleDislike}
            disabled={isVotePending || vote === 'like'}
            className={`text-xs font-medium flex items-center gap-1 transition-colors ${
              vote === 'dislike'
                ? 'text-red-500 dark:text-red-400'
                : vote === 'like' || isVotePending
                ? 'text-neutral-400 dark:text-neutral-500 cursor-not-allowed'
                : 'text-neutral-500 dark:text-neutral-400 hover:text-red-500 dark:hover:text-red-400'
            }`}
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5" viewBox="0 0 20 20" fill="currentColor">
              <path d="M18 9.5a1.5 1.5 0 11-3 0v-6a1.5 1.5 0 013 0v6zM14 9.667v-5.43a2 2 0 00-1.105-1.79l-.05-.025A4 4 0 0011.055 2H5.64a2 2 0 00-1.962 1.608l-1.2 6A2 2 0 004.44 12H8v4a2 2 0 002 2 1 1 0 001-1v-.667a4 4 0 01.8-2.4l1.4-1.866a4 4 0 00.8-2.4z" />
            </svg>
            <span>{optDislikeCount}</span>
          </button>
          {/* 回复按钮 */}
          <button
            type="button"
            onClick={() => setShowReplyForm(!showReplyForm)}
            className="text-xs font-bold text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300 uppercase tracking-wider"
          >
            {showReplyForm ? t('article.cancelReply') : t('article.reply')}
          </button>
        </div>
      </div>

      {/* 回复表单 */}
      {showReplyForm && (
        <div className={`mt-4 ${isReply ? 'ml-4' : 'ml-8'}`}>
          <CommentForm
            articleId={articleId}
            parentId={comment.parent_id || comment.id} // 如果是回复，使用相同的 parent_id；如果是直评，使用当前 ID 作为 parent
            replyToUserId={comment.user_id} // 被回复人的 ID
            replyToUsername={commentUser.username} // 被回复人的用户名
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
