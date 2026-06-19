import type { HTMLAttributes } from 'react';
import { cn } from '@/utils';

interface PanelProps extends HTMLAttributes<HTMLDivElement> {
  padding?: 'none' | 'sm' | 'md' | 'lg';
  variant?: 'default' | 'muted' | 'elevated' | 'interactive';
}

const paddingClassName: Record<NonNullable<PanelProps['padding']>, string> = {
  none: '',
  sm: 'p-3',
  md: 'p-4',
  lg: 'p-6',
};

const variantClassName: Record<NonNullable<PanelProps['variant']>, string> = {
  default: 'border-neutral-100 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900',
  muted: 'border-neutral-100 bg-neutral-50/70 dark:border-neutral-800 dark:bg-neutral-900/60',
  elevated: 'border-neutral-100 bg-white shadow-soft dark:border-neutral-800 dark:bg-neutral-900',
  interactive:
    'border-neutral-100 bg-white shadow-sm transition-colors hover:border-primary-200 dark:border-neutral-800 dark:bg-neutral-900 dark:hover:border-primary-500/40',
};

export const Panel = ({ className, padding = 'md', variant = 'default', ...props }: PanelProps) => (
  <div
    className={cn(
      'rounded-2xl border',
      variantClassName[variant],
      paddingClassName[padding],
      className
    )}
    {...props}
  />
);
