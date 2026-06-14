import assert from 'node:assert/strict';
import { mkdir, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { build } from 'esbuild';

const tempDir = path.join(tmpdir(), 'wendao-article-planet-gravity-tests');
const bundlePath = path.join(tempDir, 'articlePlanetGravity.test-bundle.mjs');

const loadGravity = async () => {
  await mkdir(tempDir, { recursive: true });

  await build({
    entryPoints: [new URL('./articlePlanetGravity.ts', import.meta.url).pathname],
    bundle: true,
    format: 'esm',
    outfile: bundlePath,
    platform: 'node',
  });

  return import(`file://${bundlePath}?cache=${Date.now()}`);
};

const makeArticle = (overrides = {}) => ({
  id: 1,
  title: '文章',
  slug: 'article',
  summary: '摘要',
  view_count: 0,
  comment_count: 0,
  is_top: false,
  source_type: 'manual',
  category: { id: 1, name: 'AI', slug: 'ai' },
  created_at: '2026-05-24T12:00:00Z',
  ...overrides,
});

const makeNode = (article, position) => ({
  article,
  color: '#8ee7ff',
  emissiveIntensity: 1,
  index: article.id,
  key: `${article.id}-${article.slug}`,
  position,
  radius: 0.12,
  visual: {},
  weight: 1,
});

test.after(async () => {
  await rm(tempDir, { recursive: true, force: true });
});

test('getArticlePlanetGravityScores merges forward and reverse semantic neighbors', async () => {
  const { getArticlePlanetGravityScores } = await loadGravity();
  const nodes = [
    makeNode(makeArticle({ id: 1, slug: 'a', semantic_neighbors: [{ article_id: 2, score: 0.8 }] }), [0, 0, 0]),
    makeNode(makeArticle({ id: 2, slug: 'b' }), [1, 0, 0]),
    makeNode(makeArticle({ id: 3, slug: 'c', semantic_neighbors: [{ article_id: 1, score: 0.7 }] }), [0, 1, 0]),
  ];

  const scores = getArticlePlanetGravityScores(nodes, 1);

  assert.equal(scores.get(2), 0.8);
  assert.equal(scores.get(3), 0.7);
});

test('buildArticlePlanetGravityLayout pulls related nodes toward the active node and dims unrelated nodes', async () => {
  const { buildArticlePlanetGravityLayout } = await loadGravity();
  const nodes = [
    makeNode(makeArticle({ id: 1, slug: 'a', semantic_neighbors: [{ article_id: 2, score: 1 }] }), [0, 0, 0]),
    makeNode(makeArticle({ id: 2, slug: 'b' }), [10, 0, 0]),
    makeNode(makeArticle({ id: 3, slug: 'c' }), [0, 10, 0]),
  ];

  const gravityNodes = buildArticlePlanetGravityLayout(nodes, 1);

  assert.equal(gravityNodes[0].gravityRole, 'source');
  assert.equal(gravityNodes[1].gravityRole, 'related');
  assert.equal(gravityNodes[2].gravityRole, 'dimmed');
  assert.ok(gravityNodes[1].position[0] < nodes[1].position[0]);
  assert.ok(gravityNodes[1].position[0] > 8);
});

test('getArticlePlanetGravityRecommendations returns sorted semantic recommendations', async () => {
  const { getArticlePlanetGravityRecommendations } = await loadGravity();
  const active = makeArticle({
    id: 1,
    slug: 'a',
    semantic_neighbors: [
      { article_id: 3, score: 0.62 },
      { article_id: 2, score: 0.91 },
    ],
  });
  const articles = [
    active,
    makeArticle({ id: 2, slug: 'b', title: '高相关' }),
    makeArticle({ id: 3, slug: 'c', title: '中相关' }),
    makeArticle({ id: 4, slug: 'd', title: '反向相关', semantic_neighbors: [{ article_id: 1, score: 0.72 }] }),
  ];

  const recommendations = getArticlePlanetGravityRecommendations(articles, active);

  assert.deepEqual(
    recommendations.map((recommendation) => recommendation.article.id),
    [2, 4, 3]
  );
  assert.deepEqual(
    recommendations.map((recommendation) => recommendation.score),
    [0.91, 0.72, 0.62]
  );
});
