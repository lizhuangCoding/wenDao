import type { HTMLAttributes } from 'react';
import { cn } from '@/utils';

export type StatusBadgeVariant = 'success' | 'warning' | 'danger' | 'info' | 'primary' | 'neutral';

interface StatusBadgeProps extends HTMLAttributes<HTMLSpanElement> {
  variant?: StatusBadgeVariant;
}

const variantClassName: Record<StatusBadgeVariant, string> = {
  success: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300',
  warning: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300',
  danger: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300',
  info: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300',
  primary: 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300',
  neutral: 'bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300',
};

export const StatusBadge = ({
  variant = 'neutral',
  className,
  ...props
}: StatusBadgeProps) => (
  <span
    className={cn('inline-flex items-center rounded-full px-3 py-1.5 text-xs font-medium', variantClassName[variant], className)}
    {...props}
  />
);

