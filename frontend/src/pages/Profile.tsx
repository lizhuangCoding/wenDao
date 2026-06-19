import { useRef, useState, type ChangeEvent } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Button,
  Layout,
  Loading,
  PageHeader,
  PageShell,
  Panel,
  SegmentedControl,
  TextInput,
  ToggleSwitch,
} from '@/components/common';
import { ArticleCard } from '@/components/article';
import { uploadApi, authApi, articleApi } from '@/api';
import { useAuth } from '@/hooks';
import { useUIStore } from '@/store';
import { useTranslation } from 'react-i18next';

type ArticleActivityTab = 'liked' | 'favorite';

export const Profile = () => {
  const { t } = useTranslation();
  const { user, setUser } = useAuth();
  const { showToast } = useUIStore();
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [isEditingUsername, setIsEditingUsername] = useState(false);
  const [newUsername, setNewUsername] = useState(user?.username || '');
  const [isSaving, setIsSaving] = useState(false);
  const [isSavingPreferences, setIsSavingPreferences] = useState(false);
  const [articleActivityTab, setArticleActivityTab] = useState<ArticleActivityTab>('liked');
  const [articleActivityPage, setArticleActivityPage] = useState(1);

  const articleActivityQuery = useQuery({
    queryKey: [
      'profile',
      articleActivityTab === 'liked' ? 'liked-articles' : 'favorite-articles',
      articleActivityPage,
    ],
    queryFn: () =>
      articleActivityTab === 'liked'
        ? articleApi.getLikedArticles({ page: articleActivityPage, pageSize: 4 })
        : articleApi.getFavoriteArticles({ page: articleActivityPage, pageSize: 4 }),
    enabled: !!user,
    staleTime: 30_000,
  });

  if (!user) {
    return (
      <Layout>
        <Loading />
      </Layout>
    );
  }

  const avatarUrl =
    user.avatar_url ||
    `https://api.dicebear.com/7.x/initials/svg?seed=${encodeURIComponent(user.username)}`;

  const handleUploadClick = () => {
    if (isUploading) {
      return;
    }

    fileInputRef.current?.click();
  };

  const handleFileChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const input = event.target;
    const file = input.files?.[0];

    if (!file) {
      input.value = '';
      return;
    }

    setIsUploading(true);

    try {
      const updatedUser = await uploadApi.uploadAvatar(file);
      setUser(updatedUser);
      showToast(t('profile.avatarSuccess'), 'success');
    } catch (error: any) {
      showToast(error.message || t('profile.avatarError'), 'error');
    } finally {
      input.value = '';
      setIsUploading(false);
    }
  };

  const handleUsernameUpdate = async () => {
    if (!newUsername.trim() || newUsername === user.username) {
      setIsEditingUsername(false);
      return;
    }

    if (newUsername.length < 2) {
      showToast(t('profile.usernameTooShort'), 'error');
      return;
    }

    setIsSaving(true);
    try {
      await authApi.updateUsername(newUsername);
      setUser({ ...user, username: newUsername });
      showToast(t('profile.usernameSuccess'), 'success');
      setIsEditingUsername(false);
    } catch (error: any) {
      showToast(error.message || t('profile.usernameError'), 'error');
    } finally {
      setIsSaving(false);
    }
  };

  const handleCommentReplyEmailChange = async (enabled: boolean) => {
    setIsSavingPreferences(true);
    try {
      const updatedUser = await authApi.updatePreferences({
        comment_reply_email_enabled: enabled,
      });
      setUser(updatedUser);
      showToast(t('profile.preferencesSuccess'), 'success');
    } catch (error: any) {
      showToast(error.message || t('profile.preferencesError'), 'error');
    } finally {
      setIsSavingPreferences(false);
    }
  };

  const handleArticleActivityTabChange = (tab: ArticleActivityTab) => {
    setArticleActivityTab(tab);
    setArticleActivityPage(1);
  };

  const activityArticles = articleActivityQuery.data?.data || [];
  const activityTotalPages = articleActivityQuery.data?.totalPages || 1;
  const activityTitle = articleActivityTab === 'liked' ? t('profile.likedArticles') : t('profile.favoriteArticles');
  const activityEmptyText =
    articleActivityTab === 'liked' ? t('profile.noLikedArticles') : t('profile.noFavoriteArticles');

  return (
    <Layout>
      <PageShell width="default" padding="lg">
        <Panel padding="lg">
          <div className="flex flex-col sm:flex-row sm:items-center gap-6 mb-10">
            <img
              src={avatarUrl}
              alt={`${user.username} avatar`}
              className="w-24 h-24 rounded-full object-cover border border-neutral-200 dark:border-neutral-700"
            />

            <div className="flex-1">
              <PageHeader title={t('profile.title')} description={t('profile.subtitle')} />
            </div>

            <div>
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                onChange={handleFileChange}
                className="hidden"
                disabled={isUploading}
              />
              <Button
                type="button"
                onClick={handleUploadClick}
                disabled={isUploading}
              >
                {t('profile.changeAvatar')}
              </Button>
            </div>
          </div>

          <div className="space-y-6">
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300">
                  {t('profile.username')}
                </label>
                {!isEditingUsername ? (
                  <button
                    type="button"
                    onClick={() => setIsEditingUsername(true)}
                    className="text-xs text-primary-600 hover:text-primary-700 font-bold"
                  >
                    {t('profile.edit')}
                  </button>
                ) : (
                  <div className="flex gap-2">
                    <button
                      type="button"
                      onClick={() => {
                        setIsEditingUsername(false);
                        setNewUsername(user.username);
                      }}
                      className="text-xs text-neutral-500 hover:text-neutral-600 font-bold"
                      disabled={isSaving}
                    >
                      {t('profile.cancel')}
                    </button>
                    <button
                      type="button"
                      onClick={handleUsernameUpdate}
                      className="text-xs text-primary-600 hover:text-primary-700 font-bold"
                      disabled={isSaving}
                    >
                      {isSaving ? t('profile.saving') : t('profile.save')}
                    </button>
                  </div>
                )}
              </div>
              <TextInput
                type="text"
                value={isEditingUsername ? newUsername : user.username}
                onChange={(e) => setNewUsername(e.target.value)}
                readOnly={!isEditingUsername}
                className={
                  !isEditingUsername
                    ? '[&_input]:cursor-not-allowed [&_input]:bg-neutral-50 dark:[&_input]:bg-neutral-800/80'
                    : '[&_input]:border-primary-400'
                }
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    handleUsernameUpdate();
                  } else if (e.key === 'Escape') {
                    setIsEditingUsername(false);
                    setNewUsername(user.username);
                  }
                }}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
                {t('profile.email')}
              </label>
              <TextInput
                type="email"
                value={user.email}
                readOnly
                className="[&_input]:cursor-not-allowed [&_input]:bg-neutral-50 dark:[&_input]:bg-neutral-800/80"
              />
            </div>

            <div className="border-t border-neutral-200 pt-6 dark:border-neutral-700">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <label
                    htmlFor="comment-reply-email"
                    className="block text-sm font-bold text-neutral-800 dark:text-neutral-100"
                  >
                    {t('profile.commentReplyEmail')}
                  </label>
                  <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
                    {t('profile.commentReplyEmailHint')}
                  </p>
                </div>
                <ToggleSwitch
                  id="comment-reply-email"
                  checked={user.comment_reply_email_enabled ?? true}
                  disabled={isSavingPreferences}
                  onClick={() =>
                    handleCommentReplyEmailChange(!(user.comment_reply_email_enabled ?? true))
                  }
                />
              </div>
            </div>

            <section className="border-t border-neutral-200 pt-8 dark:border-neutral-700">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h2 className="text-xl font-serif font-black text-neutral-900 dark:text-neutral-100">
                    {t('profile.articleActivity')}
                  </h2>
                  <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
                    {t('profile.articleActivityHint')}
                  </p>
                </div>
                <SegmentedControl<ArticleActivityTab>
                  value={articleActivityTab}
                  onChange={handleArticleActivityTabChange}
                  className="w-full sm:w-auto"
                  items={[
                    { value: 'liked', label: t('profile.likedArticles') },
                    { value: 'favorite', label: t('profile.favoriteArticles') },
                  ]}
                />
              </div>

              <div className="mt-8">
                {articleActivityQuery.isLoading ? (
                  <div className="rounded-2xl border border-dashed border-neutral-200 py-12 text-center text-sm font-medium text-neutral-400 dark:border-neutral-700 dark:text-neutral-500">
                    {t('common.loading')} {activityTitle}
                  </div>
                ) : articleActivityQuery.isError ? (
                  <div className="rounded-2xl border border-dashed border-red-200 bg-red-50/60 py-12 text-center dark:border-red-500/30 dark:bg-red-500/10">
                    <p className="text-sm font-bold text-red-600 dark:text-red-300">
                      {t('profile.activityLoadFailed', { title: activityTitle })}
                    </p>
                    <button
                      type="button"
                      onClick={() => articleActivityQuery.refetch()}
                      className="mt-3 text-xs font-black uppercase tracking-widest text-red-500 hover:text-red-600 dark:text-red-300"
                    >
                      {t('common.retry')}
                    </button>
                  </div>
                ) : activityArticles.length === 0 ? (
                  <div className="rounded-2xl border border-dashed border-neutral-200 py-12 text-center text-sm font-medium text-neutral-400 dark:border-neutral-700 dark:text-neutral-500">
                    {activityEmptyText}
                  </div>
                ) : (
                  <>
                    <div className="grid gap-8 sm:grid-cols-2">
                      {activityArticles.map((article) => (
                        <ArticleCard key={article.id} article={article} />
                      ))}
                    </div>
                    {activityTotalPages > 1 && (
                      <div className="mt-8 flex items-center justify-between border-t border-neutral-200 pt-6 dark:border-neutral-700">
                        <Button
                          type="button"
                          variant="secondary"
                          onClick={() => setArticleActivityPage((page) => Math.max(1, page - 1))}
                          disabled={articleActivityPage <= 1}
                        >
                          {t('admin.previous')}
                        </Button>
                        <span className="text-xs font-bold tracking-widest text-neutral-400 dark:text-neutral-500">
                          {articleActivityPage} / {activityTotalPages}
                        </span>
                        <Button
                          type="button"
                          variant="secondary"
                          onClick={() =>
                            setArticleActivityPage((page) => Math.min(activityTotalPages, page + 1))
                          }
                          disabled={articleActivityPage >= activityTotalPages}
                        >
                          {t('admin.next')}
                        </Button>
                      </div>
                    )}
                  </>
                )}
              </div>
            </section>
          </div>
        </Panel>
      </PageShell>
    </Layout>
  );
};
