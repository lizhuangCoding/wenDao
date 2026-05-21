import { useRef, useState, type FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { Eye, EyeOff, Loader2 } from 'lucide-react';
import { authApi } from '@/api';
import { useCountdown } from '@/hooks';
import { useUIStore } from '@/store';
import {
  mapAuthErrorToForm,
  validateAuthEmail,
  validatePasswordResetForm,
  type AuthField,
  type PasswordResetFormValues,
} from '@/utils/authForm';
import { AuthFormMessage } from './AuthFormMessage';
import { AuthTextField } from './AuthTextField';
import { VerificationCodeField } from './VerificationCodeField';

type ResetField = Extract<AuthField, 'email' | 'verificationCode' | 'password' | 'confirmPassword'>;

const resetFieldOrder: ResetField[] = ['email', 'verificationCode', 'password', 'confirmPassword'];

interface PasswordResetFormProps {
  initialEmail?: string;
  onBack: () => void;
  onSuccess: (email: string) => void;
}

export const PasswordResetForm = ({ initialEmail = '', onBack, onSuccess }: PasswordResetFormProps) => {
  const [email, setEmail] = useState(initialEmail);
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [verificationCode, setVerificationCode] = useState('');
  const [fieldErrors, setFieldErrors] = useState<Partial<Record<AuthField, string>>>({});
  const [formError, setFormError] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [isCodeSending, setIsCodeSending] = useState(false);
  const emailInputRef = useRef<HTMLInputElement>(null);
  const passwordInputRef = useRef<HTMLInputElement>(null);
  const confirmPasswordInputRef = useRef<HTMLInputElement>(null);
  const verificationCodeInputRef = useRef<HTMLInputElement>(null);
  const { t } = useTranslation();
  const { showToast } = useUIStore();
  const codeCooldown = useCountdown();

  const resolveMessage = (messageKey: string, fallbackMessage?: string) => {
    const translated = t(messageKey);
    return translated === messageKey ? fallbackMessage || t('auth.passwordResetFailed') : translated;
  };

  const getFieldError = (field: AuthField) => {
    const errorKey = fieldErrors[field];
    return errorKey ? resolveMessage(errorKey) : undefined;
  };

  const getValues = (nextValues: Partial<PasswordResetFormValues> = {}): PasswordResetFormValues => ({
    email,
    password,
    confirmPassword,
    verificationCode,
    ...nextValues,
  });

  const focusField = (field: ResetField) => {
    const refs = {
      email: emailInputRef,
      password: passwordInputRef,
      confirmPassword: confirmPasswordInputRef,
      verificationCode: verificationCodeInputRef,
    };
    requestAnimationFrame(() => refs[field].current?.focus());
  };

  const setFieldError = (field: AuthField, errorKey?: string) => {
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

  const validateFields = (fields: ResetField[], values: PasswordResetFormValues) => {
    const result = validatePasswordResetForm(values);
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

  const handleEmailChange = (nextEmail: string) => {
    setEmail(nextEmail);
    setFormError('');
    if (fieldErrors.email) {
      validateFields(['email'], getValues({ email: nextEmail }));
    }
  };

  const handlePasswordChange = (nextPassword: string) => {
    setPassword(nextPassword);
    setFormError('');
    const fieldsToRefresh: ResetField[] = [];
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

  const handleVerificationCodeChange = (nextCode: string) => {
    const normalizedCode = nextCode.replace(/\D/g, '').slice(0, 6);
    setVerificationCode(normalizedCode);
    setFormError('');
    if (fieldErrors.verificationCode) {
      validateFields(['verificationCode'], getValues({ verificationCode: normalizedCode }));
    }
  };

  const handleSendCode = async () => {
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
      await authApi.requestPasswordResetCode({ email: emailValidation.value });
      codeCooldown.start(60);
      showToast(t('auth.passwordResetCodeSent'), 'success');
      requestAnimationFrame(() => verificationCodeInputRef.current?.focus());
    } catch (error: any) {
      const feedback = mapAuthErrorToForm(error?.message, 'passwordReset');
      const message = resolveMessage(feedback.messageKey, feedback.fallbackMessage);

      if (resetFieldOrder.includes(feedback.target as ResetField)) {
        const target = feedback.target as ResetField;
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

    const validation = validatePasswordResetForm({ email, password, confirmPassword, verificationCode });
    setEmail(validation.values.email);

    if (!validation.isValid) {
      setFieldErrors(validation.fieldErrors);
      const firstInvalidField = resetFieldOrder.find((field) => validation.fieldErrors[field]);
      if (firstInvalidField) {
        focusField(firstInvalidField);
      }
      return;
    }

    setFieldErrors({});
    setIsLoading(true);
    try {
      await authApi.confirmPasswordReset({
        email: validation.values.email,
        password: validation.values.password,
        verification_code: validation.values.verificationCode,
      });
      showToast(t('auth.passwordResetSuccess'), 'success');
      onSuccess(validation.values.email);
    } catch (error: any) {
      const feedback = mapAuthErrorToForm(error?.message, 'passwordReset');
      const message = resolveMessage(feedback.messageKey, feedback.fallbackMessage);

      if (resetFieldOrder.includes(feedback.target as ResetField)) {
        const target = feedback.target as ResetField;
        setFieldError(target, feedback.messageKey);
        focusField(target);
      } else {
        setFormError(message);
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <div className="text-center mb-8">
        <h1 className="text-3xl font-serif font-black text-neutral-700 dark:text-neutral-100 mb-2">
          {t('auth.resetPassword')}
        </h1>
        <p className="text-sm text-neutral-500 dark:text-neutral-400">
          {t('auth.resetPasswordHint')}
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-5" noValidate>
        <AuthFormMessage message={formError} />

        <AuthTextField
          ref={emailInputRef}
          id="reset-email"
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
          id="reset-verification-code"
          label={t('auth.verificationCode')}
          value={verificationCode}
          onChange={(e) => handleVerificationCodeChange(e.target.value)}
          onBlur={() => validateFields(['verificationCode'], getValues())}
          disabled={isLoading}
          isSending={isCodeSending}
          cooldownSeconds={codeCooldown.seconds}
          error={getFieldError('verificationCode')}
          onSendCode={handleSendCode}
          sendLabel={t('auth.sendCode')}
          sendingLabel={t('auth.sendingCode')}
          countdownLabel={(seconds) => t('auth.resendIn', { seconds })}
        />

        <AuthTextField
          ref={passwordInputRef}
          id="reset-password"
          label={t('auth.newPassword')}
          type={showPassword ? 'text' : 'password'}
          value={password}
          onChange={(e) => handlePasswordChange(e.target.value)}
          onBlur={() =>
            validateFields(confirmPassword ? ['password', 'confirmPassword'] : ['password'], getValues())
          }
          placeholder={t('auth.newPassword')}
          autoComplete="new-password"
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

        <AuthTextField
          ref={confirmPasswordInputRef}
          id="reset-confirm-password"
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
              {t('auth.resettingPassword')}
            </>
          ) : (
            t('auth.resetPassword')
          )}
        </button>
      </form>

      <p className="text-center text-neutral-600 dark:text-neutral-400 mt-6">
        <button
          type="button"
          onClick={onBack}
          className="text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300 transition-colors"
        >
          {t('auth.backToLogin')}
        </button>
      </p>
    </>
  );
};

