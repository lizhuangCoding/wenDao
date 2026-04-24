import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Eye, EyeOff } from 'lucide-react';
import { authApi } from '@/api';
import { GitHubAuthButton } from '@/components/common';
import { useAuth } from '@/hooks';
import { useUIStore } from '@/store';

export const Register = () => {
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const { t } = useTranslation();
  const { register } = useAuth();
  const { showToast } = useUIStore();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!username || !email || !password || !confirmPassword) {
      showToast(t('admin.pleaseFillComplete'), 'error');
      return;
    }

    if (password !== confirmPassword) {
      showToast(t('auth.passwordMismatch'), 'error');
      return;
    }

    if (password.length < 6) {
      showToast(t('auth.passwordTooShort'), 'error');
      return;
    }

    setIsLoading(true);
    try {
      await register(username, email, password);
      showToast(t('auth.registerSuccess'), 'success');
      navigate('/');
    } catch (error: any) {
      showToast(error.message || t('auth.register'), 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const handleGitHubLogin = () => {
    authApi.startGitHubLogin();
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-neutral-50 dark:bg-neutral-950 px-4 transition-colors">
      <div className="max-w-md w-full bg-white/95 dark:bg-neutral-900/95 backdrop-blur rounded-[28px] shadow-elevated border border-neutral-100 dark:border-neutral-800 p-8 md:p-10">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-serif font-black text-neutral-700 dark:text-neutral-100 text-center mb-2">
            {t('auth.register')}
          </h1>
          <p className="text-sm text-neutral-500 dark:text-neutral-400">
            {t('chat.askAbout')}
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
              {t('auth.username')}
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder={t('auth.username')}
              disabled={isLoading}
              className="input w-full dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-100 dark:placeholder-neutral-500 dark:focus:bg-neutral-800"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
              {t('auth.email')}
            </label>
            <input
              type="text"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="your@email.com"
              disabled={isLoading}
              className="input w-full dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-100 dark:placeholder-neutral-500 dark:focus:bg-neutral-800"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
              {t('auth.password')}
            </label>
            <div className="relative">
              <input
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={t('auth.password')}
                disabled={isLoading}
                className="input w-full pr-14 dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-100 dark:placeholder-neutral-500 dark:focus:bg-neutral-800"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-4 top-1/2 -translate-y-1/2 p-2 text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200 transition-colors focus:outline-none"
                tabIndex={-1}
              >
                {showPassword ? <EyeOff size={20} /> : <Eye size={20} />}
              </button>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
              {t('auth.confirmPassword')}
            </label>
            <div className="relative">
              <input
                type={showConfirmPassword ? 'text' : 'password'}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder={t('auth.confirmPassword')}
                disabled={isLoading}
                className="input w-full pr-14 dark:bg-neutral-800 dark:border-neutral-700 dark:text-neutral-100 dark:placeholder-neutral-500 dark:focus:bg-neutral-800"
              />
              <button
                type="button"
                onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                className="absolute right-4 top-1/2 -translate-y-1/2 p-2 text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200 transition-colors focus:outline-none"
                tabIndex={-1}
              >
                {showConfirmPassword ? <EyeOff size={20} /> : <Eye size={20} />}
              </button>
            </div>
          </div>

          <button
            type="submit"
            disabled={isLoading}
            className="btn btn-primary w-full justify-center disabled:opacity-60 disabled:cursor-not-allowed"
          >
            {isLoading ? `${t('auth.register')}...` : t('auth.register')}
          </button>
        </form>

        <div className="mt-7 space-y-4">
          <div className="flex items-center gap-4 text-[11px] font-medium tracking-[0.08em] text-neutral-400 dark:text-neutral-500">
            <div className="h-px flex-1 bg-neutral-200 dark:bg-neutral-800" />
            <span>{t('auth.orContinueWithGithub')}</span>
            <div className="h-px flex-1 bg-neutral-200 dark:bg-neutral-800" />
          </div>

          <GitHubAuthButton
            label={t('auth.continueWithGithubRegister')}
            onClick={handleGitHubLogin}
            disabled={isLoading}
          />
        </div>

        <p className="text-center text-neutral-600 dark:text-neutral-400 mt-6">
          {t('auth.hasAccount')}{' '}
          <Link to="/login" className="text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300 transition-colors">
            {t('nav.login')}
          </Link>
        </p>
      </div>
    </div>
  );
};
