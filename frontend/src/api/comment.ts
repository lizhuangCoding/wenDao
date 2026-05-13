import { request } from './client';
import type { Comment, CreateCommentRequest, PaginatedResponse, PaginationParams } from '@/types';
import { toPaginationQuery } from './pagination';

// 评论 API
export const commentApi = {
  // 获取文章评论
  getComments: (articleId: number) => {
    return request.get<Comment[]>(`/comments/article/${articleId}`);
  },

  // 获取所有评论（管理员）
  getAdminComments: (params: PaginationParams & { status?: string; keyword?: string }) => {
    return request.get<PaginatedResponse<Comment>>('/admin/comments', { params: toPaginationQuery(params) });
  },

  // 创建评论
  createComment: (data: CreateCommentRequest) => {
    return request.post<Comment>('/comments', {
      article_id: data.articleId,
      content: data.content,
      parent_id: data.parentId,
      reply_to_user_id: data.replyToUserId,
    });
  },

  // 删除评论
  deleteComment: (id: number) => {
    return request.delete(`/comments/${id}`);
  },

  // 管理员删除评论
  adminDeleteComment: (id: number) => {
    return request.delete(`/admin/comments/${id}`);
  },

  // 管理员批量删除评论
  batchDeleteComments: (ids: number[]) => {
    return request.post<{ message: string; deleted_count: number }>('/admin/comments/batch-delete', { ids });
  },

  // 管理员恢复评论
  adminRestoreComment: (id: number) => {
    return request.post(`/admin/comments/${id}/restore`);
  },
};
