import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { notificationApi } from '@/api';
import { Loading, Pagination, EmptyState } from '@/components/common';
import { useNotificationStore } from '@/store';
import { formatDate } from '@/utils';
import type { Notification } from '@/types';

export const NotificationList = () => {
  const queryClient = useQueryClient();
  const { setUnreadCount } = useNotificationStore();
  const [page, setPage] = useState(1);
  const pageSize = 20;

  const { data, isLoading, isError } = useQuery({
    queryKey: ['notifications', page],
    queryFn: () => notificationApi.list(page, pageSize),
  });

  const handleMarkRead = async (id: number) => {
    try {
      await notificationApi.markRead(id);
      setUnreadCount(Math.max(0, useNotificationStore.getState().unreadCount - 1));
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    } catch {
      // 静默失败
    }
  };

  const handleMarkAllRead = async () => {
    try {
      await notificationApi.markAllRead();
      setUnreadCount(0);
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    } catch {
      // 静默失败
    }
  };

  if (isLoading) return <Loading />;

  return (
    <div className="max-w-3xl mx-auto px-4 py-8">
      <div className="flex items-center justify-between mb-8">
        <h1 className="text-2xl font-bold text-neutral-700 dark:text-neutral-100">通知</h1>
        {data?.data && data.data.length > 0 && (
          <button
            type="button"
            onClick={handleMarkAllRead}
            className="text-sm text-primary-600 dark:text-primary-400 hover:underline"
          >
            全部标为已读
          </button>
        )}
      </div>

      {isError ? (
        <EmptyState title="加载失败" description="无法加载通知，请稍后重试" />
      ) : data?.data && data.data.length > 0 ? (
        <>
          <div className="space-y-2">
            {data.data.map((notif: Notification) => (
              <Link
                key={notif.id}
                to={notif.link_url || '/notifications'}
                onClick={() => {
                  if (!notif.is_read) handleMarkRead(notif.id);
                }}
                className={`flex items-start gap-4 p-4 rounded-xl transition-colors hover:bg-neutral-50 dark:hover:bg-neutral-800/50 ${
                  !notif.is_read
                    ? 'bg-primary-50/50 dark:bg-primary-900/10 border-l-2 border-primary-500'
                    : 'bg-white dark:bg-neutral-900 border border-neutral-100 dark:border-neutral-800'
                }`}
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <h3 className="text-sm font-semibold text-neutral-700 dark:text-neutral-200">
                      {notif.title}
                    </h3>
                    {!notif.is_read && (
                      <span className="px-2 py-0.5 text-[10px] font-bold text-primary-600 dark:text-primary-400 bg-primary-100 dark:bg-primary-900/30 rounded-full">
                        新
                      </span>
                    )}
                  </div>
                  <p className="text-sm text-neutral-500 dark:text-neutral-400 line-clamp-2">
                    {notif.content}
                  </p>
                  <span className="text-xs text-neutral-400 dark:text-neutral-500 mt-2 inline-block">
                    {formatDate(notif.created_at)}
                  </span>
                </div>
              </Link>
            ))}
          </div>

          {data.totalPages > 1 && (
            <div className="mt-8">
              <Pagination
                page={page}
                totalPages={data.totalPages}
                onChange={setPage}
              />
            </div>
          )}
        </>
      ) : (
        <EmptyState title="暂无通知" description="你还没有收到任何通知" />
      )}
    </div>
  );
};
