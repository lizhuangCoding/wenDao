import { Trash2 } from 'lucide-react';

interface BulkActionBarProps {
  selectedCount: number;
  onDelete: () => void;
  onClear: () => void;
  isDeleting?: boolean;
  deleteLabel?: string;
}

export const BulkActionBar = ({
  selectedCount,
  onDelete,
  onClear,
  isDeleting = false,
  deleteLabel = '批量删除',
}: BulkActionBarProps) => {
  if (selectedCount <= 0) {
    return null;
  }

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-red-100 bg-red-50 px-4 py-3 dark:border-red-900/40 dark:bg-red-950/20">
      <span className="text-sm font-semibold text-red-700 dark:text-red-300">
        已选择 {selectedCount} 项
      </span>
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onClear}
          className="rounded-xl px-3 py-2 text-xs font-bold text-red-500 transition-colors hover:bg-red-100 dark:text-red-300 dark:hover:bg-red-900/30"
        >
          取消选择
        </button>
        <button
          type="button"
          onClick={onDelete}
          disabled={isDeleting}
          className="inline-flex items-center gap-2 rounded-xl bg-red-600 px-4 py-2 text-xs font-bold text-white transition-colors hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-60"
        >
          <Trash2 className="h-4 w-4" />
          {deleteLabel}
        </button>
      </div>
    </div>
  );
};
