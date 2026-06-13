import type { ArticleOrbitItem } from '@/types';

export type ArticlePlanetTimeMode = 'all' | number;

export const getArticleOrbitPublishedDate = (article: ArticleOrbitItem) => {
  const value = article.published_at || article.created_at;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
};

export const getArticleOrbitYear = (article: ArticleOrbitItem) => {
  const date = getArticleOrbitPublishedDate(article);
  return date ? date.getFullYear() : undefined;
};

export const getArticlePlanetYears = (articles: ArticleOrbitItem[]) => {
  const years = new Set<number>();
  for (const article of articles) {
    const year = getArticleOrbitYear(article);
    if (year) {
      years.add(year);
    }
  }
  return Array.from(years).sort((a, b) => a - b);
};

export const filterArticlesByPlanetTime = (
  articles: ArticleOrbitItem[],
  timeMode: ArticlePlanetTimeMode
) => {
  if (timeMode === 'all') {
    return articles;
  }
  return articles.filter((article) => {
    const year = getArticleOrbitYear(article);
    return year !== undefined && year <= timeMode;
  });
};

export const getArticlePlanetTimeLabel = (timeMode: ArticlePlanetTimeMode) =>
  timeMode === 'all' ? '全部时间' : `${timeMode} 年以前`;
