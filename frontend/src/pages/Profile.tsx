import { useRef, useState, type ChangeEvent } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Layout, Loading } from '@/components/common';
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
  const activityTitle = articleActivityTab === 'liked' ? '我的点赞' : '我的收藏';
  const activityEmptyText =
    articleActivityTab === 'liked' ? '还没有点赞过文章' : '还没有收藏过文章';

  return (
    <Layout>
      <div className="max-w-5xl mx-auto px-6 sm:px-10 py-16">
        <div className="bg-white dark:bg-neutral-900 border border-neutral-100 dark:border-neutral-800 rounded-3xl shadow-sm p-8 sm:p-10">
          <div className="flex flex-col sm:flex-row sm:items-center gap-6 mb-10">
            <img
              src={avatarUrl}
              alt={`${user.username} avatar`}
              className="w-24 h-24 rounded-full object-cover border border-neutral-200 dark:border-neutral-700"
            />

            <div className="flex-1">
              <h1 className="text-3xl font-serif font-black text-neutral-900 dark:text-neutral-100 mb-2">
                {t('profile.title')}
              </h1>
              <p className="text-sm text-neutral-500 dark:text-neutral-400">
                {t('profile.subtitle')}
              </p>
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
              <button
                type="button"
                onClick={handleUploadClick}
                disabled={isUploading}
                className="btn btn-primary disabled:opacity-60 disabled:cursor-not-allowed"
              >
                {t('profile.changeAvatar')}
              </button>
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
              <input
                type="text"
                value={isEditingUsername ? newUsername : user.username}
                onChange={(e) => setNewUsername(e.target.value)}
                readOnly={!isEditingUsername}
                className={`input w-full ${
                  !isEditingUsername
                    ? 'bg-neutral-50 dark:bg-neutral-800/80 dark:border-neutral-700 dark:text-neutral-100 cursor-not-allowed'
                    : 'bg-white dark:bg-neutral-800 dark:border-primary-500 dark:text-neutral-100'
                }`}
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
              <input
                type="email"
                value={user.email}
                readOnly
                className="input w-full bg-neutral-50 dark:bg-neutral-800/80 dark:border-neutral-700 dark:text-neutral-100 cursor-not-allowed"
              />
            </div>

            <div className="border-t border-neutral-100 pt-6 dark:border-neutral-800">
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
                <button
                  id="comment-reply-email"
                  type="button"
                  role="switch"
                  aria-checked={user.comment_reply_email_enabled ?? true}
                  disabled={isSavingPreferences}
                  onClick={() =>
                    handleCommentReplyEmailChange(!(user.comment_reply_email_enabled ?? true))
                  }
                  className={`relative inline-flex h-7 w-12 flex-shrink-0 rounded-full border-2 border-transparent transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${
                    user.comment_reply_email_enabled ?? true
                      ? 'bg-primary-500'
                      : 'bg-neutral-300 dark:bg-neutral-700'
                  }`}
                >
                  <span
                    className={`inline-block h-6 w-6 transform rounded-full bg-white shadow transition-transform ${
                      user.comment_reply_email_enabled ?? true ? 'translate-x-5' : 'translate-x-0'
                    }`}
                  />
                </button>
              </div>
            </div>

            <section className="border-t border-neutral-100 pt-8 dark:border-neutral-800">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h2 className="text-xl font-serif font-black text-neutral-900 dark:text-neutral-100">
                    文章互动
                  </h2>
                  <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
                    查看你喜欢和收藏过的文章
                  </p>
                </div>
                <div className="inline-flex rounded-full border border-neutral-200 bg-neutral-50 p-1 dark:border-neutral-700 dark:bg-neutral-800">
                  <button
                    type="button"
                    onClick={() => handleArticleActivityTabChange('liked')}
                    className={`rounded-full px-4 py-2 text-sm font-bold transition-colors ${
                      articleActivityTab === 'liked'
                        ? 'bg-white text-neutral-900 shadow-sm dark:bg-neutral-950 dark:text-neutral-100'
                        : 'text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-neutral-100'
                    }`}
                  >
                    我的点赞
                  </button>
                  <button
                    type="button"
                    onClick={() => handleArticleActivityTabChange('favorite')}
                    className={`rounded-full px-4 py-2 text-sm font-bold transition-colors ${
                      articleActivityTab === 'favorite'
                        ? 'bg-white text-neutral-900 shadow-sm dark:bg-neutral-950 dark:text-neutral-100'
                        : 'text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-neutral-100'
                    }`}
                  >
                    我的收藏
                  </button>
                </div>
              </div>

              <div className="mt-8">
                {articleActivityQuery.isLoading ? (
                  <div className="rounded-2xl border border-dashed border-neutral-200 py-12 text-center text-sm font-medium text-neutral-400 dark:border-neutral-700 dark:text-neutral-500">
                    正在加载{activityTitle}
                  </div>
                ) : articleActivityQuery.isError ? (
                  <div className="rounded-2xl border border-dashed border-red-200 bg-red-50/60 py-12 text-center dark:border-red-500/30 dark:bg-red-500/10">
                    <p className="text-sm font-bold text-red-600 dark:text-red-300">
                      {activityTitle}加载失败
                    </p>
                    <button
                      type="button"
                      onClick={() => articleActivityQuery.refetch()}
                      className="mt-3 text-xs font-black uppercase tracking-widest text-red-500 hover:text-red-600 dark:text-red-300"
                    >
                      重试
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
                      <div className="mt-8 flex items-center justify-between border-t border-neutral-100 pt-6 dark:border-neutral-800">
                        <button
                          type="button"
                          onClick={() => setArticleActivityPage((page) => Math.max(1, page - 1))}
                          disabled={articleActivityPage <= 1}
                          className="btn btn-secondary disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          上一页
                        </button>
                        <span className="text-xs font-bold tracking-widest text-neutral-400 dark:text-neutral-500">
                          {articleActivityPage} / {activityTotalPages}
                        </span>
                        <button
                          type="button"
                          onClick={() =>
                            setArticleActivityPage((page) => Math.min(activityTotalPages, page + 1))
                          }
                          disabled={articleActivityPage >= activityTotalPages}
                          className="btn btn-secondary disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          下一页
                        </button>
                      </div>
                    )}
                  </>
                )}
              </div>
            </section>
          </div>
        </div>
      </div>
    </Layout>
  );
};
