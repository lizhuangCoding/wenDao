import { forwardRef, type ChangeEventHandler, type FocusEventHandler } from 'react';
import { Loader2 } from 'lucide-react';
import { AuthTextField } from './AuthTextField';

interface VerificationCodeFieldProps {
  id: string;
  label: string;
  value: string;
  onChange: ChangeEventHandler<HTMLInputElement>;
  onBlur?: FocusEventHandler<HTMLInputElement>;
  onSendCode: () => void;
  error?: string;
  disabled?: boolean;
  isSending?: boolean;
  cooldownSeconds?: number;
  sendLabel: string;
  sendingLabel: string;
  countdownLabel: (seconds: number) => string;
}

export const VerificationCodeField = forwardRef<HTMLInputElement, VerificationCodeFieldProps>(
  (
    {
      id,
      label,
      value,
      onChange,
      onBlur,
      onSendCode,
      error,
      disabled,
      isSending = false,
      cooldownSeconds = 0,
      sendLabel,
      sendingLabel,
      countdownLabel,
    },
    ref
  ) => {
    const isCoolingDown = cooldownSeconds > 0;
    const buttonDisabled = disabled || isSending || isCoolingDown;

    return (
      <AuthTextField
        ref={ref}
        id={id}
        label={label}
        type="text"
        value={value}
        onChange={onChange}
        onBlur={onBlur}
        placeholder="123456"
        autoComplete="one-time-code"
        inputMode="numeric"
        disabled={disabled}
        error={error}
        inputClassName="pr-32"
        trailing={
          <button
            type="button"
            onClick={onSendCode}
            disabled={buttonDisabled}
            className="inline-flex h-8 min-w-24 items-center justify-center gap-1 rounded-full bg-neutral-900 px-3 text-xs font-semibold text-white transition-colors hover:bg-neutral-700 disabled:cursor-not-allowed disabled:bg-neutral-200 disabled:text-neutral-500 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-white dark:disabled:bg-neutral-700 dark:disabled:text-neutral-400"
          >
            {isSending ? (
              <>
                <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
                {sendingLabel}
              </>
            ) : isCoolingDown ? (
              countdownLabel(cooldownSeconds)
            ) : (
              sendLabel
            )}
          </button>
        }
      />
    );
  }
);

VerificationCodeField.displayName = 'VerificationCodeField';
