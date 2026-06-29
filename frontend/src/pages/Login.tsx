import { useRef, useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Eye, EyeOff, Loader2 } from 'lucide-react';
import { authApi } from '@/api';
import { AuthFormMessage, AuthTextField, PasswordResetForm } from '@/components/auth';
import { GitHubAuthButton } from '@/components/common';
import { useAuth } from '@/hooks';
import { useUIStore } from '@/store';
import { normalizeApiError } from '@/utils/apiError';
import {
  mapAuthErrorToForm,
  validateLoginForm,
  type AuthField,
  type LoginFormValues,
} from '@/utils/authForm';

type LoginField = Extract<AuthField, 'email' | 'password'>;

const loginFieldOrder: LoginField[] = ['email', 'password'];

export const Login = () => {
  const [isResetMode, setIsResetMode] = useState(false);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [fieldErrors, setFieldErrors] = useState<Partial<Record<AuthField, string>>>({});
  const [formError, setFormError] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const emailInputRef = useRef<HTMLInputElement>(null);
  const passwordInputRef = useRef<HTMLInputElement>(null);
  const { t } = useTranslation();
  const { login } = useAuth();
  const { showToast } = useUIStore();
  const navigate = useNavigate();

  const resolveMessage = (messageKey: string, fallbackMessage?: string) => {
    const translated = t(messageKey);
    return translated === messageKey ? fallbackMessage || t('auth.loginFailed') : translated;
  };

  const getFieldError = (field: LoginField) => {
    const errorKey = fieldErrors[field];
    return errorKey ? resolveMessage(errorKey) : undefined;
  };

  const getValues = (nextValues: Partial<LoginFormValues> = {}): LoginFormValues => ({
    email,
    password,
    ...nextValues,
  });

  const focusField = (field: LoginField) => {
    const ref = field === 'email' ? emailInputRef : passwordInputRef;
    requestAnimationFrame(() => ref.current?.focus());
  };

  const setFieldError = (field: LoginField, errorKey?: string) => {
    setFieldErrors((current) => {
      const next = { ...current };
      if (errorKey) {
        next[field] = errorKey;
      } else {
        delete next[field];
      }
      return next;
    });
  };

  const validateField = (field: LoginField, values: LoginFormValues) => {
    const result = validateLoginForm(values);
    setFieldError(field, result.fieldErrors[field]);
  };

  const handleEmailChange = (nextEmail: string) => {
    setEmail(nextEmail);
    setFormError('');
    if (fieldErrors.email) {
      validateField('email', getValues({ email: nextEmail }));
    }
  };

  const handlePasswordChange = (nextPassword: string) => {
    setPassword(nextPassword);
    setFormError('');
    if (fieldErrors.password) {
      validateField('password', getValues({ password: nextPassword }));
    }
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setFormError('');

    const validation = validateLoginForm({ email, password });
    setEmail(validation.values.email);

    if (!validation.isValid) {
      setFieldErrors(validation.fieldErrors);
      const firstInvalidField = loginFieldOrder.find((field) => validation.fieldErrors[field]);
      if (firstInvalidField) {
        focusField(firstInvalidField);
      }
      return;
    }

    setFieldErrors({});
    setIsLoading(true);
    try {
      await login(validation.values.email, validation.values.password);
      showToast(t('auth.loginSuccess'), 'success');
      navigate('/');
    } catch (error) {
      const feedback = mapAuthErrorToForm(
        normalizeApiError(error, t('auth.loginFailed')).message,
        'login'
      );
      const message = resolveMessage(feedback.messageKey, feedback.fallbackMessage);

      if (feedback.target === 'email' || feedback.target === 'password') {
        setFieldError(feedback.target, feedback.messageKey);
        focusField(feedback.target);
      } else {
        setFormError(message);
        if (feedback.messageKey === 'auth.invalidCredentials') {
          focusField('password');
        }
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleResetSuccess = (resetEmail: string) => {
    setEmail(resetEmail);
    setPassword('');
    setIsResetMode(false);
    requestAnimationFrame(() => passwordInputRef.current?.focus());
  };

  const handleGitHubLogin = () => {
    authApi.startGitHubLogin();
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-neutral-50 dark:bg-neutral-950 px-4 transition-colors">
      <div className="max-w-md w-full bg-white/95 dark:bg-neutral-900/95 backdrop-blur rounded-[28px] shadow-elevated border border-neutral-200 dark:border-neutral-700 p-8 md:p-10">
        {isResetMode ? (
          <PasswordResetForm
            initialEmail={email}
            onBack={() => setIsResetMode(false)}
            onSuccess={handleResetSuccess}
          />
        ) : (
          <>
            <div className="text-center mb-8">
              <h1 className="text-3xl font-serif font-black text-neutral-700 dark:text-neutral-100 mb-2">
                {t('auth.login')}
              </h1>
              <p className="text-sm text-neutral-500 dark:text-neutral-400">
                {t('chat.askAbout')}
              </p>
            </div>

            <form onSubmit={handleSubmit} className="space-y-5" noValidate>
              <AuthFormMessage message={formError} />

              <AuthTextField
                ref={emailInputRef}
                id="login-email"
                label={t('auth.email')}
                type="email"
                value={email}
                onChange={(e) => handleEmailChange(e.target.value)}
                onBlur={() => validateField('email', getValues())}
                placeholder="your@email.com"
                autoComplete="email"
                inputMode="email"
                disabled={isLoading}
                error={getFieldError('email')}
              />

              <AuthTextField
                ref={passwordInputRef}
                id="login-password"
                label={t('auth.password')}
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => handlePasswordChange(e.target.value)}
                onBlur={() => validateField('password', getValues())}
                placeholder="••••••••"
                autoComplete="current-password"
                disabled={isLoading}
                error={getFieldError('password')}
                trailing={
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    disabled={isLoading}
                    aria-label={showPassword ? t('auth.hidePassword') : t('auth.showPassword')}
                    aria-pressed={showPassword}
                    title={showPassword ? t('auth.hidePassword') : t('auth.showPassword')}
                    className="inline-flex h-9 w-9 items-center justify-center rounded-full text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-700 focus:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/10 disabled:cursor-not-allowed dark:hover:bg-neutral-700 dark:hover:text-neutral-100"
                  >
                    {showPassword ? <EyeOff size={20} /> : <Eye size={20} />}
                  </button>
                }
              />

              <div className="-mt-2 flex justify-end">
                <button
                  type="button"
                  onClick={() => {
                    setFormError('');
                    setFieldErrors({});
                    setPassword('');
                    setIsResetMode(true);
                  }}
                  className="text-sm font-semibold text-primary-600 transition-colors hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
                >
                  {t('auth.forgotPassword')}
                </button>
              </div>

              <button
                type="submit"
                disabled={isLoading}
                className="btn btn-primary w-full justify-center disabled:opacity-60 disabled:cursor-not-allowed"
              >
                {isLoading ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                    {t('auth.loggingIn')}
                  </>
                ) : (
                  t('auth.login')
                )}
              </button>
            </form>

            <div className="mt-7 space-y-4">
              <div className="flex items-center gap-4 text-[11px] font-medium tracking-[0.08em] text-neutral-400 dark:text-neutral-500">
                <div className="h-px flex-1 bg-neutral-200 dark:bg-neutral-800" />
                <span>{t('auth.orContinueWithGithub')}</span>
                <div className="h-px flex-1 bg-neutral-200 dark:bg-neutral-800" />
              </div>

              <GitHubAuthButton
                label={t('auth.continueWithGithubLogin')}
                onClick={handleGitHubLogin}
                disabled={isLoading}
              />
            </div>

            <p className="text-center text-neutral-600 dark:text-neutral-400 mt-6">
              {t('auth.noAccount')}{' '}
              <Link
                to="/register"
                className="text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300 transition-colors"
              >
                {t('nav.signup')}
              </Link>
            </p>
          </>
        )}
      </div>
    </div>
  );
};
