import { request } from './client';
import type { Notification, PaginatedResponse } from '@/types';

export const notificationApi = {
  // 获取通知列表
  list: (page: number = 1, pageSize: number = 20) => {
    return request.get<PaginatedResponse<Notification>>('/notifications', {
      params: { page, page_size: pageSize },
    });
  },

  // 获取未读通知数
  getUnreadCount: () => {
    return request.get<{ unread_count: number }>('/notifications/unread-count');
  },

  // 标记单条已读
  markRead: (id: number) => {
    return request.put(`/notifications/${id}/read`);
  },

  // 全部标为已读
  markAllRead: () => {
    return request.put('/notifications/read-all');
  },

  // 管理员发送广播
  broadcast: (data: { title: string; content: string; link_url?: string }) => {
    return request.post('/admin/notifications/broadcast', data);
  },
};
