import type { ReactNode } from 'react';
import { motion } from 'framer-motion';
import { cn } from '@/utils';

interface PageHeaderProps {
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  className?: string;
  eyebrow?: ReactNode;
  tone?: 'default' | 'admin';
}

export const PageHeader = ({ title, description, actions, className, eyebrow, tone = 'default' }: PageHeaderProps) => (
  <motion.div
    initial={{ opacity: 0, y: -16 }}
    animate={{ opacity: 1, y: 0 }}
    className={cn('flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between', className)}
  >
    <div className="min-w-0">
      {eyebrow ? (
        <p className="mb-2 text-xs font-bold uppercase tracking-[0.18em] text-primary-600 dark:text-primary-400">
          {eyebrow}
        </p>
      ) : null}
      <h1
        className={cn(
          'text-3xl font-bold text-neutral-900 dark:text-neutral-100',
          tone === 'default' ? 'font-serif sm:text-4xl' : 'font-sans tracking-normal'
        )}
      >
        {title}
      </h1>
      {description ? (
        <p className="mt-2 max-w-2xl text-sm leading-6 text-neutral-500 dark:text-neutral-400">
          {description}
        </p>
      ) : null}
    </div>
    {actions ? (
      <div className="flex w-full flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center lg:w-auto lg:justify-end">
        {actions}
      </div>
    ) : null}
  </motion.div>
);
