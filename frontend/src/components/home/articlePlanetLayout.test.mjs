import assert from 'node:assert/strict';
import { mkdir, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { build } from 'esbuild';

const tempDir = path.join(tmpdir(), 'wendao-article-planet-layout-tests');
const bundlePath = path.join(tempDir, 'articlePlanetLayout.test-bundle.mjs');

const loadLayout = async () => {
  await mkdir(tempDir, { recursive: true });

  await build({
    entryPoints: [new URL('./articlePlanetLayout.ts', import.meta.url).pathname],
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

test.after(async () => {
  await rm(tempDir, { recursive: true, force: true });
});

test('buildArticlePlanetLayout returns one stable node per article', async () => {
  const { buildArticlePlanetLayout } = await loadLayout();
  const articles = [
    makeArticle({ id: 1, slug: 'a' }),
    makeArticle({ id: 2, slug: 'b', category: { id: 2, name: 'Go', slug: 'go' } }),
    makeArticle({ id: 3, slug: 'c' }),
  ];

  const first = buildArticlePlanetLayout(articles);
  const second = buildArticlePlanetLayout(articles);

  assert.equal(first.length, 3);
  assert.deepEqual(second, first);
  for (const node of first) {
    const distance = Math.hypot(node.position[0], node.position[1], node.position[2]);
    assert.ok(distance > 2.35 && distance < 2.75, `expected node on sphere surface, got ${distance}`);
  }
});

test('calculateArticlePlanetWeight rewards top and active articles', async () => {
  const { calculateArticlePlanetWeight } = await loadLayout();

  const base = calculateArticlePlanetWeight(makeArticle());
  const active = calculateArticlePlanetWeight(makeArticle({ is_top: true, view_count: 1000, comment_count: 25 }));

  assert.ok(active > base);
  assert.ok(active <= 3);
});

test('getArticlePlanetColor maps the same category to the same color', async () => {
  const { getArticlePlanetColor } = await loadLayout();

  assert.equal(getArticlePlanetColor(4), getArticlePlanetColor(4));
  assert.notEqual(getArticlePlanetColor(4), getArticlePlanetColor(5));
});

test('buildArticlePlanetLayout handles empty article lists', async () => {
  const { buildArticlePlanetLayout } = await loadLayout();

  assert.deepEqual(buildArticlePlanetLayout([]), []);
});

test('buildArticlePlanetLayout assigns layered gem visual profile to every node', async () => {
  const { buildArticlePlanetLayout } = await loadLayout();

  const [baseNode, activeNode] = buildArticlePlanetLayout([
    makeArticle({ id: 1, slug: 'base' }),
    makeArticle({ id: 2, slug: 'active', is_top: true, view_count: 3000, comment_count: 42 }),
  ]);

  assert.ok(baseNode.visual.coreRadius > 0);
  assert.ok(baseNode.visual.shellRadius > baseNode.visual.coreRadius);
  assert.ok(baseNode.visual.haloRadius > baseNode.visual.shellRadius);
  assert.ok(baseNode.visual.ringRadius > baseNode.visual.shellRadius);
  assert.ok(baseNode.visual.glintRadius > 0);
  assert.ok(activeNode.visual.activeScale > baseNode.visual.activeScale);
  assert.ok(activeNode.visual.haloOpacity > baseNode.visual.haloOpacity);
});
