import { Suspense, lazy } from 'react';

interface ArticleContentProps {
  content: string;
}

const ArticleMarkdownRenderer = lazy(() =>
  import('./ArticleMarkdownRenderer').then((module) => ({ default: module.ArticleMarkdownRenderer }))
);

export const ArticleContent = ({ content }: ArticleContentProps) => {
  return (
    <Suspense fallback={<div className="min-h-[12rem] animate-pulse rounded-2xl bg-neutral-100 dark:bg-neutral-800/50" />}>
      <ArticleMarkdownRenderer content={content} />
    </Suspense>
  );
};
