import { Component, lazy, Suspense, useEffect, useMemo, useRef, useState } from 'react';
import type { FormEvent, ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Loading } from '@/components/common';
import type { ArticleOrbitItem, Category, Tag } from '@/types';
import { ArticlePlanetOverlay } from './ArticlePlanetOverlay';
import {
  filterArticlesByPlanetTime,
  getArticlePlanetYears,
  type ArticlePlanetTimeMode,
} from './articlePlanetTime';
import { getArticlePlanetGravityRecommendations } from './articlePlanetGravity';

const ArticlePlanetScene = lazy(() =>
  import('./ArticlePlanetScene').then((module) => ({ default: module.ArticlePlanetScene }))
);

type IdleSchedulerWindow = Window & typeof globalThis & {
  requestIdleCallback?: (callback: () => void, options?: { timeout: number }) => number;
  cancelIdleCallback?: (handle: number) => void;
};

type NetworkInformationLike = {
  saveData?: boolean;
  effectiveType?: string;
};

const SCENE_IDLE_TIMEOUT_MS = 1200;
const SCENE_FALLBACK_DELAY_MS = 180;
const SCENE_ROOT_MARGIN = '160px 0px';

const shouldPreferStaticHero = () => {
  const connection = (navigator as Navigator & { connection?: NetworkInformationLike }).connection;

  if (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) {
    return true;
  }

  return connection?.saveData === true || connection?.effectiveType === '2g';
};

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
  selectedTag?: number;
  slogan?: string;
  tags?: Tag[];
  timeMode: ArticlePlanetTimeMode;
  onCategoryChange: (categoryId?: number) => void;
  onTagChange: (tagId?: number) => void;
  onSearch: (event: FormEvent) => void;
  onSearchInputChange: (value: string) => void;
  onTimeModeChange: (mode: ArticlePlanetTimeMode) => void;
}

export const ArticlePlanetHero = ({
  articles,
  categories,
  inputValue,
  isError,
  isLoading,
  selectedCategory,
  selectedTag,
  slogan,
  tags,
  timeMode,
  onCategoryChange,
  onTagChange,
  onSearch,
  onSearchInputChange,
  onTimeModeChange,
}: ArticlePlanetHeroProps) => {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const heroRef = useRef<HTMLElement>(null);
  const [activeArticleId, setActiveArticleId] = useState<number>();
  const [isActiveArticleCardVisible, setIsActiveArticleCardVisible] = useState(true);
  const [shouldRenderScene, setShouldRenderScene] = useState(false);
  const planetYears = useMemo(() => getArticlePlanetYears(articles), [articles]);
  const visibleArticles = useMemo(
    () => filterArticlesByPlanetTime(articles, timeMode),
    [articles, timeMode]
  );
  const activeArticle = useMemo(
    () => visibleArticles.find((article) => article.id === activeArticleId) ?? visibleArticles[0],
    [activeArticleId, visibleArticles]
  );
  const activeCollectionArticles = useMemo(() => {
    if (!activeArticle?.collection) return [];
    return visibleArticles
      .filter((article) => article.collection?.id === activeArticle.collection?.id)
      .sort((a, b) => (a.collection?.position ?? 0) - (b.collection?.position ?? 0) || a.id - b.id);
  }, [activeArticle, visibleArticles]);
  const activeGravityRecommendations = useMemo(
    () => getArticlePlanetGravityRecommendations(visibleArticles, activeArticle),
    [activeArticle, visibleArticles]
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

  useEffect(() => {
    if (shouldRenderScene) return;
    if (typeof window === 'undefined') return;
    if (shouldPreferStaticHero()) return;

    const heroElement = heroRef.current;
    if (!heroElement) return;

    const idleWindow = window as IdleSchedulerWindow;
    let idleHandle: number | undefined;
    let timeoutHandle: number | undefined;

    const activateScene = () => {
      if (idleWindow.requestIdleCallback) {
        idleHandle = idleWindow.requestIdleCallback(
          () => setShouldRenderScene(true),
          { timeout: SCENE_IDLE_TIMEOUT_MS }
        );
        return;
      }

      timeoutHandle = window.setTimeout(() => setShouldRenderScene(true), SCENE_FALLBACK_DELAY_MS);
    };

    if (typeof IntersectionObserver === 'undefined') {
      activateScene();
      return () => {
        if (idleHandle !== undefined) idleWindow.cancelIdleCallback?.(idleHandle);
        if (timeoutHandle !== undefined) window.clearTimeout(timeoutHandle);
      };
    }

    const observer = new IntersectionObserver((entries) => {
      if (!entries.some((entry) => entry.isIntersecting)) return;
      observer.disconnect();
      activateScene();
    }, { rootMargin: SCENE_ROOT_MARGIN });

    observer.observe(heroElement);

    return () => {
      observer.disconnect();
      if (idleHandle !== undefined) idleWindow.cancelIdleCallback?.(idleHandle);
      if (timeoutHandle !== undefined) window.clearTimeout(timeoutHandle);
    };
  }, [shouldRenderScene]);

  return (
    <section
      ref={heroRef}
      className="relative -mt-20 min-h-[100svh] overflow-hidden bg-neutral-950 sm:min-h-[calc(100vh-1rem)]"
    >
      <div className="absolute inset-0 z-0 bg-[radial-gradient(circle_at_68%_45%,rgba(16,185,129,0.28),transparent_30%),radial-gradient(circle_at_30%_85%,rgba(56,189,248,0.18),transparent_28%),linear-gradient(135deg,#020617_0%,#07111f_46%,#030712_100%)]" />
      {isLoading ? (
        <div className="absolute inset-0 flex items-center justify-center">
          <Loading />
        </div>
      ) : isError || visibleArticles.length === 0 ? (
        <ArticlePlanetSceneFallback message={isError ? t('articlePlanet.loadFailed') : t('articlePlanet.noArticles')} />
      ) : !shouldRenderScene ? (
        <ArticlePlanetSceneFallback />
      ) : (
        <SceneErrorBoundary
          fallbackMessage={t('articlePlanet.renderFailed')}
          resetKey={`${visibleArticles.length}-${selectedCategory ?? 'all'}-${timeMode}`}
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
                activeArticleId={activeArticleId}
                articles={visibleArticles}
                onArticleFocus={focusArticle}
                onArticleOpen={openArticle}
              />
            </div>
          </Suspense>
        </SceneErrorBoundary>
      )}
      <ArticlePlanetOverlay
        activeArticle={activeArticle}
        activeCollectionArticles={activeCollectionArticles}
        activeGravityRecommendations={activeGravityRecommendations}
        categories={categories}
        inputValue={inputValue}
        isActiveArticleCardVisible={isActiveArticleCardVisible}
        planetYears={planetYears}
        selectedCategory={selectedCategory}
        selectedTag={selectedTag}
        slogan={slogan}
        tags={tags}
        timeMode={timeMode}
        visibleArticleCount={visibleArticles.length}
        totalArticleCount={articles.length}
        onActiveArticleClose={() => setIsActiveArticleCardVisible(false)}
        onCategoryChange={onCategoryChange}
        onTagChange={onTagChange}
        onSearch={onSearch}
        onSearchInputChange={onSearchInputChange}
        onTimeModeChange={onTimeModeChange}
      />
      <div className="pointer-events-none absolute inset-x-0 bottom-0 z-20 h-24 bg-gradient-to-t from-white to-transparent dark:from-neutral-900" />
    </section>
  );
};
