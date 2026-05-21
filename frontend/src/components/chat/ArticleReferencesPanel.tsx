import type { ChatArticleReference, ChatReferenceGroups } from '@/types';

interface ArticleReferencesPanelProps {
  references: ChatReferenceGroups;
}

const ReferenceGroup = ({
  external = false,
  items,
  title,
}: {
  external?: boolean;
  items: ChatArticleReference[];
  title: string;
}) => {
  if (!items.length) return null;

  return (
    <div className="space-y-2">
      <p className="text-xs font-bold text-neutral-700 dark:text-neutral-200">{title}</p>
      {items.map((reference) => (
        <a
          key={`${reference.title}-${reference.url}`}
          href={reference.url}
          target={external ? '_blank' : undefined}
          rel={external ? 'noreferrer' : undefined}
          className="block rounded-lg border border-neutral-200 dark:border-neutral-600 px-3 py-2 text-sm font-medium text-primary-700 dark:text-primary-300 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-colors no-underline"
        >
          {reference.title}
        </a>
      ))}
    </div>
  );
};

export const ArticleReferencesPanel = ({ references }: ArticleReferencesPanelProps) => {
  if (!references.blog.length && !references.external.length) return null;

  return (
    <div className="mt-4 border-t border-neutral-200 dark:border-neutral-600 pt-4">
      <div className="space-y-4">
        <ReferenceGroup title="参考博主文章" items={references.blog} />
        <ReferenceGroup title="参考外部文章" items={references.external} external />
      </div>
    </div>
  );
};
