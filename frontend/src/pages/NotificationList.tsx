import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { ArrowLeft, CheckCheck, ExternalLink, Mail, MailOpen } from 'lucide-react';
import { notificationApi } from '@/api';
import {
  EmptyState,
  Layout,
  Loading,
  PageHeader,
  Pagination,
  SegmentedControl,
} from '@/components/common';
import { ArticleContent } from '@/components/article';
import { useNotificationStore } from '@/store';
import { formatDate } from '@/utils';
import type { Notification, NotificationType } from '@/types';

type NotificationFilter = 'all' | NotificationType;

export const NotificationList = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { decrementUnread, fetchUnreadCount, setUnreadCount } = useNotificationStore();
  const [page, setPage] = useState(1);
  const [filterType, setFilterType] = useState<NotificationFilter>('all');
  const pageSize = 20;

  const { data, isLoading, isError } = useQuery({
    queryKey: ['notifications', page, filterType],
    queryFn: () => notificationApi.list(page, pageSize, filterType === 'all' ? undefined : filterType),
  });

  const handleFilterChange = (nextType: NotificationFilter) => {
    setFilterType(nextType);
    setPage(1);
  };

  const handleMarkRead = async (id: number) => {
    try {
      await notificationApi.markRead(id);
      decrementUnread();
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
      void fetchUnreadCount();
    } catch {
      // 静默失败
    }
  };

  if (isLoading) {
    return (
      <Layout>
        <Loading />
      </Layout>
    );
  }

  const notifications = data?.data ?? [];
  const hasNotifications = notifications.length > 0;
  return (
    <Layout>
      <div className="mx-auto max-w-5xl px-5 py-10 sm:px-8 lg:px-10">
        <PageHeader
          title={t('notification.title')}
          description={t('notification.description')}
          className="mb-6"
          actions={
            <>
              <Link to="/" className="btn btn-secondary">
                <ArrowLeft className="h-4 w-4" />
                {t('notification.returnHome')}
              </Link>
              {hasNotifications && (
                <button type="button" onClick={handleMarkAllRead} className="btn btn-primary">
                  <CheckCheck className="h-4 w-4" />
                  {t('notification.markAllRead')}
                </button>
              )}
            </>
          }
        />

        <div className="mb-6 rounded-2xl border border-neutral-200 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-900/60">
          <SegmentedControl
            value={filterType}
            onChange={handleFilterChange}
            className="w-full"
            items={[
              { value: 'all', label: t('notification.all') },
              { value: 'comment_reply', label: t('notification.commentReply') },
              { value: 'comment_like', label: t('notification.commentLike') },
              { value: 'admin_broadcast', label: t('notification.adminBroadcast') },
            ]}
          />
        </div>

        {isError ? (
          <EmptyState title={t('common.failed')} description={t('notification.loadFailedDescription')} />
        ) : hasNotifications ? (
          <>
            <div className="space-y-4">
              {notifications.map((notif: Notification) => (
                <article
                  key={notif.id}
                  className={`group overflow-hidden rounded-2xl border bg-white shadow-sm transition-colors dark:bg-neutral-900 ${
                    !notif.is_read
                      ? 'border-primary-200 ring-1 ring-primary-100 dark:border-primary-500/30 dark:ring-primary-500/10'
                      : 'border-neutral-100 dark:border-neutral-800'
                  }`}
                >
                  <div className="flex flex-col gap-4 p-5 sm:flex-row sm:items-start sm:justify-between">
                    <div className="flex min-w-0 gap-4">
                      <div
                        className={`mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl ${
                          notif.is_read
                            ? 'bg-neutral-100 text-neutral-500 dark:bg-neutral-800 dark:text-neutral-400'
                            : 'bg-primary-50 text-primary-600 dark:bg-primary-500/10 dark:text-primary-300'
                        }`}
                      >
                        {notif.is_read ? (
                          <MailOpen className="h-5 w-5" />
                        ) : (
                          <Mail className="h-5 w-5" />
                        )}
                      </div>
                      <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-2">
                          <h2 className="text-lg font-bold text-neutral-800 dark:text-neutral-100">
                            {notif.title}
                          </h2>
                          {!notif.is_read && (
                            <span className="rounded-full bg-primary-50 px-2.5 py-1 text-[10px] font-bold uppercase tracking-wider text-primary-600 dark:bg-primary-500/10 dark:text-primary-300">
                              {t('notification.newMessage')}
                            </span>
                          )}
                        </div>
                        <p className="mt-1 text-xs font-semibold uppercase tracking-wider text-neutral-400 dark:text-neutral-500">
                          {formatDate(notif.created_at)}
                        </p>
                      </div>
                    </div>

                    <div className="flex shrink-0 flex-wrap items-center gap-2">
                      {notif.link_url && (
                        <Link
                          to={notif.link_url}
                          onClick={() => {
                            if (!notif.is_read) void handleMarkRead(notif.id);
                          }}
                          className="inline-flex h-9 items-center gap-2 rounded-full border border-neutral-200 px-3 text-xs font-bold text-neutral-600 transition-colors hover:border-primary-200 hover:text-primary-600 dark:border-neutral-700 dark:text-neutral-300 dark:hover:border-primary-500/40 dark:hover:text-primary-300"
                        >
                          {t('notification.viewRelatedContent')}
                          <ExternalLink className="h-3.5 w-3.5" />
                        </Link>
                      )}
                      {!notif.is_read && (
                        <button
                          type="button"
                          onClick={() => handleMarkRead(notif.id)}
                          className="inline-flex h-9 items-center gap-2 rounded-full bg-neutral-900 px-3 text-xs font-bold text-white transition-colors hover:bg-primary-600 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-primary-500 dark:hover:text-white"
                        >
                          <CheckCheck className="h-3.5 w-3.5" />
                          {t('notification.markRead')}
                        </button>
                      )}
                    </div>
                  </div>

                  <div className="border-t border-neutral-100 bg-neutral-50/70 px-5 py-5 dark:border-neutral-800 dark:bg-neutral-950/30">
                    <div className="notification-message-body article-reading-body max-w-none text-sm text-neutral-700 dark:text-neutral-200">
                      <ArticleContent content={notif.content} />
                    </div>
                  </div>
                </article>
              ))}
            </div>

            {data!.totalPages > 1 && (
              <div className="mt-8">
                <Pagination page={page} totalPages={data!.totalPages} onChange={setPage} />
              </div>
            )}
          </>
        ) : (
          <EmptyState title={t('notification.noNotifications')} description={t('notification.noNotificationsDescription')} />
        )}
      </div>
    </Layout>
  );
};
