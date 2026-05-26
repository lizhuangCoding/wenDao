import type { ArticleOrbitItem } from '@/types';

const CATEGORY_COLORS = [
  '#10b981',
  '#38bdf8',
  '#f59e0b',
  '#ec4899',
  '#a78bfa',
  '#f43f5e',
  '#22c55e',
  '#06b6d4',
];

export interface ArticlePlanetNodeLayout {
  article: ArticleOrbitItem;
  color: string;
  emissiveIntensity: number;
  key: string;
  position: [number, number, number];
  radius: number;
  visual: ArticlePlanetNodeVisual;
  weight: number;
}

export interface ArticlePlanetNodeVisual {
  activeScale: number;
  coreRadius: number;
  glintRadius: number;
  haloOpacity: number;
  haloRadius: number;
  ringOpacity: number;
  ringRadius: number;
  shellOpacity: number;
  shellRadius: number;
}

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

export const getArticlePlanetColor = (categoryId?: number) => {
  if (!categoryId || Number.isNaN(categoryId)) {
    return CATEGORY_COLORS[0];
  }
  return CATEGORY_COLORS[Math.abs(categoryId) % CATEGORY_COLORS.length];
};

export const calculateArticlePlanetWeight = (article: ArticleOrbitItem) => {
  const topBonus = article.is_top ? 0.7 : 0;
  const viewBonus = clamp(Math.log10(article.view_count + 1) * 0.25, 0, 0.8);
  const commentBonus = clamp(Math.log10(article.comment_count + 1) * 0.2, 0, 0.5);
  return clamp(1 + topBonus + viewBonus + commentBonus, 1, 3);
};

export const buildArticlePlanetVisual = (weight: number): ArticlePlanetNodeVisual => {
  const influence = clamp((weight - 1) / 2, 0, 1);
  const coreRadius = 0.072 + influence * 0.04;

  return {
    activeScale: 1.42 + influence * 0.28,
    coreRadius,
    glintRadius: coreRadius * 0.34,
    haloOpacity: 0.26 + influence * 0.16,
    haloRadius: coreRadius * (4.2 + influence * 0.95),
    ringOpacity: 0.44 + influence * 0.26,
    ringRadius: coreRadius * (2.8 + influence * 0.68),
    shellOpacity: 0.42 + influence * 0.12,
    shellRadius: coreRadius * (2.05 + influence * 0.22),
  };
};

export const buildArticlePlanetLayout = (
  articles: ArticleOrbitItem[],
  sphereRadius = 2.55
): ArticlePlanetNodeLayout[] => {
  if (articles.length === 0) {
    return [];
  }

  const goldenAngle = Math.PI * (3 - Math.sqrt(5));
  const denominator = Math.max(1, articles.length - 1);

  return articles.map((article, index) => {
    const y = 1 - (index / denominator) * 2;
    const radial = Math.sqrt(Math.max(0, 1 - y * y));
    const theta = index * goldenAngle;
    const categoryOffset = ((article.category?.id ?? 0) % 7) * 0.018;
    const radius = sphereRadius + categoryOffset;
    const weight = calculateArticlePlanetWeight(article);
    const visual = buildArticlePlanetVisual(weight);

    return {
      article,
      color: getArticlePlanetColor(article.category?.id),
      emissiveIntensity: 0.55 + weight * 0.28,
      key: `${article.id}-${article.slug}`,
      position: [
        Math.cos(theta) * radial * radius,
        y * radius,
        Math.sin(theta) * radial * radius,
      ],
      radius: visual.coreRadius,
      visual,
      weight,
    };
  });
};
