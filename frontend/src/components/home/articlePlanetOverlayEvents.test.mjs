import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const readOverlaySource = () => readFile(new URL('./ArticlePlanetOverlay.tsx', import.meta.url), 'utf8');

test('article planet overlay only captures pointer events on real controls', async () => {
  const source = await readOverlaySource();

  assert.equal(
    source.includes('pointer-events-auto max-w-3xl'),
    false,
    'the broad left copy column must not block clicks intended for the WebGL planet'
  );
  assert.match(source, /<form[\s\S]*?className="[^"]*pointer-events-auto/);
  assert.match(source, /data-testid="article-planet-category-filter"[\s\S]*?className="[^"]*pointer-events-none/);
  assert.match(source, /<button[\s\S]*?className=\{`[^`]*pointer-events-auto/);
  assert.match(source, /className="[^"]*pointer-events-auto[^"]*sm:hidden/);
});

test('desktop article summary card exposes a close button', async () => {
  const source = await readOverlaySource();

  assert.match(source, /isActiveArticleCardVisible/);
  assert.match(source, /onActiveArticleClose/);
  assert.match(source, /aria-label=\{t\('common\.close'\)\}/);
});

test('desktop article summary card exposes the active collection reading path', async () => {
  const source = await readOverlaySource();

  assert.match(source, /activeCollectionArticles/);
  assert.match(source, /星座路径/);
  assert.match(source, /activeArticle\.collection\.name/);
  assert.match(source, /activeCollectionArticles\.slice\(0, 5\)\.map/);
});
