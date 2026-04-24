import { useRef, useState, type ChangeEvent } from 'react';
import { Layout, Loading } from '@/components/common';
import { uploadApi, authApi } from '@/api';
import { useAuth } from '@/hooks';
import { useUIStore } from '@/store';
import { useTranslation } from 'react-i18next';

export const Profile = () => {
  const { t } = useTranslation();
  const { user, setUser } = useAuth();
  const { showToast } = useUIStore();
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [isEditingUsername, setIsEditingUsername] = useState(false);
  const [newUsername, setNewUsername] = useState(user?.username || '');
  const [isSaving, setIsSaving] = useState(false);

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

  return (
    <Layout>
      <div className="max-w-3xl mx-auto px-6 sm:px-10 py-16">
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
          </div>
        </div>
      </div>
    </Layout>
  );
};
