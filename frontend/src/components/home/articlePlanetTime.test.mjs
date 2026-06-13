import assert from 'node:assert/strict';
import { mkdir, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { build } from 'esbuild';

const tempDir = path.join(tmpdir(), 'wendao-article-planet-time-tests');
const bundlePath = path.join(tempDir, 'articlePlanetTime.test-bundle.mjs');

const loadTime = async () => {
  await mkdir(tempDir, { recursive: true });

  await build({
    entryPoints: [new URL('./articlePlanetTime.ts', import.meta.url).pathname],
    bundle: true,
    format: 'esm',
    outfile: bundlePath,
    platform: 'node',
  });

  return import(`file://${bundlePath}?cache=${Date.now()}`);
};

const makeArticle = (id, publishedAt) => ({
  id,
  title: `文章 ${id}`,
  slug: `article-${id}`,
  summary: '摘要',
  view_count: 0,
  comment_count: 0,
  is_top: false,
  source_type: 'manual',
  created_at: publishedAt,
  published_at: publishedAt,
});

test.after(async () => {
  await rm(tempDir, { recursive: true, force: true });
});

test('getArticlePlanetYears returns ascending publication years', async () => {
  const { getArticlePlanetYears } = await loadTime();

  assert.deepEqual(
    getArticlePlanetYears([
      makeArticle(1, '2026-01-01T00:00:00Z'),
      makeArticle(2, '2024-01-01T00:00:00Z'),
      makeArticle(3, '2026-02-01T00:00:00Z'),
    ]),
    [2024, 2026]
  );
});

test('filterArticlesByPlanetTime keeps articles published up to the selected year', async () => {
  const { filterArticlesByPlanetTime } = await loadTime();
  const articles = [
    makeArticle(1, '2024-01-01T00:00:00Z'),
    makeArticle(2, '2025-01-01T00:00:00Z'),
    makeArticle(3, '2026-01-01T00:00:00Z'),
  ];

  assert.deepEqual(
    filterArticlesByPlanetTime(articles, 2025).map((article) => article.id),
    [1, 2]
  );
  assert.equal(filterArticlesByPlanetTime(articles, 'all').length, 3);
});
