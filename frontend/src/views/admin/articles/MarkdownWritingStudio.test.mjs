import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadArticleEditor = () => readFile(new URL('./ArticleEditor.tsx', import.meta.url), 'utf8');
const loadWritingStudio = () =>
  readFile(new URL('./components/MarkdownWritingStudio.tsx', import.meta.url), 'utf8');
const loadArticlePreview = () => readFile(new URL('./ArticlePreview.tsx', import.meta.url), 'utf8');

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
  assert.match(source, /articleEditor\.textColorApply/);
  assert.match(source, /articleEditor\.focusEnter/);
  assert.match(source, /articleEditor\.focusExit/);
  assert.match(source, /onImmersiveChange/);
});

test('MarkdownWritingStudio owns the collapsible AI toolbar panel', async () => {
  const source = await loadWritingStudio();

  assert.match(source, /Sparkles/);
  assert.match(source, /ChevronDown/);
  assert.match(source, /isAIPanelOpen/);
  assert.match(source, /articleEditor\.aiAssistant/);
  assert.match(source, /articleEditor\.summaryGenerate/);
  assert.match(source, /articleEditor\.aiPolish/);
  assert.match(source, /articleEditor\.aiExpand/);
  assert.match(source, /articleEditor\.aiShorten/);
  assert.match(source, /articleEditor\.aiSEOTitle/);
  assert.match(source, /articleEditor\.aiWritingResultHint/);
  assert.match(source, /onGenerateSummary/);
  assert.match(source, /onApplySummary/);
  assert.match(source, /onGenerateWritingAction/);
  assert.match(source, /onApplyWritingResult/);
});

test('MarkdownWritingStudio exposes richer block formatting actions', async () => {
  const source = await loadWritingStudio();

  assert.match(source, /heading-3/);
  assert.match(source, /heading-4/);
  assert.match(source, /Heading3/);
  assert.match(source, /Heading4/);
  assert.match(source, /unordered-list-indented/);
  assert.doesNotMatch(source, /task-list/);
  assert.match(source, /toolbarNestedUnorderedList/);
});

test('MarkdownWritingStudio renders immersive mode as a fullscreen writing surface', async () => {
  const source = await loadWritingStudio();

  assert.match(source, /fixed inset-0 z-50/);
  assert.match(source, /overflow-hidden bg-neutral-50/);
  assert.match(source, /min-h-0 flex-1/);
  assert.match(source, /h-full/);
});

test('MarkdownWritingStudio keeps color controls responsive', async () => {
  const source = await loadWritingStudio();

  assert.match(source, /flex-wrap/);
  assert.match(source, /max-w-full/);
  assert.match(source, /max-w-\[112px\]/);
});

test('MarkdownWritingStudio synchronizes editor and preview scrolling', async () => {
  const source = await loadWritingStudio();

  assert.match(source, /previewScrollRef/);
  assert.match(source, /syncMarkdownScroll/);
  assert.match(source, /syncPreviewScrollToEditorAnchor/);
  assert.match(source, /syncEditorScrollToPreviewAnchor/);
  assert.match(source, /getMarkdownScrollAnchorLines/);
  assert.match(source, /getEditorMarkdownAnchors/);
  assert.match(source, /data-editor-md-line/);
  assert.match(source, /editorScrollMirrorRef/);
  assert.match(source, /getScrollMap/);
  assert.match(source, /interpolateScrollMap/);
  assert.match(source, /getBoundingClientRect/);
  assert.match(source, /onScroll=\{handleEditorScroll\}/);
  assert.match(source, /onScroll=\{handlePreviewScroll\}/);
  assert.match(source, /getSynchronizedScrollTop/);
});

test('ArticlePreview annotates rendered blocks with markdown source lines', async () => {
  const source = await loadArticlePreview();

  assert.match(source, /data-md-line/);
  assert.match(source, /node\?\.position\?\.start\?\.line/);
  assert.match(source, /CollapsibleCodeBlock/);
});
