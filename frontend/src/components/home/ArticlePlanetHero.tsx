import { Component, lazy, Suspense, useMemo, useState } from 'react';
import type { FormEvent, ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';
import { Loading } from '@/components/common';
import type { ArticleOrbitItem, Category } from '@/types';
import { ArticlePlanetOverlay } from './ArticlePlanetOverlay';

const ArticlePlanetScene = lazy(() =>
  import('./ArticlePlanetScene').then((module) => ({ default: module.ArticlePlanetScene }))
);

interface SceneErrorBoundaryProps {
  children: ReactNode;
  fallbackMessage: string;
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
      return <ArticlePlanetSceneFallback message={this.props.fallbackMessage} />;
    }

    return this.props.children;
  }
}

const ArticlePlanetSceneFallback = ({ message }: { message?: string }) => (
  <div className="pointer-events-none absolute inset-0 z-[1] overflow-hidden">
    <div className="absolute left-1/2 top-[58%] h-[22rem] w-[22rem] -translate-x-1/2 -translate-y-1/2 rounded-full border border-cyan-200/15 bg-[radial-gradient(circle_at_35%_30%,rgba(125,211,252,0.26),rgba(16,185,129,0.14)_32%,rgba(15,23,42,0.12)_62%,transparent_70%)] shadow-[0_0_90px_rgba(16,185,129,0.22)] sm:left-[68%] sm:top-1/2 sm:h-[30rem] sm:w-[30rem]" />
    <div className="absolute left-[14%] top-[34%] h-1.5 w-1.5 rounded-full bg-primary-200/80 shadow-[0_0_18px_rgba(110,231,183,0.9)]" />
    <div className="absolute right-[18%] top-[28%] h-1 w-1 rounded-full bg-sky-200/70 shadow-[0_0_16px_rgba(186,230,253,0.8)]" />
    <div className="absolute bottom-[18%] left-[28%] h-1 w-1 rounded-full bg-white/60 shadow-[0_0_14px_rgba(255,255,255,0.8)]" />
    <div className="absolute inset-x-8 top-[62%] h-px rotate-[-8deg] bg-gradient-to-r from-transparent via-primary-200/25 to-transparent sm:inset-x-auto sm:right-10 sm:top-1/2 sm:w-[34rem]" />
    {message && (
      <div className="absolute bottom-24 right-6 hidden max-w-xs rounded-2xl border border-white/10 bg-neutral-950/55 px-4 py-3 text-xs font-bold text-white/65 backdrop-blur-md md:block">
        {message}
      </div>
    )}
  </div>
);

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
  const [isActiveArticleCardVisible, setIsActiveArticleCardVisible] = useState(true);
  const activeArticle = useMemo(
    () => articles.find((article) => article.id === activeArticleId) ?? articles[0],
    [activeArticleId, articles]
  );

  const focusArticle = (article: ArticleOrbitItem) => {
    setActiveArticleId(article.id);
    setIsActiveArticleCardVisible(true);
  };

  const openArticle = (article: ArticleOrbitItem) => {
    if (article.slug) {
      navigate(`/article/${article.slug}`);
    }
  };

  return (
    <section className="relative -mt-20 min-h-[100svh] overflow-hidden bg-neutral-950 sm:min-h-[calc(100vh-1rem)]">
      <div className="absolute inset-0 z-0 bg-[radial-gradient(circle_at_68%_45%,rgba(16,185,129,0.28),transparent_30%),radial-gradient(circle_at_30%_85%,rgba(56,189,248,0.18),transparent_28%),linear-gradient(135deg,#020617_0%,#07111f_46%,#030712_100%)]" />
      {isLoading ? (
        <div className="absolute inset-0 flex items-center justify-center">
          <Loading />
        </div>
      ) : isError || articles.length === 0 ? (
        <ArticlePlanetSceneFallback message={isError ? '文章星球加载失败' : '暂无可展示文章'} />
      ) : (
        <SceneErrorBoundary
          fallbackMessage="文章星球渲染失败"
          resetKey={`${articles.length}-${selectedCategory ?? 'all'}`}
        >
          <Suspense
            fallback={
              <div className="absolute inset-0 flex items-center justify-center">
                <Loading />
              </div>
            }
          >
            <div className="absolute inset-0 z-[1]">
              <ArticlePlanetScene
                activeArticleId={activeArticle?.id}
                articles={articles}
                onArticleFocus={focusArticle}
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
        isActiveArticleCardVisible={isActiveArticleCardVisible}
        selectedCategory={selectedCategory}
        slogan={slogan}
        onActiveArticleClose={() => setIsActiveArticleCardVisible(false)}
        onCategoryChange={onCategoryChange}
        onSearch={onSearch}
        onSearchInputChange={onSearchInputChange}
      />
      <div className="pointer-events-none absolute inset-x-0 bottom-0 z-20 h-24 bg-gradient-to-t from-white to-transparent dark:from-neutral-900" />
    </section>
  );
};
