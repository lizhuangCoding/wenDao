import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadArticleEditor = () => readFile(new URL('./ArticleEditor.tsx', import.meta.url), 'utf8');
const loadWritingStudio = () =>
  readFile(new URL('./components/MarkdownWritingStudio.tsx', import.meta.url), 'utf8');

test('ArticleEditor delegates Markdown editing to MarkdownWritingStudio', async () => {
  const source = await loadArticleEditor();

  assert.match(source, /MarkdownWritingStudio/);
  assert.match(source, /isWritingFocused/);
  assert.match(source, /setIsWritingFocused/);
  assert.match(source, /max-w-display/);
  assert.doesNotMatch(source, /const markdownToolbarActions/);
  assert.doesNotMatch(source, /applyMarkdownAction/);
});

test('MarkdownWritingStudio owns toolbar color controls and immersive toggle', async () => {
  const source = await loadWritingStudio();

  assert.match(source, /ColorPicker/);
  assert.match(source, /TEXT_COLOR_PRESETS/);
  assert.match(source, /applyMarkdownColor/);
  assert.match(source, /normalizeMarkdownColor/);
  assert.match(source, /aria-label="应用当前字体颜色"/);
  assert.match(source, /专注写作/);
  assert.match(source, /退出专注/);
  assert.match(source, /onImmersiveChange/);
});

test('MarkdownWritingStudio keeps immersive and color controls responsive', async () => {
  const source = await loadWritingStudio();

  assert.match(source, /max\(640px,calc\(100vh-220px\)\)/);
  assert.match(source, /max\(580px,calc\(100vh-292px\)\)/);
  assert.match(source, /flex-wrap/);
  assert.match(source, /max-w-full/);
  assert.match(source, /max-w-\[112px\]/);
});
