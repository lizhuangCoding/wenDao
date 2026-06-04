import type { ComponentProps, ReactNode } from 'react';
import { cn } from '@/utils';

interface DataTableProps {
  children: ReactNode;
  className?: string;
  emptyState?: ReactNode;
  tableClassName?: string;
}

export const DataTable = ({ children, className, emptyState, tableClassName }: DataTableProps) => (
  <div
    className={cn(
      'overflow-hidden rounded-2xl border border-neutral-100 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900',
      className
    )}
  >
    <div className="overflow-x-auto">
      <table className={cn('w-full border-collapse text-left', tableClassName)}>{children}</table>
    </div>
    {emptyState}
  </div>
);

export const DataTableHeadRow = ({ className, ...props }: ComponentProps<'tr'>) => (
  <tr
    className={cn(
      'border-b border-neutral-100 bg-neutral-50 dark:border-neutral-800 dark:bg-neutral-800/50',
      className
    )}
    {...props}
  />
);

interface DataTableHeaderCellProps extends ComponentProps<'th'> {
  align?: 'left' | 'right' | 'center';
}

export const DataTableHeaderCell = ({
  align = 'left',
  className,
  ...props
}: DataTableHeaderCellProps) => (
  <th
    className={cn(
      'px-6 py-4 text-sm font-semibold text-neutral-600 dark:text-neutral-400',
      align === 'right' ? 'text-right' : '',
      align === 'center' ? 'text-center' : '',
      className
    )}
    {...props}
  />
);

export const DataTableBody = ({ className, ...props }: ComponentProps<'tbody'>) => (
  <tbody className={cn('divide-y divide-neutral-100 dark:divide-neutral-800', className)} {...props} />
);

export const DataTableRow = ({ className, ...props }: ComponentProps<'tr'>) => (
  <tr
    className={cn('transition-colors hover:bg-neutral-50 dark:hover:bg-neutral-800/50', className)}
    {...props}
  />
);

interface DataTableCellProps extends ComponentProps<'td'> {
  align?: 'left' | 'right' | 'center';
}

export const DataTableCell = ({ align = 'left', className, ...props }: DataTableCellProps) => (
  <td
    className={cn(
      'px-6 py-4 text-sm text-neutral-500 dark:text-neutral-400',
      align === 'right' ? 'text-right' : '',
      align === 'center' ? 'text-center' : '',
      className
    )}
    {...props}
  />
);
