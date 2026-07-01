import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadSourceFile = async (relativePath) => {
  return readFile(new URL(`../${relativePath}`, import.meta.url), 'utf8');
};

test('article renderer delegates pre blocks to the collapsible code block component', async () => {
  const source = await loadSourceFile('article/ArticleMarkdownRenderer.tsx');

  assert.match(source, /CollapsibleCodeBlock/);
  assert.match(source, /pre:\s*CollapsibleCodeBlock/);
});

test('article renderer keeps raw HTML disabled for public article content', async () => {
  const source = await loadSourceFile('article/ArticleMarkdownRenderer.tsx');

  assert.doesNotMatch(source, /rehypeRaw/);
  assert.doesNotMatch(source, /allowDangerousHtml/);
  assert.match(source, /remarkPlugins=\{\[remarkGfm\]\}/);
  assert.match(source, /rehypePlugins=\{\[rehypeHighlight\]\}/);
});

test('collapsible code blocks default long snippets to collapsed state', async () => {
  const source = await loadSourceFile('article/CollapsibleCodeBlock.tsx');

  assert.match(source, /MAX_COLLAPSED_CODE_LINES/);
  assert.match(source, /shouldCollapse/);
  assert.match(source, /const \[isExpanded, setIsExpanded\]/);
  assert.match(source, /!shouldCollapse/);
});

test('collapsible code blocks expose an explicit expand and collapse affordance', async () => {
  const source = await loadSourceFile('article/CollapsibleCodeBlock.tsx');

  assert.match(source, /codeBlock\.expand/);
  assert.match(source, /codeBlock\.collapse/);
  assert.match(source, /aria-expanded/);
  assert.match(source, /ChevronDown|ChevronUp/);
  assert.match(source, /absolute inset-x-0 bottom-2/);
  assert.doesNotMatch(source, /justify-center border-t/);
  assert.doesNotMatch(source, /absolute bottom-4 left-1\/2/);
});

test('collapsible code blocks render visible line numbers', async () => {
  const source = await loadSourceFile('article/CollapsibleCodeBlock.tsx');

  assert.match(source, /lineNumbers/);
  assert.match(source, /grid-cols-\[2\.25rem_minmax\(0,1fr\)\]/);
  assert.match(source, /text-\[11px\]/);
  assert.match(source, /!px-2 !py-2/);
  assert.match(source, /!leading-5/);
  assert.doesNotMatch(source, /!p-5/);
  assert.doesNotMatch(source, /leading-7/);
});

test('collapsible code blocks expose a standard copy-to-clipboard control', async () => {
  const source = await loadSourceFile('article/CollapsibleCodeBlock.tsx');
  const translations = await loadSourceFile('../i18n/resources/article.ts');

  assert.match(source, /navigator\.clipboard\.writeText\(codeText\)/);
  assert.match(source, /codeBlock\.copy/);
  assert.match(source, /codeBlock\.copied/);
  assert.match(source, /aria-label=\{copyLabel\}/);
  assert.match(source, /Copy/);
  assert.match(source, /Check/);
  assert.match(source, /top-2 right-2/);
  assert.doesNotMatch(source, /!pr-\d+/);
  assert.match(translations, /"copy": "Copy code"/);
  assert.match(translations, /"copied": "Copied"/);
  assert.match(translations, /"copy": "复制代码"/);
  assert.match(translations, /"copied": "已复制"/);
});
