import { useTranslation } from 'react-i18next';
import { ExternalLink, Link2, Mail, MessageCircleMore } from 'lucide-react';
import { cn } from '@/utils';
import type { ContactLink } from '@/types';
import { defaultContactLinks } from './contactLinksData';

interface ContactLinksProps {
  className?: string;
  links?: ContactLink[];
}

const iconByType: Record<string, typeof Mail> = {
  email: Mail,
  github: Link2,
  wechat: MessageCircleMore,
  link: Link2,
};

const normalizeUrl = (link: ContactLink) => {
  const value = link.value.trim();
  if (link.url?.trim()) return link.url.trim();
  if (link.type === 'email' && value) return `mailto:${value}`;
  if (link.type === 'github' && value) return `https://github.com/${value.replace(/^@/, '')}`;
  return '';
};

export const ContactLinks = ({ className, links = defaultContactLinks }: ContactLinksProps) => {
  const { t } = useTranslation();
  return (
    <div className={cn('grid gap-3 sm:grid-cols-2', className)}>
      {links
        .filter((link) => link.enabled)
        .slice()
        .sort((left, right) => left.sort_order - right.sort_order)
        .map((link) => {
          const Icon = iconByType[link.type] || ExternalLink;
          const href = normalizeUrl(link);
          const cardClassName =
            'group flex items-center gap-3 rounded-2xl border border-neutral-200 bg-white px-4 py-3 text-left transition-colors hover:border-primary-200 hover:bg-primary-50/50 dark:border-neutral-800 dark:bg-neutral-950/40 dark:hover:border-primary-900 dark:hover:bg-neutral-900';
          const iconClassName =
            'flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-900 text-white transition-transform group-hover:scale-105 dark:bg-neutral-100 dark:text-neutral-900';

          const content = (
            <>
              <span className={iconClassName}>
                <Icon className="h-5 w-5" />
              </span>
              <span className="min-w-0">
                <span className="block text-[11px] font-black uppercase tracking-[0.28em] text-neutral-400 dark:text-neutral-500">
                  {link.label || t('common.default')}
                </span>
                <span className="mt-1 block truncate text-sm font-semibold text-neutral-900 dark:text-neutral-100">
                  {link.value}
                </span>
              </span>
            </>
          );

          if (!href) {
            return (
              <div key={`${link.type}-${link.sort_order}-${link.value}`} className={cardClassName}>
                {content}
              </div>
            );
          }

          return (
            <a
              key={`${link.type}-${link.sort_order}-${link.value}`}
              href={href}
              target={href.startsWith('http') ? '_blank' : undefined}
              rel={href.startsWith('http') ? 'noreferrer' : undefined}
              className={cardClassName}
            >
              {content}
            </a>
          );
        })}
    </div>
  );
};
