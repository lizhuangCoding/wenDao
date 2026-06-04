import type { ReactNode } from 'react';
import { motion } from 'framer-motion';
import { cn } from '@/utils';

interface PageHeaderProps {
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  className?: string;
}

export const PageHeader = ({ title, description, actions, className }: PageHeaderProps) => (
  <motion.div
    initial={{ opacity: 0, y: -16 }}
    animate={{ opacity: 1, y: 0 }}
    className={cn('flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between', className)}
  >
    <div className="min-w-0">
      <h1 className="text-3xl font-serif font-bold text-neutral-800 dark:text-neutral-100">
        {title}
      </h1>
      {description ? (
        <p className="mt-1 text-sm font-medium text-neutral-500 dark:text-neutral-400">
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
