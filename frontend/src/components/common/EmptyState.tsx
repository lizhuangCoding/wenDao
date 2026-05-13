interface EmptyStateProps {
  title: string;
  description?: string;
  className?: string;
}

export const EmptyState = ({ title, description, className = '' }: EmptyStateProps) => {
  return (
    <div
      className={`rounded-2xl border border-dashed border-neutral-200 bg-neutral-50 px-6 py-12 text-center dark:border-neutral-800 dark:bg-neutral-900/60 ${className}`}
    >
      <div className="text-base font-semibold text-neutral-600 dark:text-neutral-300">{title}</div>
      {description && (
        <div className="mt-2 text-sm text-neutral-400 dark:text-neutral-500">{description}</div>
      )}
    </div>
  );
};
