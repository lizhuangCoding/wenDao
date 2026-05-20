import { forwardRef, type InputHTMLAttributes, type ReactNode } from 'react';
import { AlertCircle } from 'lucide-react';
import { cn } from '@/utils';

interface AuthTextFieldProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'className'> {
  id: string;
  label: string;
  error?: string;
  hint?: ReactNode;
  trailing?: ReactNode;
  inputClassName?: string;
}

export const AuthTextField = forwardRef<HTMLInputElement, AuthTextFieldProps>(
  ({ id, label, error, hint, trailing, inputClassName, ...inputProps }, ref) => {
    const errorId = `${id}-error`;
    const hintId = `${id}-hint`;
    const describedBy = error ? errorId : hint ? hintId : undefined;

    return (
      <div className="space-y-2">
        <label
          htmlFor={id}
          className="block text-sm font-medium text-neutral-700 dark:text-neutral-300"
        >
          {label}
        </label>

        <div className="relative">
          <input
            ref={ref}
            id={id}
            aria-invalid={Boolean(error)}
            aria-describedby={describedBy}
            className={cn(
              'input w-full',
              trailing ? 'pr-14' : '',
              error
                ? 'border-red-300 bg-red-50/70 focus:border-red-500 focus:ring-red-500/10 dark:border-red-500/60 dark:bg-red-950/20 dark:focus:border-red-400'
                : '',
              inputClassName
            )}
            {...inputProps}
          />
          {trailing ? (
            <div className="absolute right-3 top-1/2 -translate-y-1/2">{trailing}</div>
          ) : null}
        </div>

        <div className="min-h-[1.25rem]" aria-live="polite">
          {error ? (
            <p
              id={errorId}
              className="flex items-center gap-1.5 text-xs font-semibold leading-5 text-red-600 animate-slide-up dark:text-red-300"
            >
              <AlertCircle className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
              <span>{error}</span>
            </p>
          ) : hint ? (
            <div id={hintId}>{hint}</div>
          ) : null}
        </div>
      </div>
    );
  }
);

AuthTextField.displayName = 'AuthTextField';
