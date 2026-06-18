import type { ComponentProps, ReactNode } from 'react';
import { cn } from '@/utils';

type DataTableColumnWidth = 'select' | 'compact' | 'medium' | 'wide' | 'actionsCompact' | 'actions' | 'actionsWide';

const columnWidthClasses: Record<DataTableColumnWidth, string> = {
  select: 'w-12 sm:w-14',
  compact: 'w-24 sm:w-28',
  medium: 'w-32 sm:w-40',
  wide: 'w-[34%]',
  actionsCompact: 'w-24 sm:w-28',
  actions: 'w-32 sm:w-36',
  actionsWide: 'w-48 sm:w-56',
};

interface DataTableProps {
  children: ReactNode;
  className?: string;
  emptyState?: ReactNode;
  layout?: 'fixed' | 'auto';
  minWidth?: string;
  stretch?: boolean;
  tableClassName?: string;
}

export const DataTable = ({
  children,
  className,
  emptyState,
  layout = 'fixed',
  minWidth = '880px',
  stretch = true,
  tableClassName,
}: DataTableProps) => (
  <div
    className={cn(
      'overflow-hidden rounded-2xl border border-neutral-100 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900',
      className
    )}
  >
    <div className="overflow-x-auto">
      <table
        className={cn(
          'border-collapse text-left',
          stretch ? 'w-full' : '',
          layout === 'fixed' ? 'table-fixed' : 'table-auto',
          tableClassName
        )}
        style={{ minWidth, width: stretch ? undefined : minWidth }}
      >
        {children}
      </table>
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

interface DataTableHeaderCellProps extends Omit<ComponentProps<'th'>, 'width'> {
  align?: 'left' | 'right' | 'center';
  width?: DataTableColumnWidth;
  nowrap?: boolean;
  truncate?: boolean;
}

export const DataTableHeaderCell = ({
  align = 'left',
  className,
  width,
  nowrap = true,
  truncate = false,
  ...props
}: DataTableHeaderCellProps) => (
  <th
    className={cn(
      'px-4 py-3 text-sm font-semibold text-neutral-600 dark:text-neutral-400 sm:px-6 sm:py-4',
      width ? columnWidthClasses[width] : '',
      nowrap ? 'whitespace-nowrap' : '',
      truncate ? 'overflow-hidden text-ellipsis' : '',
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

interface DataTableCellProps extends Omit<ComponentProps<'td'>, 'width'> {
  align?: 'left' | 'right' | 'center';
  width?: DataTableColumnWidth;
  nowrap?: boolean;
  truncate?: boolean;
}

export const DataTableCell = ({
  align = 'left',
  className,
  width,
  nowrap = false,
  truncate = false,
  ...props
}: DataTableCellProps) => (
  <td
    className={cn(
      'px-4 py-3 text-sm text-neutral-500 dark:text-neutral-400 sm:px-6 sm:py-4',
      width ? columnWidthClasses[width] : '',
      nowrap ? 'whitespace-nowrap' : '',
      truncate ? 'overflow-hidden text-ellipsis whitespace-nowrap' : '',
      align === 'right' ? 'text-right' : '',
      align === 'center' ? 'text-center' : '',
      className
    )}
    {...props}
  />
);
