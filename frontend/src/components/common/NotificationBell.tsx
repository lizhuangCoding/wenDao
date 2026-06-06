import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { notificationApi } from '@/api';
import { useAuthStore, useNotificationStore } from '@/store';
import type { Notification } from '@/types';
import { formatDate } from '@/utils';
import { markdownToPlainText } from '@/utils/markdown';

const getNotificationPreview = (content: string): string => {
  return markdownToPlainText(content) || '暂无内容';
};

export const NotificationBell = () => {
  const queryClient = useQueryClient();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const { unreadCount, decrementUnread, fetchUnreadCount, startPolling, stopPolling } =
    useNotificationStore();
  const [isOpen, setIsOpen] = useState(false);

  const { data: recentNotifs, isError } = useQuery({
    queryKey: ['notifications', 'recent'],
    queryFn: () => notificationApi.list(1, 5),
    enabled: isOpen,
    staleTime: 30_000,
  });

  useEffect(() => {
    if (isAuthenticated) {
      fetchUnreadCount();
      startPolling();
      return () => stopPolling();
    }
  }, [isAuthenticated, fetchUnreadCount, startPolling, stopPolling]);

  const handleMarkRead = async (id: number) => {
    try {
      await notificationApi.markRead(id);
      decrementUnread();
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    } catch {
      // 静默失败
    }
  };

  if (!isAuthenticated) return null;

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="relative p-2 rounded-lg text-neutral-500 dark:text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors"
        aria-label="通知"
      >
        <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
        </svg>
        {unreadCount > 0 && (
          <span className="absolute -top-0.5 -right-0.5 flex items-center justify-center min-w-[18px] h-[18px] px-1 text-[10px] font-bold text-white bg-red-500 rounded-full">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </button>

      {isOpen && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setIsOpen(false)} />
          <div className="absolute right-0 top-full mt-2 w-80 bg-white dark:bg-neutral-800 rounded-xl shadow-lg border border-neutral-100 dark:border-neutral-700 z-50 overflow-hidden">
            <div className="flex items-center justify-between px-4 py-3 border-b border-neutral-100 dark:border-neutral-700">
              <h3 className="font-semibold text-neutral-700 dark:text-neutral-200 text-sm">通知</h3>
              <Link
                to="/notifications"
                onClick={() => setIsOpen(false)}
                className="text-xs text-primary-600 dark:text-primary-400 hover:underline"
              >
                查看全部
              </Link>
            </div>

            <div className="max-h-80 overflow-y-auto">
              {!isError && recentNotifs?.data && recentNotifs.data.length > 0 ? (
                recentNotifs.data.map((notif: Notification) => (
                  <Link
                    key={notif.id}
                    to={notif.link_url || '/notifications'}
                    onClick={() => {
                      if (!notif.is_read) handleMarkRead(notif.id);
                      setIsOpen(false);
                    }}
                    className={`flex items-start gap-3 px-4 py-3 hover:bg-neutral-50 dark:hover:bg-neutral-700/50 transition-colors ${
                      !notif.is_read ? 'bg-primary-50/50 dark:bg-primary-900/10' : ''
                    }`}
                  >
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-neutral-700 dark:text-neutral-200 truncate">
                        {notif.title}
                      </p>
                      <p className="text-xs text-neutral-500 dark:text-neutral-400 mt-0.5 line-clamp-2">
                        {getNotificationPreview(notif.content)}
                      </p>
                      <p className="text-[11px] text-neutral-400 dark:text-neutral-500 mt-1">
                        {formatDate(notif.created_at)}
                      </p>
                    </div>
                    {!notif.is_read && (
                      <span className="flex-shrink-0 w-2 h-2 mt-2 rounded-full bg-primary-500" />
                    )}
                  </Link>
                ))
              ) : (
                <div className="py-8 text-center text-sm text-neutral-400 dark:text-neutral-500">
                  暂无新通知
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
};
