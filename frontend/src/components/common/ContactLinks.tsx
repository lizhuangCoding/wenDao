import { ExternalLink, Mail } from 'lucide-react';
import { cn } from '@/utils';

interface ContactLinksProps {
  className?: string;
}

export const ContactLinks = ({ className }: ContactLinksProps) => {
  return (
    <div className={cn('grid gap-3 sm:grid-cols-2', className)}>
      <a
        href="mailto:3174285493@qq.com"
        className="group flex items-center gap-3 rounded-2xl border border-neutral-200 bg-white px-4 py-3 text-left transition-colors hover:border-primary-200 hover:bg-primary-50/50 dark:border-neutral-800 dark:bg-neutral-950/40 dark:hover:border-primary-900 dark:hover:bg-neutral-900"
      >
        <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-900 text-white transition-transform group-hover:scale-105 dark:bg-neutral-100 dark:text-neutral-900">
          <Mail className="h-5 w-5" />
        </span>
        <span className="min-w-0">
          <span className="block text-[11px] font-black uppercase tracking-[0.28em] text-neutral-400 dark:text-neutral-500">
            QQ 邮箱
          </span>
          <span className="mt-1 block truncate text-sm font-semibold text-neutral-900 dark:text-neutral-100">
            3174285493@qq.com
          </span>
        </span>
      </a>

      <a
        href="https://github.com/lizhuangCoding"
        target="_blank"
        rel="noreferrer"
        className="group flex items-center gap-3 rounded-2xl border border-neutral-200 bg-white px-4 py-3 text-left transition-colors hover:border-primary-200 hover:bg-primary-50/50 dark:border-neutral-800 dark:bg-neutral-950/40 dark:hover:border-primary-900 dark:hover:bg-neutral-900"
      >
        <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-900 text-white transition-transform group-hover:scale-105 dark:bg-neutral-100 dark:text-neutral-900">
          <ExternalLink className="h-5 w-5" />
        </span>
        <span className="min-w-0">
          <span className="block text-[11px] font-black uppercase tracking-[0.28em] text-neutral-400 dark:text-neutral-500">
            GitHub
          </span>
          <span className="mt-1 block truncate text-sm font-semibold text-neutral-900 dark:text-neutral-100">
            lizhuangCoding
          </span>
        </span>
      </a>
    </div>
  );
};
