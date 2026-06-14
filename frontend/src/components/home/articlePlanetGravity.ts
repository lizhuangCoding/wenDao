import type { ArticleOrbitItem } from '@/types';
import type { ArticlePlanetNodeLayout } from './articlePlanetLayout';

export type ArticlePlanetGravityRole = 'source' | 'related' | 'dimmed';

export interface ArticlePlanetGravityRecommendation {
  article: ArticleOrbitItem;
  score: number;
}

const GRAVITY_RECOMMENDATION_LIMIT = 4;
const GRAVITY_PULL_MIN = 0.1;
const GRAVITY_PULL_MAX = 0.18;

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

const addGravityScore = (scores: Map<number, number>, articleId: number, score: number) => {
  scores.set(articleId, Math.max(scores.get(articleId) ?? 0, clamp(score, 0, 1)));
};

export const getArticlePlanetGravityScores = (
  nodes: ArticlePlanetNodeLayout[],
  activeArticleId?: number
) => {
  if (!activeArticleId) return new Map<number, number>();

  const activeNode = nodes.find((node) => node.article.id === activeArticleId);
  if (!activeNode) return new Map<number, number>();

  const scores = new Map<number, number>();
  for (const neighbor of activeNode.article.semantic_neighbors ?? []) {
    if (neighbor.article_id !== activeArticleId) {
      addGravityScore(scores, neighbor.article_id, neighbor.score);
    }
  }

  for (const node of nodes) {
    if (node.article.id === activeArticleId) continue;
    const reverseNeighbor = node.article.semantic_neighbors?.find(
      (neighbor) => neighbor.article_id === activeArticleId
    );
    if (reverseNeighbor) {
      addGravityScore(scores, node.article.id, reverseNeighbor.score);
    }
  }

  return new Map(
    [...scores.entries()]
      .sort((left, right) => right[1] - left[1] || left[0] - right[0])
      .slice(0, GRAVITY_RECOMMENDATION_LIMIT)
  );
};

export const buildArticlePlanetGravityLayout = (
  nodes: ArticlePlanetNodeLayout[],
  activeArticleId?: number
): ArticlePlanetNodeLayout[] => {
  const activeNode = activeArticleId
    ? nodes.find((node) => node.article.id === activeArticleId)
    : undefined;
  const scores = getArticlePlanetGravityScores(nodes, activeArticleId);
  if (!activeNode || scores.size === 0) {
    return nodes.map((node) => ({ ...node, gravityRole: undefined, gravityScore: undefined }));
  }

  return nodes.map((node) => {
    if (node.article.id === activeArticleId) {
      return { ...node, gravityRole: 'source', gravityScore: 1 };
    }

    const score = scores.get(node.article.id);
    if (!score) {
      return { ...node, gravityRole: 'dimmed', gravityScore: 0 };
    }

    const pull = GRAVITY_PULL_MIN + (GRAVITY_PULL_MAX - GRAVITY_PULL_MIN) * score;
    return {
      ...node,
      gravityRole: 'related',
      gravityScore: score,
      position: [
        node.position[0] + (activeNode.position[0] - node.position[0]) * pull,
        node.position[1] + (activeNode.position[1] - node.position[1]) * pull,
        node.position[2] + (activeNode.position[2] - node.position[2]) * pull,
      ],
    };
  });
};

export const getArticlePlanetGravityRecommendations = (
  articles: ArticleOrbitItem[],
  activeArticle?: ArticleOrbitItem,
  limit = GRAVITY_RECOMMENDATION_LIMIT
): ArticlePlanetGravityRecommendation[] => {
  if (!activeArticle) return [];

  const scores = new Map<number, number>();
  for (const neighbor of activeArticle.semantic_neighbors ?? []) {
    if (neighbor.article_id !== activeArticle.id) {
      addGravityScore(scores, neighbor.article_id, neighbor.score);
    }
  }

  for (const article of articles) {
    if (article.id === activeArticle.id) continue;
    const reverseNeighbor = article.semantic_neighbors?.find(
      (neighbor) => neighbor.article_id === activeArticle.id
    );
    if (reverseNeighbor) {
      addGravityScore(scores, article.id, reverseNeighbor.score);
    }
  }

  return articles
    .filter((article) => article.id !== activeArticle.id && scores.has(article.id))
    .map((article) => ({
      article,
      score: scores.get(article.id) ?? 0,
    }))
    .sort((left, right) => right.score - left.score || left.article.id - right.article.id)
    .slice(0, limit);
};
