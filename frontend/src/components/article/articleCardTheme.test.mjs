import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadArticleCardSource = async () => {
  return readFile(new URL('./ArticleCard.tsx', import.meta.url), 'utf8');
};

test('ArticleCard keeps metadata and placeholders readable in light and dark mode', async () => {
  const source = await loadArticleCardSource();

  assert.match(source, /dark:from-neutral-800/);
  assert.match(source, /dark:to-neutral-900/);
  assert.match(source, /dark:border-neutral-700/);
  assert.match(source, /dark:text-neutral-500/);
  assert.match(source, /dark:bg-neutral-700/);
});

