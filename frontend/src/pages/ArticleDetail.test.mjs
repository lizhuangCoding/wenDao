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

test('article detail keeps the reading surface free of cursor atmosphere effects', async () => {
  const source = await loadSourceFile('pages/ArticleDetail.tsx');

  assert.doesNotMatch(source, /ParticleAtmosphere/);
  assert.doesNotMatch(source, /CursorCometTrail/);
  assert.match(source, /relative z-10 max-w-display/);
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

test('article detail does not render the sidebar publisher panel', async () => {
  const source = await loadSourceFile('pages/ArticleDetail.tsx');

  assert.doesNotMatch(source, /article\.sharedBy/);
  assert.doesNotMatch(source, /article\.contributor/);
  assert.doesNotMatch(source, /dicebear\.com\/7\.x\/avataaars/);
});

test('article detail resets window scroll when navigating between article slugs', async () => {
  const source = await loadSourceFile('pages/ArticleDetail.tsx');

  assert.match(source, /window\.scrollTo\(\{\s*top:\s*0,\s*left:\s*0,\s*behavior:\s*'auto'\s*\}\)/);
  assert.match(source, /\}, \[slug\]\)/);
});

test('article detail renders a scroll-based reading progress bar', async () => {
  const source = await loadSourceFile('pages/ArticleDetail.tsx');

  assert.match(source, /readingProgress/);
  assert.match(source, /window\.scrollY|document\.documentElement\.scrollTop/);
  assert.match(source, /document\.documentElement\.scrollHeight/);
  assert.match(source, /window\.innerHeight|document\.documentElement\.clientHeight/);
  assert.match(source, /addEventListener\('scroll'/);
  assert.match(source, /addEventListener\('resize'/);
  assert.match(source, /article-reading-progress/);
  assert.match(source, /article-print-hidden/);
});

test('article detail exposes export actions beside like and favorite controls', async () => {
  const source = await loadSourceFile('pages/ArticleDetail.tsx');
  const styles = await loadSourceFile('styles/index.css');

  assert.match(source, /downloadArticleMarkdown\(article\)/);
  assert.match(source, /window\.print\(\)/);
  assert.match(source, /article-export-actions/);
  assert.match(source, /article-interaction-actions[\s\S]*article-export-actions/);
  assert.match(styles, /@media print/);
  assert.match(styles, /\.article-print-hidden/);
  assert.match(styles, /\.article-interaction-actions/);
  assert.match(styles, /\.site-header/);
  assert.doesNotMatch(styles, /header,\s*\n\s*footer,/);
});
