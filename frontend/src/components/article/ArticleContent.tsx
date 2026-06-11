import { Suspense, lazy } from 'react';
import { ArticleContentSkeleton } from './ArticleContentSkeleton';

interface ArticleContentProps {
  content: string;
}

const ArticleMarkdownRenderer = lazy(() =>
  import('./ArticleMarkdownRenderer').then((module) => ({ default: module.ArticleMarkdownRenderer }))
);

export const ArticleContent = ({ content }: ArticleContentProps) => {
  return (
    <Suspense fallback={<ArticleContentSkeleton />}>
      <ArticleMarkdownRenderer content={content} />
    </Suspense>
  );
};
