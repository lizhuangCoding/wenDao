import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadSource = (path) => readFile(new URL(path, import.meta.url), 'utf8');

test('article detail SEO metadata uses normalized absolute URLs', async () => {
  const source = await loadSource('../pages/ArticleDetail.tsx');

  assert.match(source, /toAbsoluteSeoUrl/);
  assert.match(source, /const canonicalUrl = slug \? toAbsoluteSeoUrl\(`\/article\/\$\{slug\}`\) : '';/);
  assert.match(source, /const ogImage = toAbsoluteSeoUrl\(article\.cover_image \|\| '\/favicon\.svg'\);/);
  assert.match(source, /image: ogImage/);
  assert.match(source, /url: canonicalUrl/);
  assert.doesNotMatch(source, /window\.location\.origin/);
});

test('frontend SEO helper supports a configured public site URL', async () => {
  const source = await loadSource('./seo.ts');
  const viteEnv = await loadSource('../vite-env.d.ts');
  const envExample = await readFile(new URL('../../.env.example', import.meta.url), 'utf8');

  assert.match(source, /VITE_SITE_URL/);
  assert.match(source, /toAbsoluteSeoUrl/);
  assert.match(viteEnv, /VITE_SITE_URL\?: string/);
  assert.match(envExample, /VITE_SITE_URL=/);
});
