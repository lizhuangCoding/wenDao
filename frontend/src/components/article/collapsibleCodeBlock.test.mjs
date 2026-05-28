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

test('collapsible code blocks default long snippets to collapsed state', async () => {
  const source = await loadSourceFile('article/CollapsibleCodeBlock.tsx');

  assert.match(source, /MAX_COLLAPSED_CODE_LINES/);
  assert.match(source, /shouldCollapse/);
  assert.match(source, /const \[isExpanded, setIsExpanded\]/);
  assert.match(source, /!shouldCollapse/);
});

test('collapsible code blocks expose an explicit expand and collapse affordance', async () => {
  const source = await loadSourceFile('article/CollapsibleCodeBlock.tsx');

  assert.match(source, /展开完整代码/);
  assert.match(source, /收起代码/);
  assert.match(source, /aria-expanded/);
  assert.match(source, /ChevronDown|ChevronUp/);
});

