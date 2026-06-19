import { ArticleContentSkeleton } from './ArticleContentSkeleton';

const SkeletonLine = ({ className }: { className: string }) => {
  return <div className={`animate-pulse rounded-full bg-neutral-100 dark:bg-neutral-800/70 ${className}`} />;
};

const SkeletonChip = ({ className }: { className: string }) => {
  return <div className={`animate-pulse rounded-full bg-neutral-100 dark:bg-neutral-800/70 ${className}`} />;
};

export const ArticleDetailSkeleton = () => {
  return (
    <div className="flex flex-col justify-center gap-16 lg:flex-row">
      <aside className="hidden w-64 shrink-0 lg:block">
        <div className="sticky top-32 space-y-6 rounded-2xl border border-neutral-200/70 bg-white/70 p-5 shadow-soft backdrop-blur-xl dark:border-neutral-700 dark:bg-[#07111a]/80">
          <SkeletonLine className="h-3 w-24 rounded-md" />
          <div className="space-y-3">
            <SkeletonLine className="h-3 w-4/5 rounded-md" />
            <SkeletonLine className="h-3 w-[88%] rounded-md" />
            <SkeletonLine className="h-3 w-3/4 rounded-md" />
            <SkeletonLine className="h-3 w-[82%] rounded-md" />
            <SkeletonLine className="h-3 w-[70%] rounded-md" />
          </div>
          <div className="pt-4">
            <SkeletonLine className="h-px w-full rounded-none" />
          </div>
          <div className="space-y-3">
            <SkeletonLine className="h-10 w-10 rounded-full" />
            <SkeletonLine className="h-3 w-28 rounded-md" />
            <SkeletonLine className="h-3 w-16 rounded-md" />
          </div>
        </div>
      </aside>

      <article className="flex-1 min-w-0 max-w-reading">
        <header className="mb-16">
          <div className="mb-8 flex items-center gap-4">
            <SkeletonChip className="h-6 w-24" />
            <SkeletonLine className="h-px w-8 rounded-none" />
            <SkeletonChip className="h-3 w-20 rounded-md" />
            <SkeletonChip className="h-3 w-24 rounded-md" />
          </div>

          <div className="space-y-4 mb-10">
            <SkeletonLine className="h-12 w-[92%] rounded-none" />
            <SkeletonLine className="h-12 w-[80%] rounded-none" />
          </div>

          <div className="mb-12 border-l-4 border-neutral-200 pl-6 dark:border-neutral-700">
            <div className="space-y-3 py-1">
              <SkeletonLine className="h-5 w-full rounded-md" />
              <SkeletonLine className="h-5 w-[88%] rounded-md" />
            </div>
          </div>

          <div className="mb-16 aspect-[16/9] rounded-[32px] border border-neutral-200 bg-neutral-50 shadow-soft dark:border-neutral-700 dark:bg-neutral-900/50" />

          <div className="flex gap-4">
            <SkeletonChip className="h-11 w-28" />
            <SkeletonChip className="h-11 w-28" />
          </div>
        </header>

        <div className="article-reading-body">
          <ArticleContentSkeleton />
        </div>

        <div className="mt-16 flex flex-wrap items-center justify-center gap-3 border-y border-neutral-200 py-8 dark:border-neutral-700">
          <SkeletonChip className="h-11 w-36" />
          <SkeletonChip className="h-11 w-36" />
        </div>

        <div className="mt-24 border-t border-neutral-200 pt-16 dark:border-neutral-700">
          <div className="space-y-4">
            <SkeletonLine className="h-4 w-40 rounded-md" />
            <SkeletonLine className="h-4 w-full rounded-md" />
            <SkeletonLine className="h-4 w-[92%] rounded-md" />
            <SkeletonLine className="h-4 w-[84%] rounded-md" />
          </div>
        </div>
      </article>

      <div className="hidden w-64 shrink-0 xl:block" />
    </div>
  );
};
