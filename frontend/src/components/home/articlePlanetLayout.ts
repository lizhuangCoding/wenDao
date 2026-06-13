import type { ArticleOrbitItem } from '@/types';

const PREMIUM_PLANET_PALETTES = [
  {
    accentColor: '#8ee7ff',
    atmosphereColor: '#38bdf8',
    rimColor: '#bae6fd',
    shadowColor: '#0b2d4d',
    surfaceColor: '#1d9ed0',
  },
  {
    accentColor: '#c4b5fd',
    atmosphereColor: '#8b5cf6',
    rimColor: '#ddd6fe',
    shadowColor: '#211447',
    surfaceColor: '#7c3aed',
  },
  {
    accentColor: '#ffd6a5',
    atmosphereColor: '#fb923c',
    rimColor: '#ffedd5',
    shadowColor: '#48200c',
    surfaceColor: '#d97706',
  },
  {
    accentColor: '#f9a8d4',
    atmosphereColor: '#ec4899',
    rimColor: '#fbcfe8',
    shadowColor: '#4a102f',
    surfaceColor: '#be185d',
  },
  {
    accentColor: '#99f6e4',
    atmosphereColor: '#14b8a6',
    rimColor: '#ccfbf1',
    shadowColor: '#063f3a',
    surfaceColor: '#0f766e',
  },
  {
    accentColor: '#f0abfc',
    atmosphereColor: '#c084fc',
    rimColor: '#f5d0fe',
    shadowColor: '#3b0f4a',
    surfaceColor: '#9333ea',
  },
  {
    accentColor: '#86efac',
    atmosphereColor: '#22c55e',
    rimColor: '#dcfce7',
    shadowColor: '#0b3d1c',
    surfaceColor: '#16a34a',
  },
  {
    accentColor: '#a5f3fc',
    atmosphereColor: '#06b6d4',
    rimColor: '#cffafe',
    shadowColor: '#083344',
    surfaceColor: '#0891b2',
  },
];

export interface ArticlePlanetNodeLayout {
  article: ArticleOrbitItem;
  color: string;
  emissiveIntensity: number;
  index: number;
  key: string;
  position: [number, number, number];
  radius: number;
  visual: ArticlePlanetNodeVisual;
  weight: number;
}

export interface ArticlePlanetConnection {
  color: string;
  from: [number, number, number];
  isActive: boolean;
  key: string;
  opacity: number;
  strength: 'collection' | 'category';
  to: [number, number, number];
}

export interface ArticlePlanetNodeVisual {
  activeScale: number;
  accentColor: string;
  atmosphereColor: string;
  coreRadius: number;
  glintRadius: number;
  haloOpacity: number;
  haloRadius: number;
  rimColor: string;
  ringOpacity: number;
  ringRadius: number;
  shadowColor: string;
  shellOpacity: number;
  shellRadius: number;
  surfaceColor: string;
}

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

export const getArticlePlanetColor = (categoryId?: number) => {
  return getArticlePlanetPalette(categoryId).accentColor;
};

export const getArticlePlanetPalette = (categoryId?: number) => {
  if (!categoryId || Number.isNaN(categoryId)) {
    return PREMIUM_PLANET_PALETTES[0];
  }
  return PREMIUM_PLANET_PALETTES[Math.abs(categoryId) % PREMIUM_PLANET_PALETTES.length];
};

export const calculateArticlePlanetWeight = (article: ArticleOrbitItem) => {
  const topBonus = article.is_top ? 0.7 : 0;
  const viewBonus = clamp(Math.log10(article.view_count + 1) * 0.25, 0, 0.8);
  const commentBonus = clamp(Math.log10(article.comment_count + 1) * 0.2, 0, 0.5);
  return clamp(1 + topBonus + viewBonus + commentBonus, 1, 3);
};

export const buildArticlePlanetVisual = (weight: number, categoryId?: number): ArticlePlanetNodeVisual => {
  const influence = clamp((weight - 1) / 2, 0, 1);
  const palette = getArticlePlanetPalette(categoryId);
  const coreRadius = 0.115 + influence * 0.035;

  return {
    activeScale: 1.42 + influence * 0.28,
    accentColor: palette.accentColor,
    atmosphereColor: palette.atmosphereColor,
    coreRadius,
    glintRadius: coreRadius * 0.22,
    haloOpacity: 0.14 + influence * 0.04,
    haloRadius: coreRadius * (1.92 + influence * 0.24),
    rimColor: palette.rimColor,
    ringOpacity: 0.34 + influence * 0.18,
    ringRadius: coreRadius * (1.72 + influence * 0.26),
    shadowColor: palette.shadowColor,
    shellOpacity: 0.2 + influence * 0.06,
    shellRadius: coreRadius * (1.26 + influence * 0.12),
    surfaceColor: palette.surfaceColor,
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
    const visual = buildArticlePlanetVisual(weight, article.category?.id);

    return {
      article,
      color: getArticlePlanetColor(article.category?.id),
      emissiveIntensity: 0.55 + weight * 0.28,
      index,
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

const sortByCollectionOrder = (a: ArticlePlanetNodeLayout, b: ArticlePlanetNodeLayout) => {
  const positionDiff = (a.article.collection?.position ?? 0) - (b.article.collection?.position ?? 0);
  if (positionDiff !== 0) return positionDiff;
  return a.article.id - b.article.id;
};

const sortByPlanetOrder = (a: ArticlePlanetNodeLayout, b: ArticlePlanetNodeLayout) => a.index - b.index;

const groupNodes = (
  nodes: ArticlePlanetNodeLayout[],
  getKey: (node: ArticlePlanetNodeLayout) => string | undefined
) => {
  const groups = new Map<string, ArticlePlanetNodeLayout[]>();
  for (const node of nodes) {
    const key = getKey(node);
    if (!key) continue;
    const group = groups.get(key) ?? [];
    group.push(node);
    groups.set(key, group);
  }
  return groups;
};

const activeContext = (nodes: ArticlePlanetNodeLayout[], activeArticleId?: number) => {
  const activeNode = nodes.find((node) => node.article.id === activeArticleId);
  return {
    categoryId: activeNode?.article.category?.id,
    collectionId: activeNode?.article.collection?.id,
  };
};

export const buildArticlePlanetConnections = (
  nodes: ArticlePlanetNodeLayout[],
  activeArticleId?: number
): ArticlePlanetConnection[] => {
  const active = activeContext(nodes, activeArticleId);
  const connections: ArticlePlanetConnection[] = [];

  for (const [collectionId, group] of groupNodes(nodes, (node) =>
    node.article.collection ? String(node.article.collection.id) : undefined
  )) {
    if (group.length < 2) continue;
    const sorted = [...group].sort(sortByCollectionOrder);
    for (let index = 0; index < sorted.length - 1; index += 1) {
      const from = sorted[index];
      const to = sorted[index + 1];
      const isActive = active.collectionId === from.article.collection?.id;
      connections.push({
        color: isActive ? '#f8fafc' : '#7dd3fc',
        from: from.position,
        isActive,
        key: `collection-${collectionId}-${from.article.id}-${to.article.id}`,
        opacity: isActive ? 0.82 : 0.46,
        strength: 'collection',
        to: to.position,
      });
    }
  }

  for (const [categoryId, group] of groupNodes(nodes, (node) =>
    node.article.category ? String(node.article.category.id) : undefined
  )) {
    if (group.length < 2) continue;
    const sorted = [...group].sort(sortByPlanetOrder);
    const maxCategoryConnections = Math.min(sorted.length - 1, 6);
    for (let index = 0; index < maxCategoryConnections; index += 1) {
      const from = sorted[index];
      const to = sorted[index + 1];
      const isActive = active.categoryId === from.article.category?.id && !active.collectionId;
      connections.push({
        color: isActive ? '#a7f3d0' : getArticlePlanetColor(from.article.category?.id),
        from: from.position,
        isActive,
        key: `category-${categoryId}-${from.article.id}-${to.article.id}`,
        opacity: isActive ? 0.34 : 0.12,
        strength: 'category',
        to: to.position,
      });
    }
  }

  return connections;
};
