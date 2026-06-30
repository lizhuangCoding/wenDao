import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const readSource = (relativePath) => readFile(new URL(relativePath, import.meta.url), 'utf8');

test('article planet hero delays 3D enhancement until idle time in the viewport', async () => {
  const heroSource = await readSource('./components/home/ArticlePlanetHero.tsx');

  assert.match(heroSource, /shouldRenderScene/);
  assert.match(heroSource, /IntersectionObserver/);
  assert.match(heroSource, /requestIdleCallback/);
  assert.match(heroSource, /prefers-reduced-motion: reduce/);
  assert.match(heroSource, /connection\?\.saveData === true/);
  assert.match(heroSource, /: !shouldRenderScene \?/);
});

test('dashboard custom date range controls load on demand instead of in the base route chunk', async () => {
  const [dashboardSource, pickerSource] = await Promise.all([
    readSource('./views/admin/Dashboard.tsx'),
    readSource('./views/admin/DashboardDateRangePicker.tsx'),
  ]);

  assert.match(dashboardSource, /const DashboardDateRangePicker = lazy/);
  assert.doesNotMatch(dashboardSource, /from 'tdesign-react'/);
  assert.doesNotMatch(dashboardSource, /tdesign-react\/es\/style\/index\.css/);
  assert.match(pickerSource, /from 'tdesign-react'/);
  assert.match(pickerSource, /tdesign-react\/es\/style\/index\.css/);
});

test('vite build separates chart vendor code and markdown viewer/admin preview chunks', async () => {
  const viteConfig = await readSource('../vite.config.ts');

  assert.match(viteConfig, /return 'chart-vendor'/);
  assert.match(viteConfig, /return 'markdown-viewer'/);
  assert.match(viteConfig, /return 'markdown-admin-preview'/);
  assert.match(viteConfig, /id\.includes\('\/recharts\/'\)/);
  assert.match(viteConfig, /id\.includes\('\/rehype-raw\/'\)/);
  assert.match(viteConfig, /id\.includes\('\/react-markdown\/'\)/);
});
