import type { FormEvent } from 'react';
import { ArrowRight, Search, Sparkles } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import type { ArticleOrbitItem, Category } from '@/types';

interface ArticlePlanetOverlayProps {
  activeArticle?: ArticleOrbitItem;
  categories?: Category[];
  inputValue: string;
  selectedCategory?: number;
  slogan?: string;
  onCategoryChange: (categoryId?: number) => void;
  onSearch: (event: FormEvent) => void;
  onSearchInputChange: (value: string) => void;
}

export const ArticlePlanetOverlay = ({
  activeArticle,
  categories,
  inputValue,
  selectedCategory,
  slogan,
  onCategoryChange,
  onSearch,
  onSearchInputChange,
}: ArticlePlanetOverlayProps) => {
  const { t } = useTranslation();

  return (
    <div className="pointer-events-none absolute inset-0 z-10 flex flex-col justify-end overflow-y-auto px-5 pb-6 pt-24 sm:px-10 sm:pb-8 lg:px-12 lg:pb-14">
      <div className="max-w-display mx-auto flex w-full flex-col gap-5 sm:gap-7 lg:flex-row lg:items-end lg:justify-between lg:gap-8">
        <div className="pointer-events-auto max-w-3xl">
          <div className="mb-3 inline-flex items-center gap-3 text-primary-300 sm:mb-5">
            <Sparkles className="h-4 w-4" />
            <span className="text-xs font-black uppercase tracking-[0.28em]">{t('home.heroSub')}</span>
          </div>
          <h1 className="max-w-4xl text-4xl font-black leading-[1.05] text-white drop-shadow-2xl sm:text-5xl lg:text-7xl">
            {slogan || '我不在执着于得到，而是享受走到'}
          </h1>
          <form onSubmit={onSearch} className="relative mt-5 max-w-xl sm:mt-8">
            <input
              type="text"
              placeholder={t('home.searchPlaceholder')}
              className="w-full border-b-2 border-white/25 bg-transparent py-3 pl-0 pr-12 text-sm font-bold tracking-widest text-white outline-none transition-colors placeholder:text-white/45 focus:border-primary-300"
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
          <div className="mt-5 flex gap-3 overflow-x-auto pb-1 scrollbar-hide sm:mt-7">
            <button
              type="button"
              onClick={() => onCategoryChange(undefined)}
              aria-pressed={selectedCategory === undefined}
              className={`shrink-0 border px-4 py-2 text-[10px] font-black uppercase tracking-[0.22em] transition-colors ${
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
                className={`shrink-0 border px-4 py-2 text-[10px] font-black uppercase tracking-[0.22em] transition-colors ${
                  selectedCategory === category.id
                    ? 'border-primary-300 bg-primary-300 text-neutral-950'
                    : 'border-white/20 bg-white/5 text-white/70 hover:border-white/40 hover:text-white'
                }`}
              >
                {category.name}
              </button>
            ))}
          </div>
          {activeArticle && (
            <Link
              to={`/article/${activeArticle.slug}`}
              className="mt-4 flex items-center justify-between gap-4 border border-white/15 bg-neutral-950/65 px-4 py-3 text-left shadow-2xl backdrop-blur-xl transition-colors hover:border-primary-300/70 sm:hidden"
            >
              <span className="min-w-0">
                <span className="block text-[10px] font-black uppercase tracking-[0.22em] text-primary-300">
                  {activeArticle.category?.name || 'Article'}
                </span>
                <span className="mt-1 block truncate text-sm font-black text-white">{activeArticle.title}</span>
              </span>
              <ArrowRight className="h-4 w-4 shrink-0 text-white/70" />
            </Link>
          )}
        </div>

        {activeArticle && (
          <Link
            to={`/article/${activeArticle.slug}`}
            className="pointer-events-auto hidden w-full max-w-md border border-white/15 bg-neutral-950/60 p-4 text-left shadow-2xl backdrop-blur-xl transition-colors hover:border-primary-300/70 sm:block sm:p-5 lg:mb-2"
          >
            <div className="mb-3 flex items-center justify-between gap-4">
              <span className="text-[10px] font-black uppercase tracking-[0.24em] text-primary-300">
                {activeArticle.category?.name || 'Article'}
              </span>
              <ArrowRight className="h-4 w-4 text-white/70" />
            </div>
            <h2 className="line-clamp-2 text-xl font-black leading-tight text-white sm:text-2xl">
              {activeArticle.title}
            </h2>
            <p className="mt-3 line-clamp-3 text-sm font-medium leading-relaxed text-white/62">
              {activeArticle.summary || '暂无摘要'}
            </p>
            <div className="mt-4 flex gap-5 text-[10px] font-bold uppercase tracking-widest text-white/45 sm:mt-5">
              <span>{activeArticle.view_count} views</span>
              <span>{activeArticle.comment_count} comments</span>
            </div>
          </Link>
        )}
      </div>
    </div>
  );
};
