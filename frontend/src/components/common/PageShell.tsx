import type { HTMLAttributes } from 'react';
import { cn } from '@/utils';

type PageShellWidth = 'narrow' | 'default' | 'wide' | 'display';
type PageShellPadding = 'sm' | 'md' | 'lg';

interface PageShellProps extends HTMLAttributes<HTMLDivElement> {
  width?: PageShellWidth;
  padding?: PageShellPadding;
}

const widthClassName: Record<PageShellWidth, string> = {
  narrow: 'max-w-4xl',
  default: 'max-w-6xl',
  wide: 'max-w-7xl',
  display: 'max-w-display',
};

const paddingClassName: Record<PageShellPadding, string> = {
  sm: 'px-5 py-10 sm:px-8 sm:py-12 lg:px-10',
  md: 'px-5 py-14 sm:px-8 sm:py-16 lg:px-10',
  lg: 'px-5 py-16 sm:px-10 sm:py-20 lg:px-12',
};

export const PageShell = ({
  width = 'default',
  padding = 'md',
  className,
  ...props
}: PageShellProps) => (
  <div
    className={cn('relative z-10 mx-auto w-full', widthClassName[width], paddingClassName[padding], className)}
    {...props}
  />
);
