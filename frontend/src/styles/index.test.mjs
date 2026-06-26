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

test('article reading styles visually distinguish nested unordered lists', async () => {
  const css = await loadMarkdownCss();
  const nestedListBody = getRuleBody(css, '.article-reading-body ul ul');
  const nestedBulletBody = getRuleBody(css, '.article-reading-body ul ul > li::before');
  const deepNestedBulletBody = getRuleBody(css, '.article-reading-body ul ul ul > li::before');

  assert.match(nestedListBody, /pl-4/);
  assert.match(nestedBulletBody, /border-2/);
  assert.match(nestedBulletBody, /bg-transparent/);
  assert.match(deepNestedBulletBody, /w-3/);
});

test('article reading styles improve tables images and mobile rhythm', async () => {
  const css = await loadMarkdownCss();
  const tableBody = getRuleBody(css, '.article-reading-body table');
  const imageBody = getRuleBody(css, '.article-reading-body img');

  assert.match(tableBody, /min-width:\s*640px/);
  assert.match(css, /\.article-reading-body \.markdown-body:has\(table\)/);
  assert.match(css, /overflow-x:\s*auto/);
  assert.match(imageBody, /mx-auto/);
  assert.match(imageBody, /max-height:\s*72vh/);
  assert.match(css, /@media \(max-width:\s*640px\)/);
  assert.match(css, /\.article-reading-body h1[\s\S]*text-2xl/);
  assert.match(css, /\.article-reading-body p[\s\S]*my-4/);
});

test('dark mode uses a deeper cool background and readable secondary text', async () => {
  const indexCss = await loadIndexCss();
  const layout = await loadSourceFile('components/common/Layout.tsx');

  assert.match(indexCss, /bg-\[#050a10\]/);
  assert.match(indexCss, /text-neutral-200/);
  assert.match(layout, /dark:bg-\[#050a10\]/);
});

test('theme changes animate color surfaces while respecting reduced motion', async () => {
  const indexCss = await loadIndexCss();

  assert.match(indexCss, /--theme-transition-duration:\s*300ms/);
  assert.match(indexCss, /transition-property:\s*background-color,\s*color,\s*border-color,\s*fill,\s*stroke/);
  assert.match(indexCss, /transition-duration:\s*var\(--theme-transition-duration\)/);
  assert.match(indexCss, /prefers-reduced-motion:\s*reduce/);
  assert.match(indexCss, /--theme-transition-duration:\s*0ms/);
});

test('global button utilities include dark mode variants', async () => {
  const indexCss = await loadIndexCss();

  assert.match(indexCss, /btn-primary[\s\S]*dark:bg-primary-500/);
  assert.match(indexCss, /btn-primary[\s\S]*dark:hover:bg-primary-400/);
  assert.match(indexCss, /btn-secondary[\s\S]*dark:border-neutral-700/);
  assert.match(indexCss, /btn-secondary[\s\S]*dark:bg-neutral-900/);
  assert.match(indexCss, /btn-secondary[\s\S]*dark:text-neutral-200/);
  assert.match(indexCss, /btn-secondary[\s\S]*dark:hover:bg-neutral-800/);
});

test('app initializes persisted theme before route pages render', async () => {
  const app = await loadSourceFile('App.tsx');

  assert.match(app, /useThemeStore/);
  assert.match(app, /initTheme/);
});

test('theme store exposes dark mode to TDesign popup variables', async () => {
  const themeStore = await loadSourceFile('store/themeStore.ts');

  assert.match(themeStore, /setAttribute\('theme-mode', actualTheme\)/);
  assert.match(themeStore, /classList\.toggle\('dark', actualTheme === 'dark'\)/);
});
