import type { Article } from './article';
import type { User } from './auth';

export interface Comment {
  id: number;
  content: string;
  article_id: number;
  user_id: number;
  user?: User;
  parent_id?: number;
  reply_to_user_id?: number;
  reply_to_user?: User;
  replies?: Comment[];
  status: string;
  article?: Article;
  like_count: number;
  dislike_count: number;
  created_at: string;
  updated_at: string;
}

export interface CreateCommentRequest {
  content: string;
  articleId: number;
  parentId?: number;
  replyToUserId?: number;
}
