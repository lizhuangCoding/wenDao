import { useRef, useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Eye, EyeOff, Loader2 } from 'lucide-react';
import { authApi } from '@/api';
import { AuthFormMessage, AuthTextField, VerificationCodeField } from '@/components/auth';
import { GitHubAuthButton } from '@/components/common';
import { useAuth, useCountdown } from '@/hooks';
import { useUIStore } from '@/store';
import {
  mapAuthErrorToForm,
  validateAuthEmail,
  validateRegisterForm,
  type AuthField,
  type RegisterFormValues,
} from '@/utils/authForm';

type RegisterField = Extract<
  AuthField,
  'username' | 'email' | 'verificationCode' | 'password' | 'confirmPassword'
>;

const registerFieldOrder: RegisterField[] = [
  'username',
  'email',
  'verificationCode',
  'password',
  'confirmPassword',
];

export const Register = () => {
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [verificationCode, setVerificationCode] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [fieldErrors, setFieldErrors] = useState<Partial<Record<AuthField, string>>>({});
  const [formError, setFormError] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [isCodeSending, setIsCodeSending] = useState(false);
  const usernameInputRef = useRef<HTMLInputElement>(null);
  const emailInputRef = useRef<HTMLInputElement>(null);
  const verificationCodeInputRef = useRef<HTMLInputElement>(null);
  const passwordInputRef = useRef<HTMLInputElement>(null);
  const confirmPasswordInputRef = useRef<HTMLInputElement>(null);
  const { t } = useTranslation();
  const { register } = useAuth();
  const { showToast } = useUIStore();
  const navigate = useNavigate();
  const codeCooldown = useCountdown();

  const passwordProgress = Math.min(password.length / 6, 1) * 100;
  const passwordMeetsRequirement = password.length >= 6;

  const resolveMessage = (messageKey: string, fallbackMessage?: string) => {
    const translated = t(messageKey);
    return translated === messageKey ? fallbackMessage || t('auth.registerFailed') : translated;
  };

  const getFieldError = (field: RegisterField) => {
    const errorKey = fieldErrors[field];
    return errorKey ? resolveMessage(errorKey) : undefined;
  };

  const getValues = (nextValues: Partial<RegisterFormValues> = {}): RegisterFormValues => ({
    username,
    email,
    verificationCode,
    password,
    confirmPassword,
    ...nextValues,
  });

  const focusField = (field: RegisterField) => {
    const refs = {
      username: usernameInputRef,
      email: emailInputRef,
      verificationCode: verificationCodeInputRef,
      password: passwordInputRef,
      confirmPassword: confirmPasswordInputRef,
    };
    requestAnimationFrame(() => refs[field].current?.focus());
  };

  const setFieldError = (field: RegisterField, errorKey?: string) => {
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

  const validateFields = (fields: RegisterField[], values: RegisterFormValues) => {
    const result = validateRegisterForm(values);
    setFieldErrors((current) => {
      const next = { ...current };
      fields.forEach((field) => {
        const errorKey = result.fieldErrors[field];
        if (errorKey) {
          next[field] = errorKey;
        } else {
          delete next[field];
        }
      });
      return next;
    });
  };

  const handleUsernameChange = (nextUsername: string) => {
    setUsername(nextUsername);
    setFormError('');
    if (fieldErrors.username) {
      validateFields(['username'], getValues({ username: nextUsername }));
    }
  };

  const handleEmailChange = (nextEmail: string) => {
    setEmail(nextEmail);
    setFormError('');
    if (fieldErrors.email) {
      validateFields(['email'], getValues({ email: nextEmail }));
    }
  };

  const handleVerificationCodeChange = (nextCode: string) => {
    const normalizedCode = nextCode.replace(/\D/g, '').slice(0, 6);
    setVerificationCode(normalizedCode);
    setFormError('');
    if (fieldErrors.verificationCode) {
      validateFields(['verificationCode'], getValues({ verificationCode: normalizedCode }));
    }
  };

  const handlePasswordChange = (nextPassword: string) => {
    setPassword(nextPassword);
    setFormError('');
    const fieldsToRefresh: RegisterField[] = [];
    if (fieldErrors.password) fieldsToRefresh.push('password');
    if (fieldErrors.confirmPassword) fieldsToRefresh.push('confirmPassword');
    if (fieldsToRefresh.length > 0) {
      validateFields(fieldsToRefresh, getValues({ password: nextPassword }));
    }
  };

  const handleConfirmPasswordChange = (nextConfirmPassword: string) => {
    setConfirmPassword(nextConfirmPassword);
    setFormError('');
    if (fieldErrors.confirmPassword) {
      validateFields(['confirmPassword'], getValues({ confirmPassword: nextConfirmPassword }));
    }
  };

  const handleSendVerificationCode = async () => {
    setFormError('');
    const emailValidation = validateAuthEmail(email);
    setEmail(emailValidation.value);

    if (!emailValidation.isValid) {
      setFieldErrors((current) => ({ ...current, ...emailValidation.fieldErrors }));
      focusField('email');
      return;
    }

    setFieldError('email');
    setIsCodeSending(true);
    try {
      await authApi.requestRegisterCode({ email: emailValidation.value });
      codeCooldown.start(60);
      showToast(t('auth.verificationCodeSent'), 'success');
      requestAnimationFrame(() => verificationCodeInputRef.current?.focus());
    } catch (error: any) {
      const feedback = mapAuthErrorToForm(error?.message, 'register');
      const message = resolveMessage(feedback.messageKey, feedback.fallbackMessage);

      if (registerFieldOrder.includes(feedback.target as RegisterField)) {
        const target = feedback.target as RegisterField;
        setFieldError(target, feedback.messageKey);
        focusField(target);
      } else {
        setFormError(message);
      }
    } finally {
      setIsCodeSending(false);
    }
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setFormError('');

    const validation = validateRegisterForm({
      username,
      email,
      verificationCode,
      password,
      confirmPassword,
    });
    setUsername(validation.values.username);
    setEmail(validation.values.email);

    if (!validation.isValid) {
      setFieldErrors(validation.fieldErrors);
      const firstInvalidField = registerFieldOrder.find((field) => validation.fieldErrors[field]);
      if (firstInvalidField) {
        focusField(firstInvalidField);
      }
      return;
    }

    setFieldErrors({});
    setIsLoading(true);
    try {
      await register(
        validation.values.username,
        validation.values.email,
        validation.values.password,
        validation.values.verificationCode
      );
      showToast(t('auth.registerSuccess'), 'success');
      navigate('/');
    } catch (error: any) {
      const feedback = mapAuthErrorToForm(error?.message, 'register');
      const message = resolveMessage(feedback.messageKey, feedback.fallbackMessage);

      if (registerFieldOrder.includes(feedback.target as RegisterField)) {
        const target = feedback.target as RegisterField;
        setFieldError(target, feedback.messageKey);
        focusField(target);
      } else {
        setFormError(message);
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleGitHubLogin = () => {
    authApi.startGitHubLogin();
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-neutral-50 dark:bg-neutral-950 px-4 transition-colors">
      <div className="max-w-md w-full bg-white/95 dark:bg-neutral-900/95 backdrop-blur rounded-[28px] shadow-elevated border border-neutral-200 dark:border-neutral-700 p-8 md:p-10">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-serif font-black text-neutral-700 dark:text-neutral-100 text-center mb-2">
            {t('auth.register')}
          </h1>
          <p className="text-sm text-neutral-500 dark:text-neutral-400">
            {t('chat.askAbout')}
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5" noValidate>
          <AuthFormMessage message={formError} />

          <AuthTextField
            ref={usernameInputRef}
            id="register-username"
            label={t('auth.username')}
            type="text"
            value={username}
            onChange={(e) => handleUsernameChange(e.target.value)}
            onBlur={() => validateFields(['username'], getValues())}
            placeholder={t('auth.username')}
            autoComplete="username"
            disabled={isLoading}
            error={getFieldError('username')}
          />

          <AuthTextField
            ref={emailInputRef}
            id="register-email"
            label={t('auth.email')}
            type="email"
            value={email}
            onChange={(e) => handleEmailChange(e.target.value)}
            onBlur={() => validateFields(['email'], getValues())}
            placeholder="your@email.com"
            autoComplete="email"
            inputMode="email"
            disabled={isLoading}
            error={getFieldError('email')}
          />

          <VerificationCodeField
            ref={verificationCodeInputRef}
            id="register-verification-code"
            label={t('auth.verificationCode')}
            value={verificationCode}
            onChange={(e) => handleVerificationCodeChange(e.target.value)}
            onBlur={() => validateFields(['verificationCode'], getValues())}
            disabled={isLoading}
            isSending={isCodeSending}
            cooldownSeconds={codeCooldown.seconds}
            error={getFieldError('verificationCode')}
            onSendCode={handleSendVerificationCode}
            sendLabel={t('auth.sendCode')}
            sendingLabel={t('auth.sendingCode')}
            countdownLabel={(seconds) => t('auth.resendIn', { seconds })}
          />

          <AuthTextField
            ref={passwordInputRef}
            id="register-password"
            label={t('auth.password')}
            type={showPassword ? 'text' : 'password'}
            value={password}
            onChange={(e) => handlePasswordChange(e.target.value)}
            onBlur={() =>
              validateFields(
                confirmPassword ? ['password', 'confirmPassword'] : ['password'],
                getValues()
              )
            }
            placeholder={t('auth.password')}
            autoComplete="new-password"
            disabled={isLoading}
            error={getFieldError('password')}
            hint={
              <div className="flex items-center gap-2 text-xs font-medium leading-5 text-neutral-400 dark:text-neutral-500">
                <span
                  className="h-1.5 w-16 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800"
                  aria-hidden="true"
                >
                  <span
                    className={`block h-full rounded-full transition-all duration-300 ${
                      passwordMeetsRequirement
                        ? 'bg-primary-500'
                        : 'bg-neutral-300 dark:bg-neutral-600'
                    }`}
                    style={{ width: `${passwordProgress}%` }}
                  />
                </span>
                <span>{t('auth.passwordRequirement')}</span>
              </div>
            }
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

          <AuthTextField
            ref={confirmPasswordInputRef}
            id="register-confirm-password"
            label={t('auth.confirmPassword')}
            type={showConfirmPassword ? 'text' : 'password'}
            value={confirmPassword}
            onChange={(e) => handleConfirmPasswordChange(e.target.value)}
            onBlur={() => validateFields(['confirmPassword'], getValues())}
            placeholder={t('auth.confirmPassword')}
            autoComplete="new-password"
            disabled={isLoading}
            error={getFieldError('confirmPassword')}
            trailing={
              <button
                type="button"
                onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                disabled={isLoading}
                aria-label={
                  showConfirmPassword ? t('auth.hideConfirmPassword') : t('auth.showConfirmPassword')
                }
                aria-pressed={showConfirmPassword}
                title={
                  showConfirmPassword ? t('auth.hideConfirmPassword') : t('auth.showConfirmPassword')
                }
                className="inline-flex h-9 w-9 items-center justify-center rounded-full text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-700 focus:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/10 disabled:cursor-not-allowed dark:hover:bg-neutral-700 dark:hover:text-neutral-100"
              >
                {showConfirmPassword ? <EyeOff size={20} /> : <Eye size={20} />}
              </button>
            }
          />

          <button
            type="submit"
            disabled={isLoading}
            className="btn btn-primary w-full justify-center disabled:opacity-60 disabled:cursor-not-allowed"
          >
            {isLoading ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                {t('auth.registering')}
              </>
            ) : (
              t('auth.register')
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
