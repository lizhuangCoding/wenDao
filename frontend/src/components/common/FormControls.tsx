import {
  forwardRef,
  type InputHTMLAttributes,
  type ReactNode,
  type SelectHTMLAttributes,
  type TextareaHTMLAttributes,
} from 'react';
import { cn } from '@/utils';

const controlClassName =
  'w-full rounded-xl border border-neutral-200 bg-white px-3 py-2.5 text-sm text-neutral-800 outline-none transition-colors placeholder:text-neutral-400 focus:border-primary-400 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:placeholder:text-neutral-500';

interface TextInputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'className'> {
  className?: string;
  leading?: ReactNode;
}

export const TextInput = forwardRef<HTMLInputElement, TextInputProps>(
  ({ className, leading, ...props }, ref) => (
    <div className={cn('relative', className)}>
      {leading ? (
        <div className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-neutral-400 dark:text-neutral-500">
          {leading}
        </div>
      ) : null}
      <input
        ref={ref}
        className={cn(controlClassName, leading ? 'pl-9' : '')}
        {...props}
      />
    </div>
  )
);

TextInput.displayName = 'TextInput';

interface SelectInputProps extends SelectHTMLAttributes<HTMLSelectElement> {
  className?: string;
}

export const SelectInput = forwardRef<HTMLSelectElement, SelectInputProps>(
  ({ className, ...props }, ref) => (
    <select
      ref={ref}
      className={cn(controlClassName, 'min-w-36', className)}
      {...props}
    />
  )
);

SelectInput.displayName = 'SelectInput';

interface TextAreaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  className?: string;
}

export const TextArea = forwardRef<HTMLTextAreaElement, TextAreaProps>(
  ({ className, ...props }, ref) => (
    <textarea
      ref={ref}
      className={cn(controlClassName, 'min-h-28 resize-y', className)}
      {...props}
    />
  )
);

TextArea.displayName = 'TextArea';
