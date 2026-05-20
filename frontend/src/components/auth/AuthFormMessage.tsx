import { AlertCircle, CheckCircle2, Info } from 'lucide-react';
import { cn } from '@/utils';

interface AuthFormMessageProps {
  message?: string;
  type?: 'error' | 'success' | 'info';
}

const messageStyles = {
  error: {
    icon: AlertCircle,
    className:
      'border-red-200 bg-red-50 text-red-700 dark:border-red-500/30 dark:bg-red-950/30 dark:text-red-200',
  },
  success: {
    icon: CheckCircle2,
    className:
      'border-primary-200 bg-primary-50 text-primary-700 dark:border-primary-500/30 dark:bg-primary-950/30 dark:text-primary-200',
  },
  info: {
    icon: Info,
    className:
      'border-neutral-200 bg-neutral-50 text-neutral-600 dark:border-neutral-700 dark:bg-neutral-800/70 dark:text-neutral-300',
  },
};

export const AuthFormMessage = ({ message, type = 'error' }: AuthFormMessageProps) => {
  if (!message) return null;

  const style = messageStyles[type];
  const Icon = style.icon;

  return (
    <div
      role={type === 'error' ? 'alert' : 'status'}
      aria-live="polite"
      className={cn(
        'flex items-start gap-2 rounded-2xl border px-4 py-3 text-sm font-medium leading-5 animate-slide-up',
        style.className
      )}
    >
      <Icon className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <span>{message}</span>
    </div>
  );
};
