import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const readHomeSource = () => readFile(new URL('../../pages/Home.tsx', import.meta.url), 'utf8');
const readHeroSource = () => readFile(new URL('./ArticlePlanetHero.tsx', import.meta.url), 'utf8');

test('home owns the knowledge planet time mode state', async () => {
  const source = await readHomeSource();

  assert.match(source, /planetTimeMode/);
  assert.match(source, /setPlanetTimeMode/);
  assert.match(source, /timeMode=\{planetTimeMode\}/);
  assert.match(source, /onTimeModeChange=\{setPlanetTimeMode\}/);
});

test('article planet hero filters visible articles by selected time mode', async () => {
  const source = await readHeroSource();

  assert.match(source, /filterArticlesByPlanetTime/);
  assert.match(source, /visibleArticles/);
  assert.match(source, /articles=\{visibleArticles\}/);
  assert.match(source, /activeCollectionArticles/);
  assert.match(source, /totalArticleCount=\{articles\.length\}/);
});
