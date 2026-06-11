const SkeletonLine = ({ className }: { className: string }) => {
  return <div className={`animate-pulse rounded-full bg-neutral-100 dark:bg-neutral-800/70 ${className}`} />;
};

export const ArticleContentSkeleton = () => {
  return (
    <div className="space-y-8">
      <div className="space-y-4">
        <SkeletonLine className="h-4 w-5/6" />
        <SkeletonLine className="h-4 w-4/5" />
        <SkeletonLine className="h-4 w-3/5" />
      </div>

      <div className="space-y-4">
        <SkeletonLine className="h-4 w-full" />
        <SkeletonLine className="h-4 w-[92%]" />
        <SkeletonLine className="h-4 w-[78%]" />
      </div>

      <div className="rounded-2xl border border-neutral-200 bg-neutral-50/70 p-5 dark:border-neutral-800 dark:bg-neutral-900/70">
        <div className="mb-4 flex items-center gap-2">
          <SkeletonLine className="h-3 w-16 rounded-md" />
          <SkeletonLine className="h-3 w-10 rounded-md" />
        </div>
        <div className="space-y-3">
          <SkeletonLine className="h-3 w-full rounded-md" />
          <SkeletonLine className="h-3 w-[95%] rounded-md" />
          <SkeletonLine className="h-3 w-[88%] rounded-md" />
          <SkeletonLine className="h-3 w-[72%] rounded-md" />
          <SkeletonLine className="h-3 w-[90%] rounded-md" />
        </div>
      </div>

      <div className="space-y-4">
        <SkeletonLine className="h-4 w-2/3" />
        <SkeletonLine className="h-4 w-4/5" />
        <SkeletonLine className="h-4 w-3/5" />
      </div>
    </div>
  );
};
