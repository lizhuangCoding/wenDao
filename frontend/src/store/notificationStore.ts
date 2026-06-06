import { create } from 'zustand';
import { notificationApi } from '@/api';

interface NotificationState {
  unreadCount: number;
  isPolling: boolean;

  setUnreadCount: (count: number) => void;
  decrementUnread: (by?: number) => void;
  fetchUnreadCount: () => Promise<void>;
  startPolling: () => void;
  stopPolling: () => void;
}

let pollingTimer: ReturnType<typeof setInterval> | null = null;

export const useNotificationStore = create<NotificationState>((set, get) => ({
  unreadCount: 0,
  isPolling: false,

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

  startPolling: () => {
    if (get().isPolling) return;
    set({ isPolling: true });
    get().fetchUnreadCount();
    pollingTimer = setInterval(() => {
      get().fetchUnreadCount();
    }, 30000);
  },

  stopPolling: () => {
    set({ isPolling: false });
    if (pollingTimer) {
      clearInterval(pollingTimer);
      pollingTimer = null;
    }
  },
}));
