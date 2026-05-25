import { Component, lazy, Suspense, useMemo, useState } from 'react';
import type { FormEvent, ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';
import { ErrorState, Loading } from '@/components/common';
import type { ArticleOrbitItem, Category } from '@/types';
import { ArticlePlanetOverlay } from './ArticlePlanetOverlay';

const ArticlePlanetScene = lazy(() =>
  import('./ArticlePlanetScene').then((module) => ({ default: module.ArticlePlanetScene }))
);

interface SceneErrorBoundaryProps {
  children: ReactNode;
  resetKey: string;
}

interface SceneErrorBoundaryState {
  hasError: boolean;
}

class SceneErrorBoundary extends Component<SceneErrorBoundaryProps, SceneErrorBoundaryState> {
  state: SceneErrorBoundaryState = { hasError: false };

  static getDerivedStateFromError() {
    return { hasError: true };
  }

  componentDidUpdate(previousProps: SceneErrorBoundaryProps) {
    if (previousProps.resetKey !== this.props.resetKey && this.state.hasError) {
      this.setState({ hasError: false });
    }
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="absolute inset-0 flex items-center justify-center px-6">
          <div className="w-full max-w-md">
            <ErrorState message="文章星球渲染失败" />
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

interface ArticlePlanetHeroProps {
  articles: ArticleOrbitItem[];
  categories?: Category[];
  inputValue: string;
  isError: boolean;
  isLoading: boolean;
  selectedCategory?: number;
  slogan?: string;
  onCategoryChange: (categoryId?: number) => void;
  onSearch: (event: FormEvent) => void;
  onSearchInputChange: (value: string) => void;
}

export const ArticlePlanetHero = ({
  articles,
  categories,
  inputValue,
  isError,
  isLoading,
  selectedCategory,
  slogan,
  onCategoryChange,
  onSearch,
  onSearchInputChange,
}: ArticlePlanetHeroProps) => {
  const navigate = useNavigate();
  const [activeArticleId, setActiveArticleId] = useState<number>();
  const activeArticle = useMemo(
    () => articles.find((article) => article.id === activeArticleId) ?? articles[0],
    [activeArticleId, articles]
  );

  const openArticle = (article: ArticleOrbitItem) => {
    if (article.slug) {
      navigate(`/article/${article.slug}`);
    }
  };

  return (
    <section className="relative -mt-20 min-h-[calc(100vh-1rem)] overflow-hidden bg-neutral-950">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_68%_45%,rgba(16,185,129,0.28),transparent_30%),radial-gradient(circle_at_30%_85%,rgba(56,189,248,0.18),transparent_28%),linear-gradient(135deg,#020617_0%,#07111f_46%,#030712_100%)]" />
      {isLoading ? (
        <div className="absolute inset-0 flex items-center justify-center">
          <Loading />
        </div>
      ) : isError || articles.length === 0 ? (
        <div className="absolute inset-0 flex items-center justify-center px-6">
          <div className="w-full max-w-md">
            <ErrorState message={isError ? '文章星球加载失败' : '暂无可展示文章'} />
          </div>
        </div>
      ) : (
        <SceneErrorBoundary resetKey={`${articles.length}-${selectedCategory ?? 'all'}`}>
          <Suspense
            fallback={
              <div className="absolute inset-0 flex items-center justify-center">
                <Loading />
              </div>
            }
          >
            <div className="absolute inset-0 lg:left-[24%]">
              <ArticlePlanetScene
                activeArticleId={activeArticle?.id}
                articles={articles}
                onArticleFocus={(article) => setActiveArticleId(article.id)}
                onArticleOpen={openArticle}
              />
            </div>
          </Suspense>
        </SceneErrorBoundary>
      )}
      <ArticlePlanetOverlay
        activeArticle={activeArticle}
        categories={categories}
        inputValue={inputValue}
        selectedCategory={selectedCategory}
        slogan={slogan}
        onCategoryChange={onCategoryChange}
        onSearch={onSearch}
        onSearchInputChange={onSearchInputChange}
      />
      <div className="pointer-events-none absolute inset-x-0 bottom-0 h-24 bg-gradient-to-t from-white to-transparent dark:from-neutral-900" />
    </section>
  );
};
