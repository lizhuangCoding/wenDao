import type { ButtonHTMLAttributes } from 'react';
import { cn } from '@/utils';

interface ToggleSwitchProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'aria-checked' | 'role'> {
  checked: boolean;
}

export const ToggleSwitch = ({ checked, className, ...props }: ToggleSwitchProps) => (
  <button
    type="button"
    role="switch"
    aria-checked={checked}
    className={cn(
      'relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/10 disabled:cursor-not-allowed disabled:opacity-60',
      checked ? 'bg-primary-500' : 'bg-neutral-200 dark:bg-neutral-700',
      className
    )}
    {...props}
  >
    <span
      className={cn(
        'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
        checked ? 'translate-x-6' : 'translate-x-1'
      )}
    />
  </button>
);

