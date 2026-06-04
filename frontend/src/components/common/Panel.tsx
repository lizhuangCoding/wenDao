import type { HTMLAttributes } from 'react';
import { cn } from '@/utils';

interface PanelProps extends HTMLAttributes<HTMLDivElement> {
  padding?: 'none' | 'sm' | 'md' | 'lg';
}

const paddingClassName: Record<NonNullable<PanelProps['padding']>, string> = {
  none: '',
  sm: 'p-3',
  md: 'p-4',
  lg: 'p-6',
};

export const Panel = ({ className, padding = 'md', ...props }: PanelProps) => (
  <div
    className={cn(
      'rounded-2xl border border-neutral-100 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900',
      paddingClassName[padding],
      className
    )}
    {...props}
  />
);

