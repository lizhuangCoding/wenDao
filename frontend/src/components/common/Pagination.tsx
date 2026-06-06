import { ChevronLeft, ChevronRight } from 'lucide-react';

interface PaginationProps {
  page: number;
  totalPages: number;
  total?: number;
  pageSize?: number;
  pageSizeOptions?: number[];
  onChange: (page: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
  previousLabel?: string;
  nextLabel?: string;
  className?: string;
}

const getVisiblePages = (page: number, totalPages: number) => {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, index) => index + 1);
  }
  const pages = new Set([1, totalPages, page - 1, page, page + 1]);
  return Array.from(pages)
    .filter((item) => item >= 1 && item <= totalPages)
    .sort((a, b) => a - b);
};

export const Pagination = ({
  page,
  totalPages,
  total,
  pageSize,
  pageSizeOptions = [10, 15, 20, 50],
  onChange,
  onPageSizeChange,
  previousLabel = '上一页',
  nextLabel = '下一页',
  className = '',
}: PaginationProps) => {
  const safeTotalPages = Math.max(1, totalPages);
  const safePage = Math.min(Math.max(1, page), safeTotalPages);
  const visiblePages = getVisiblePages(safePage, safeTotalPages);

  return (
    <nav className={`flex flex-col items-center gap-3 ${className}`} aria-label="分页导航">
      <div className="flex flex-wrap items-center justify-center gap-3">
        <button
          type="button"
          onClick={() => onChange(Math.max(1, safePage - 1))}
          disabled={safePage === 1}
          className="inline-flex items-center gap-2 rounded-xl bg-neutral-100 px-4 py-2 text-sm font-medium text-neutral-700 transition-all hover:bg-neutral-200 disabled:cursor-not-allowed disabled:opacity-40 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
        >
          <ChevronLeft className="h-4 w-4" />
          {previousLabel}
        </button>

        <div className="flex items-center gap-2">
          {visiblePages.map((item, index) => {
            const previous = visiblePages[index - 1];
            const showGap = previous !== undefined && item - previous > 1;
            return (
              <div key={item} className="flex items-center gap-2">
                {showGap && <span className="px-1 text-sm font-medium text-neutral-300 dark:text-neutral-600">...</span>}
                <button
                  type="button"
                  onClick={() => onChange(item)}
                  className={`h-10 min-w-10 rounded-xl px-3 text-sm font-bold transition-all ${
                    safePage === item
                      ? 'bg-primary-600 text-white shadow-sm'
                      : 'bg-neutral-100 text-neutral-500 hover:bg-neutral-200 hover:text-neutral-900 dark:bg-neutral-800 dark:text-neutral-400 dark:hover:bg-neutral-700 dark:hover:text-neutral-100'
                  }`}
                  aria-current={safePage === item ? 'page' : undefined}
                >
                  {item}
                </button>
              </div>
            );
          })}
        </div>

        <button
          type="button"
          onClick={() => onChange(Math.min(safeTotalPages, safePage + 1))}
          disabled={safePage === safeTotalPages}
          className="inline-flex items-center gap-2 rounded-xl bg-neutral-100 px-4 py-2 text-sm font-medium text-neutral-700 transition-all hover:bg-neutral-200 disabled:cursor-not-allowed disabled:opacity-40 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
        >
          {nextLabel}
          <ChevronRight className="h-4 w-4" />
        </button>
      </div>

      <div className="flex items-center justify-center gap-4">
        {total !== undefined && (
          <span className="text-xs font-medium text-neutral-400 dark:text-neutral-500">
            共 {total} 条
          </span>
        )}
        {pageSize !== undefined && onPageSizeChange && (
          <select
            value={pageSize}
            onChange={(e) => onPageSizeChange(Number(e.target.value))}
            className="rounded-lg border border-neutral-100 dark:border-neutral-700 bg-neutral-50 dark:bg-neutral-800 px-2 py-1 text-xs font-medium text-neutral-600 dark:text-neutral-400 focus:outline-none focus:border-primary-500 cursor-pointer"
            aria-label="每页条数"
          >
            {pageSizeOptions.map((size) => (
              <option key={size} value={size}>
                {size} 条/页
              </option>
            ))}
          </select>
        )}
        <span className="text-xs font-medium text-neutral-400 dark:text-neutral-500">
          {safePage} / {safeTotalPages}
        </span>
      </div>
    </nav>
  );
};
