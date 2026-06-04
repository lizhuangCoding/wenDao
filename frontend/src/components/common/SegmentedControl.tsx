import type { ReactNode } from 'react';
import { cn } from '@/utils';

interface SegmentedControlItem<T extends string> {
  label: ReactNode;
  value: T;
  disabled?: boolean;
}

interface SegmentedControlProps<T extends string> {
  value: T;
  items: Array<SegmentedControlItem<T>>;
  onChange: (value: T) => void;
  className?: string;
}

export const SegmentedControl = <T extends string>({
  value,
  items,
  onChange,
  className,
}: SegmentedControlProps<T>) => (
  <div
    className={cn(
      'grid gap-1 rounded-xl bg-neutral-100 p-1 dark:bg-neutral-800',
      className
    )}
    style={{ gridTemplateColumns: `repeat(${items.length}, minmax(0, 1fr))` }}
  >
    {items.map((item) => {
      const isActive = item.value === value;
      return (
        <button
          key={item.value}
          type="button"
          onClick={() => onChange(item.value)}
          disabled={item.disabled}
          className={cn(
            'min-w-20 rounded-lg px-3 py-1.5 text-xs font-semibold transition-all disabled:cursor-not-allowed disabled:opacity-60',
            isActive
              ? 'bg-white text-primary-600 shadow-sm dark:bg-neutral-700 dark:text-primary-300'
              : 'text-neutral-500 hover:text-neutral-800 dark:text-neutral-400 dark:hover:text-neutral-100'
          )}
        >
          {item.label}
        </button>
      );
    })}
  </div>
);

