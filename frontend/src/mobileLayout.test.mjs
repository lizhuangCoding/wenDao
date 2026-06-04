import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadSourceFile = (relativePath) => readFile(new URL(relativePath, import.meta.url), 'utf8');

test('public mobile pages use stable mobile spacing and do not create horizontal page overflow', async () => {
  const [layout, home, articleCard, hero] = await Promise.all([
    loadSourceFile('./components/common/Layout.tsx'),
    loadSourceFile('./pages/Home.tsx'),
    loadSourceFile('./components/article/ArticleCard.tsx'),
    loadSourceFile('./components/home/ArticlePlanetHero.tsx'),
  ]);

  assert.match(layout, /overflow-x-hidden/);
  assert.match(home, /py-16 sm:py-24/);
  assert.match(home, /gap-y-16 sm:gap-y-24/);
  assert.match(articleCard, /text-xl sm:text-2xl/);
  assert.match(articleCard, /flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between/);
  assert.match(hero, /min-h-\[100svh\]/);
  assert.match(hero, /ArticlePlanetSceneFallback/);
  assert.doesNotMatch(hero, /ErrorState message="文章星球渲染失败"/);
});

test('AI chat uses mobile viewport-safe sizing and compact mobile controls', async () => {
  const source = await loadSourceFile('./pages/AIChat.tsx');

  assert.match(source, /h-dvh px-0 py-0/);
  assert.match(source, /h-\[calc\(100dvh-80px\)\]/);
  assert.match(source, /px-3 py-3 sm:px-8/);
  assert.match(source, /rounded-2xl sm:rounded-\[32px\]/);
  assert.match(source, /<span className="hidden sm:inline">/);
  assert.match(source, /if \(messages\.length === 0\) return;/);
});

test('admin surfaces keep mobile controls stacked and tables scroll instead of squeezing', async () => {
  const [adminLayout, pageHeader, dataTable] = await Promise.all([
    loadSourceFile('./components/admin/AdminLayout.tsx'),
    loadSourceFile('./components/common/PageHeader.tsx'),
    loadSourceFile('./components/common/DataTable.tsx'),
  ]);

  assert.match(adminLayout, /py-6 sm:py-10/);
  assert.match(adminLayout, /overflow-x-auto md:overflow-visible/);
  assert.match(adminLayout, /shrink-0 whitespace-nowrap/);
  assert.match(adminLayout, /scrollbar-hide/);
  assert.match(pageHeader, /w-full flex-col/);
  assert.match(pageHeader, /lg:w-auto/);
  assert.match(dataTable, /min-w-\[720px\]/);
  assert.match(dataTable, /px-4 py-3 sm:px-6 sm:py-4/);
});
