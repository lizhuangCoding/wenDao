import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadIndexCss = async () => {
  return readFile(new URL('./index.css', import.meta.url), 'utf8');
};

const loadMarkdownCss = async () => {
  return readFile(new URL('./markdown.css', import.meta.url), 'utf8');
};

const loadSourceFile = async (relativePath) => {
  return readFile(new URL(`../${relativePath}`, import.meta.url), 'utf8');
};

const getRuleBody = (css, selector) => {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const match = css.match(new RegExp(`${escapedSelector}\\s*\\{([^}]*)\\}`, 'm'));
  return match?.[1] || '';
};

test('public article blockquotes keep body-sized text', async () => {
  const css = await loadMarkdownCss();
  const ruleBody = getRuleBody(css, '.article-reading-body blockquote');

  assert.match(ruleBody, /text-base/);
  assert.doesNotMatch(ruleBody, /text-2xl|text-xl/);
});

test('front detail and admin preview share article reading styles', async () => {
  const css = `${await loadIndexCss()}\n${await loadMarkdownCss()}`;
  const articleDetail = await loadSourceFile('pages/ArticleDetail.tsx');
  const markdownStudio = await loadSourceFile(
    'views/admin/articles/components/MarkdownWritingStudio.tsx'
  );
  const renderer = await loadSourceFile('components/article/ArticleMarkdownRenderer.tsx');

  assert.match(css, /\.article-reading-body\s*\{/);
  assert.match(css, /\.article-reading-body pre\s*\{/);
  assert.match(css, /\.article-reading-body ul\s*\{/);
  assert.match(css, /\.article-reading-body strong\s*\{/);
  assert.match(articleDetail, /className="article-reading-body"/);
  assert.match(markdownStudio, /article-reading-body admin-markdown-preview/);
  assert.doesNotMatch(articleDetail, /prose-refined/);
  assert.match(renderer, /const className = 'scroll-mt-24'/);
  assert.doesNotMatch(renderer, /text-3xl|text-2xl|mt-8 mb-4/);
});

test('dark mode uses a deeper cool background and readable secondary text', async () => {
  const indexCss = await loadIndexCss();
  const layout = await loadSourceFile('components/common/Layout.tsx');

  assert.match(indexCss, /bg-\[#050a10\]/);
  assert.match(indexCss, /text-neutral-200/);
  assert.match(layout, /dark:bg-\[#050a10\]/);
});

test('global button utilities include dark mode variants', async () => {
  const indexCss = await loadIndexCss();

  assert.match(indexCss, /btn-primary[\s\S]*dark:bg-neutral-100/);
  assert.match(indexCss, /btn-primary[\s\S]*dark:text-neutral-900/);
  assert.match(indexCss, /btn-secondary[\s\S]*dark:bg-neutral-900/);
  assert.match(indexCss, /btn-secondary[\s\S]*dark:text-neutral-100/);
  assert.match(indexCss, /btn-secondary[\s\S]*dark:border-neutral-700/);
});
