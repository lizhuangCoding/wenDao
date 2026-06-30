import { create } from 'zustand';
import { notificationApi } from '@/api';

interface NotificationState {
  unreadCount: number;

  setUnreadCount: (count: number) => void;
  decrementUnread: (by?: number) => void;
  fetchUnreadCount: () => Promise<void>;
}

export const useNotificationStore = create<NotificationState>((set) => ({
  unreadCount: 0,

  setUnreadCount: (count) => set({ unreadCount: count }),

  decrementUnread: (by = 1) =>
    set((state) => ({ unreadCount: Math.max(0, state.unreadCount - by) })),

  fetchUnreadCount: async () => {
    try {
      const res = await notificationApi.getUnreadCount();
      set({ unreadCount: res.unread_count });
    } catch {
      // 静默失败
    }
  },
}));
