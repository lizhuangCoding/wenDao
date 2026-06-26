import type { FormEvent } from 'react';
import { ArrowRight, Search, Sparkles, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import type { ArticleOrbitItem, Category, Tag } from '@/types';
import type { ArticlePlanetTimeMode } from './articlePlanetTime';
import type { ArticlePlanetGravityRecommendation } from './articlePlanetGravity';

interface ArticlePlanetOverlayProps {
  activeArticle?: ArticleOrbitItem;
  activeCollectionArticles?: ArticleOrbitItem[];
  activeGravityRecommendations?: ArticlePlanetGravityRecommendation[];
  categories?: Category[];
  inputValue: string;
  isActiveArticleCardVisible: boolean;
  planetYears: number[];
  selectedCategory?: number;
  selectedTag?: number;
  slogan?: string;
  tags?: Tag[];
  timeMode: ArticlePlanetTimeMode;
  totalArticleCount: number;
  visibleArticleCount: number;
  onActiveArticleClose: () => void;
  onCategoryChange: (categoryId?: number) => void;
  onTagChange: (tagId?: number) => void;
  onSearch: (event: FormEvent) => void;
  onSearchInputChange: (value: string) => void;
  onTimeModeChange: (mode: ArticlePlanetTimeMode) => void;
}

export const ArticlePlanetOverlay = ({
  activeArticle,
  activeCollectionArticles = [],
  activeGravityRecommendations = [],
  categories,
  inputValue,
  isActiveArticleCardVisible,
  planetYears,
  selectedCategory,
  selectedTag,
  slogan,
  tags,
  timeMode,
  totalArticleCount,
  visibleArticleCount,
  onActiveArticleClose,
  onCategoryChange,
  onTagChange,
  onSearch,
  onSearchInputChange,
  onTimeModeChange,
}: ArticlePlanetOverlayProps) => {
  const { t } = useTranslation();
  const timeLabel =
    timeMode === 'all'
      ? t('articlePlanet.allTime')
      : t('articlePlanet.beforeYear', { year: timeMode });

  return (
    <div className="pointer-events-none absolute inset-0 z-10 flex flex-col justify-between overflow-y-auto overflow-x-hidden px-5 pb-8 pt-28 sm:justify-end sm:px-10 sm:pb-8 sm:pt-24 lg:px-12 lg:pb-14">
      <div className="max-w-display mx-auto flex w-full min-w-0 flex-col gap-5 sm:gap-7 lg:flex-row lg:items-end lg:justify-between lg:gap-8">
        <div className="pointer-events-none min-w-0 max-w-3xl w-full">
          <div className="mb-3 inline-flex items-center gap-3 text-primary-300 sm:mb-5">
            <Sparkles className="h-4 w-4" />
            <span className="text-xs font-black uppercase tracking-[0.28em]">{t('home.heroSub')}</span>
          </div>
          <h1 className="w-full max-w-[20rem] break-all text-[clamp(1.8rem,8vw,2.25rem)] font-black leading-tight text-white drop-shadow-2xl [overflow-wrap:anywhere] sm:max-w-4xl sm:break-words sm:text-5xl sm:leading-[1.05] lg:text-7xl">
            {slogan || t('articlePlanet.sloganFallback')}
          </h1>
          <form onSubmit={onSearch} className="pointer-events-auto relative mt-5 max-w-xl sm:mt-8">
            <input
              type="text"
              placeholder={t('home.searchPlaceholder')}
              className="w-full border-b-2 border-white/25 bg-transparent py-3 pl-0 pr-12 text-sm font-bold tracking-widest text-white outline-none transition-colors placeholder:text-white/50 focus:border-primary-300"
              value={inputValue}
              onChange={(event) => onSearchInputChange(event.target.value)}
            />
            <button
              type="submit"
              className="absolute right-0 top-1/2 inline-flex h-10 w-10 -translate-y-1/2 items-center justify-center text-white/70 transition-colors hover:text-primary-200"
              aria-label={t('home.searchPlaceholder')}
            >
              <Search className="h-5 w-5" />
            </button>
          </form>
          <div
            data-testid="article-planet-category-filter"
            className="pointer-events-auto mt-5 flex max-w-full gap-3 overflow-x-auto pb-1 scrollbar-hide sm:mt-7"
          >
            <button
              type="button"
              onClick={() => onCategoryChange(undefined)}
              aria-pressed={selectedCategory === undefined}
              className={`pointer-events-auto shrink-0 border px-4 py-2 text-[10px] font-black uppercase tracking-[0.22em] transition-colors ${
                selectedCategory === undefined
                  ? 'border-primary-300 bg-primary-300 text-neutral-950'
                  : 'border-white/20 bg-white/5 text-white/70 hover:border-white/40 hover:text-white'
              }`}
            >
              {t('home.allArticles')}
            </button>
            {categories?.map((category) => (
              <button
                key={category.id}
                type="button"
                onClick={() => onCategoryChange(category.id)}
                aria-pressed={selectedCategory === category.id}
                className={`pointer-events-auto shrink-0 border px-4 py-2 text-[10px] font-black uppercase tracking-[0.22em] transition-colors ${
                  selectedCategory === category.id
                    ? 'border-primary-300 bg-primary-300 text-neutral-950'
                    : 'border-white/20 bg-white/5 text-white/70 hover:border-white/40 hover:text-white'
                }`}
              >
                {category.name}
              </button>
            ))}
          </div>
          {tags && tags.length > 0 && (
            <div className="pointer-events-auto mt-3 flex max-w-full gap-2 overflow-x-auto pb-1 scrollbar-hide">
              <button
                type="button"
                onClick={() => onTagChange(undefined)}
                aria-pressed={selectedTag === undefined}
                className={`pointer-events-auto shrink-0 border px-3 py-1.5 text-[10px] font-black uppercase tracking-[0.18em] transition-colors ${
                  selectedTag === undefined
                    ? 'border-emerald-200 bg-emerald-200 text-neutral-950'
                    : 'border-white/15 bg-white/[0.04] text-white/60 hover:border-white/35 hover:text-white'
                }`}
              >
                {t('articlePlanet.allTags')}
              </button>
              {tags.map((tag) => (
                <button
                  key={tag.id}
                  type="button"
                  onClick={() => onTagChange(tag.id)}
                  aria-pressed={selectedTag === tag.id}
                  className={`pointer-events-auto shrink-0 border px-3 py-1.5 text-[10px] font-black uppercase tracking-[0.18em] transition-colors ${
                    selectedTag === tag.id
                      ? 'border-emerald-200 bg-emerald-200 text-neutral-950'
                      : 'border-white/15 bg-white/[0.04] text-white/60 hover:border-white/35 hover:text-white'
                  }`}
                >
                  #{tag.name}
                </button>
              ))}
            </div>
          )}
          {planetYears.length > 0 && (
            <div className="pointer-events-auto mt-4 max-w-full sm:max-w-2xl">
              <div className="mb-2 flex items-center gap-3 text-[10px] font-black uppercase tracking-[0.22em] text-white/45">
                <span>{t('articlePlanet.timeMachine')}</span>
                <span className="h-px w-8 bg-white/15" />
                <span>{timeLabel}</span>
                <span>{t('articlePlanet.articleCountRatio', { visible: visibleArticleCount, total: totalArticleCount })}</span>
              </div>
              <div className="flex gap-2 overflow-x-auto pb-1 scrollbar-hide">
                <button
                  type="button"
                  onClick={() => onTimeModeChange('all')}
                  aria-pressed={timeMode === 'all'}
                  className={`pointer-events-auto shrink-0 border px-3 py-1.5 text-[10px] font-black uppercase tracking-[0.18em] transition-colors ${
                    timeMode === 'all'
                      ? 'border-sky-200 bg-sky-200 text-neutral-950'
                      : 'border-white/15 bg-white/5 text-white/60 hover:border-white/35 hover:text-white'
                  }`}
                >
                  {t('articlePlanet.all')}
                </button>
                {planetYears.map((year) => (
                  <button
                    key={year}
                    type="button"
                    onClick={() => onTimeModeChange(year)}
                    aria-pressed={timeMode === year}
                    className={`pointer-events-auto shrink-0 border px-3 py-1.5 text-[10px] font-black uppercase tracking-[0.18em] transition-colors ${
                      timeMode === year
                        ? 'border-sky-200 bg-sky-200 text-neutral-950'
                        : 'border-white/15 bg-white/5 text-white/60 hover:border-white/35 hover:text-white'
                    }`}
                  >
                    {year}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>

        {activeArticle && isActiveArticleCardVisible && (
          <div
            className="pointer-events-auto w-full min-w-0 max-w-md border border-white/15 bg-neutral-950/60 p-4 text-left shadow-2xl backdrop-blur-xl transition-colors hover:border-primary-300/70 sm:p-5 lg:mb-2"
          >
            <div className="mb-3 flex items-center justify-between gap-4">
              <span className="text-[10px] font-black uppercase tracking-[0.24em] text-primary-300">
                {activeArticle.category?.name || t('common.default')}
              </span>
              <div className="flex items-center gap-2">
                <Link
                  to={`/article/${activeArticle.slug}`}
                  className="inline-flex h-8 w-8 items-center justify-center border border-white/10 text-white/70 transition-colors hover:border-primary-300/60 hover:text-primary-200"
                  aria-label={`${t('common.view')} ${activeArticle.title}`}
                >
                  <ArrowRight className="h-4 w-4" />
                </Link>
                <button
                  type="button"
                  className="inline-flex h-8 w-8 items-center justify-center border border-white/10 text-white/55 transition-colors hover:border-white/35 hover:text-white"
                  aria-label={t('common.close')}
                  onClick={onActiveArticleClose}
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
            </div>
            <Link to={`/article/${activeArticle.slug}`} className="block">
              <h2 className="line-clamp-2 text-xl font-black leading-tight text-white sm:text-2xl">
                {activeArticle.title}
              </h2>
              <p className="mt-3 line-clamp-3 text-sm font-medium leading-relaxed text-white/70">
                {activeArticle.summary || t('article.summaryPlaceholder')}
              </p>
              <div className="mt-4 flex gap-5 text-[10px] font-bold uppercase tracking-widest text-white/50 sm:mt-5">
                <span>{activeArticle.view_count} {t('article.views')}</span>
                <span>{activeArticle.comment_count} {t('article.comments')}</span>
              </div>
            </Link>
            {activeGravityRecommendations.length > 0 && (
              <div className="mt-5 border-t border-white/10 pt-4">
                <div className="mb-3 flex items-center justify-between gap-3">
                  <div>
                    <p className="text-[10px] font-black uppercase tracking-[0.24em] text-cyan-200">
                      {t('articlePlanet.gravityRecommendations')}
                    </p>
                    <p className="mt-1 text-xs font-bold text-white/55">
                      {t('articlePlanet.gravityRecommendationHint')}
                    </p>
                  </div>
                  <span className="shrink-0 text-[10px] font-bold uppercase tracking-widest text-white/40">
                    {t('articlePlanet.articleCount', { count: activeGravityRecommendations.length })}
                  </span>
                </div>
                <div className="space-y-2">
                  {activeGravityRecommendations.map(({ article, score }) => (
                    <Link
                      key={article.id}
                      to={`/article/${article.slug}`}
                      className="flex items-center gap-3 border border-cyan-200/10 bg-cyan-200/[0.04] px-3 py-2 text-xs font-bold text-white/64 transition-colors hover:border-cyan-200/45 hover:text-white"
                    >
                      <span className="h-2 w-2 shrink-0 rounded-full bg-cyan-200 shadow-[0_0_12px_rgba(103,232,249,0.78)]" />
                      <span className="min-w-0 flex-1 truncate">{article.title}</span>
                      <span className="shrink-0 tabular-nums text-cyan-100/60">
                        {Math.round(score * 100)}%
                      </span>
                    </Link>
                  ))}
                </div>
              </div>
            )}
            {activeArticle.collection && activeCollectionArticles.length > 1 && (
              <div className="mt-5 border-t border-white/10 pt-4">
                <div className="mb-3 flex items-center justify-between gap-3">
                  <div>
                    <p className="text-[10px] font-black uppercase tracking-[0.24em] text-primary-300">
                      {t('articlePlanet.constellationPath')}
                    </p>
                    <p className="mt-1 truncate text-sm font-bold text-white/85">
                      {activeArticle.collection.name}
                    </p>
                  </div>
                  <span className="shrink-0 text-[10px] font-bold uppercase tracking-widest text-white/40">
                    {t('articlePlanet.articleCount', { count: activeCollectionArticles.length })}
                  </span>
                </div>
                <div className="space-y-2">
                  {activeCollectionArticles.slice(0, 5).map((article) => {
                    const isCurrent = article.id === activeArticle.id;
                    return (
                      <Link
                        key={article.id}
                        to={`/article/${article.slug}`}
                        className={`flex items-center gap-3 border px-3 py-2 text-xs font-bold transition-colors ${
                          isCurrent
                            ? 'border-primary-300/60 bg-primary-300/15 text-primary-100'
                            : 'border-white/10 bg-white/[0.03] text-white/58 hover:border-primary-300/40 hover:text-white'
                        }`}
                      >
                        <span className="w-5 shrink-0 text-right tabular-nums text-white/35">
                          {article.collection?.position ?? 0}
                        </span>
                        <span className="min-w-0 flex-1 truncate">{article.title}</span>
                      </Link>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
