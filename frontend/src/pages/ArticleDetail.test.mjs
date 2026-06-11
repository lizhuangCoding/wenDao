import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadSourceFile = async (relativePath) => {
  return readFile(new URL(`../${relativePath}`, import.meta.url), 'utf8');
};

test('article detail renders a loading skeleton instead of a full-screen spinner', async () => {
  const source = await loadSourceFile('pages/ArticleDetail.tsx');

  assert.match(source, /ArticleDetailSkeleton/);
  assert.match(source, /estimateReadingTime/);
  assert.doesNotMatch(source, /<Loading\s*\/>/);
});

test('article detail displays estimated reading time in the header meta row', async () => {
  const source = await loadSourceFile('pages/ArticleDetail.tsx');

  assert.match(source, /article\.readingTime/);
  assert.match(source, /readingTime/);
});

test('article detail keeps table of contents in a sticky sidebar outside motion content', async () => {
  const source = await loadSourceFile('pages/ArticleDetail.tsx');

  assert.match(source, /lg:fixed/);
  assert.match(source, /lg:top-32/);
  assert.match(source, /aria-hidden="true"/);
  assert.match(source, /<TableOfContents headings=\{headings\} \/>/);
  assert.doesNotMatch(source, /motion\.article[\s\S]*<TableOfContents/);
});
