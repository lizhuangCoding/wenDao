import { cn } from '@/utils';

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'success';
export type ButtonSize = 'sm' | 'md' | 'lg' | 'icon';

interface ButtonClassNameOptions {
  variant?: ButtonVariant;
  size?: ButtonSize;
  className?: string;
}

const variantClassName: Record<ButtonVariant, string> = {
  primary:
    'bg-primary-600 text-white shadow-sm hover:bg-primary-700 dark:bg-primary-500 dark:hover:bg-primary-400',
  secondary:
    'bg-neutral-100 text-neutral-700 hover:bg-neutral-200 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-700',
  ghost:
    'bg-transparent text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900 dark:text-neutral-300 dark:hover:bg-neutral-800 dark:hover:text-neutral-100',
  danger:
    'bg-red-600 text-white shadow-sm hover:bg-red-700 dark:bg-red-500 dark:hover:bg-red-400',
  success:
    'bg-green-600 text-white shadow-sm hover:bg-green-700 dark:bg-green-500 dark:hover:bg-green-400',
};

const sizeClassName: Record<ButtonSize, string> = {
  sm: 'h-8 rounded-lg px-3 text-xs',
  md: 'h-10 rounded-xl px-4 text-sm',
  lg: 'h-11 rounded-xl px-5 text-sm',
  icon: 'h-10 w-10 rounded-xl p-0',
};

export const getButtonClassName = ({
  variant = 'primary',
  size = 'md',
  className,
}: ButtonClassNameOptions = {}) =>
  cn(
    'inline-flex items-center justify-center gap-2 font-semibold transition-colors focus:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/10 disabled:cursor-not-allowed disabled:opacity-60',
    variantClassName[variant],
    sizeClassName[size],
    className
  );

