import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadTableOfContentsSource = async () => {
  return readFile(new URL('./TableOfContents.tsx', import.meta.url), 'utf8');
};

test('TableOfContents uses a glassmorphism panel treatment', async () => {
  const source = await loadTableOfContentsSource();

  assert.match(source, /backdrop-blur-xl/);
  assert.match(source, /bg-white\/70/);
  assert.match(source, /dark:bg-\[#07111a\]\/80/);
  assert.match(source, /border-neutral-200\/70/);
  assert.match(source, /shadow-soft/);
});
