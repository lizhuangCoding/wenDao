# Markdown Writing Studio Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add componentized font color controls and an immersive writing mode to the admin article Markdown editor.

**Architecture:** Keep `ArticleEditor` responsible for article data, metadata fields, save behavior, local drafts, and uploads. Move the Markdown toolbar, editor panel, preview panel, color control, and view mode UI into `MarkdownWritingStudio`, with a pure utility function handling safe color-span insertion.

**Tech Stack:** React 18, TypeScript, Tailwind CSS, lucide-react, tdesign-react `ColorPicker`, ReactMarkdown preview, Node `node:test`, esbuild utility-test bundling.

---

## File Structure

**Create:**
- `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/components/MarkdownWritingStudio.tsx` - controlled Markdown writing studio component with toolbar, color control, preview, image button, stats, and immersive toggle.
- `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/MarkdownWritingStudio.test.mjs` - source-level checks for the editor component boundary and required UI controls.

**Modify:**
- `/Users/lizhuang/go/src/wenDao/frontend/src/utils/markdownEditor.ts` - add safe color normalization and color-span insertion helpers.
- `/Users/lizhuang/go/src/wenDao/frontend/src/utils/markdownEditor.test.mjs` - add color insertion and sanitization tests.
- `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/ArticleEditor.tsx` - remove inline Markdown studio UI, render the new component, and add page-level immersive layout state.
- `/Users/lizhuang/go/src/wenDao/frontend/src/styles/index.test.mjs` - update the admin preview source check after the preview markup moves into `MarkdownWritingStudio`.

---

### Task 1: Add Color Utility Tests

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/utils/markdownEditor.test.mjs`

- [ ] **Step 1: Add failing tests for color insertion**

Append these tests to `/Users/lizhuang/go/src/wenDao/frontend/src/utils/markdownEditor.test.mjs`:

```js
test('applyMarkdownColor wraps selected text in a safe color span', async () => {
  const { applyMarkdownColor } = await loadMarkdownEditor();

  const result = applyMarkdownColor(
    {
      text: 'hello world',
      selectionStart: 6,
      selectionEnd: 11,
    },
    '#0EA5E9'
  );

  assert.equal(result.text, 'hello <span style="color: #0ea5e9">world</span>');
  assert.deepEqual(result.selection, { start: 36, end: 41 });
  assert.deepEqual(result.edit, {
    start: 6,
    end: 11,
    replacement: '<span style="color: #0ea5e9">world</span>',
    selection: { start: 36, end: 41 },
  });
});

test('applyMarkdownColor inserts selected fallback text when selection is empty', async () => {
  const { applyMarkdownColor } = await loadMarkdownEditor();

  const result = applyMarkdownColor(
    {
      text: 'before ',
      selectionStart: 7,
      selectionEnd: 7,
    },
    '#ef4444'
  );

  assert.equal(result.text, 'before <span style="color: #ef4444">彩色文字</span>');
  assert.deepEqual(result.selection, { start: 37, end: 41 });
});

test('applyMarkdownColor falls back when color value is unsafe', async () => {
  const { applyMarkdownColor, DEFAULT_TEXT_COLOR } = await loadMarkdownEditor();

  const result = applyMarkdownColor(
    {
      text: 'danger',
      selectionStart: 0,
      selectionEnd: 6,
    },
    '#ef4444";background-image:url(https://example.com/x)'
  );

  assert.equal(result.text, `<span style="color: ${DEFAULT_TEXT_COLOR}">danger</span>`);
  assert.deepEqual(result.selection, { start: 30, end: 36 });
});

test('normalizeMarkdownColor expands short hex colors', async () => {
  const { normalizeMarkdownColor } = await loadMarkdownEditor();

  assert.equal(normalizeMarkdownColor('#0af'), '#00aaff');
});
```

- [ ] **Step 2: Run the focused utility test and verify it fails**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend && node --test src/utils/markdownEditor.test.mjs
```

Expected: FAIL with an export error for `applyMarkdownColor` or `normalizeMarkdownColor`.

---

### Task 2: Implement Safe Markdown Color Insertion

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/utils/markdownEditor.ts`
- Test: `/Users/lizhuang/go/src/wenDao/frontend/src/utils/markdownEditor.test.mjs`

- [ ] **Step 1: Add color helper exports and input type**

In `/Users/lizhuang/go/src/wenDao/frontend/src/utils/markdownEditor.ts`, make `ApplyMarkdownActionInput` exported and add the color helper constants below the result interfaces:

```ts
export interface ApplyMarkdownActionInput {
  text: string;
  selectionStart: number;
  selectionEnd: number;
  action: MarkdownAction;
}

export type ApplyMarkdownTextInput = Omit<ApplyMarkdownActionInput, 'action'>;

export const DEFAULT_TEXT_COLOR = '#ef4444';

const HEX_COLOR_PATTERN = /^#(?:[0-9a-f]{3}|[0-9a-f]{6}|[0-9a-f]{8})$/i;
```

- [ ] **Step 2: Add normalization and color insertion functions**

Add these functions above `export const applyMarkdownAction`:

```ts
export const normalizeMarkdownColor = (
  color: string | undefined,
  fallback = DEFAULT_TEXT_COLOR
): string => {
  const trimmed = (color || '').trim();

  if (!HEX_COLOR_PATTERN.test(trimmed)) {
    return fallback;
  }

  const lower = trimmed.toLowerCase();
  if (lower.length === 4) {
    return `#${lower[1]}${lower[1]}${lower[2]}${lower[2]}${lower[3]}${lower[3]}`;
  }

  return lower;
};

export const applyMarkdownColor = (
  input: ApplyMarkdownTextInput,
  color: string
): ApplyMarkdownActionResult => {
  const safeColor = normalizeMarkdownColor(color);
  const selectedText = input.text.slice(input.selectionStart, input.selectionEnd) || '彩色文字';
  const before = `<span style="color: ${safeColor}">`;
  const after = '</span>';
  const replacement = `${before}${selectedText}${after}`;
  const start = input.selectionStart + before.length;

  return replaceRange(input.text, input.selectionStart, input.selectionEnd, replacement, {
    start,
    end: start + selectedText.length,
  });
};
```

- [ ] **Step 3: Run the utility test and verify it passes**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend && node --test src/utils/markdownEditor.test.mjs
```

Expected: PASS for all existing Markdown action tests and new color tests.

- [ ] **Step 4: Commit the utility change**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao && git add frontend/src/utils/markdownEditor.ts frontend/src/utils/markdownEditor.test.mjs && git commit -m "feat: add markdown color insertion helper"
```

Expected: commit succeeds.

---

### Task 3: Add Source Tests for Component Boundary

**Files:**
- Create: `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/MarkdownWritingStudio.test.mjs`
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/styles/index.test.mjs`

- [ ] **Step 1: Create component-boundary source tests**

Create `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/MarkdownWritingStudio.test.mjs`:

```js
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
```

- [ ] **Step 2: Update shared Markdown style source test**

In `/Users/lizhuang/go/src/wenDao/frontend/src/styles/index.test.mjs`, change the second test so it loads `MarkdownWritingStudio.tsx` and checks the preview class there:

```js
test('front detail and admin preview share article reading styles', async () => {
  const css = `${await loadIndexCss()}\n${await loadMarkdownCss()}`;
  const articleDetail = await loadSourceFile('pages/ArticleDetail.tsx');
  const markdownStudio = await loadSourceFile('views/admin/articles/components/MarkdownWritingStudio.tsx');
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
```

- [ ] **Step 3: Run source tests and verify they fail before component extraction**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend && node --test src/views/admin/articles/MarkdownWritingStudio.test.mjs src/styles/index.test.mjs
```

Expected: FAIL because `components/MarkdownWritingStudio.tsx` does not exist yet.

---

### Task 4: Create MarkdownWritingStudio Component

**Files:**
- Create: `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/components/MarkdownWritingStudio.tsx`
- Test: `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/MarkdownWritingStudio.test.mjs`

- [ ] **Step 1: Create the component file**

Create `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/components/MarkdownWritingStudio.tsx`:

```tsx
import {
  type ClipboardEvent,
  type RefObject,
  Suspense,
  lazy,
  useState,
} from 'react';
import { ColorPicker } from 'tdesign-react';
import {
  Bold,
  Code,
  Eye,
  Heading2,
  ImagePlus,
  Link as LinkIcon,
  List,
  ListOrdered,
  Maximize2,
  Minimize2,
  Minus,
  Palette,
  PanelLeft,
  Pilcrow,
  Quote,
  SplitSquareHorizontal,
  type LucideIcon,
} from 'lucide-react';
import {
  DEFAULT_TEXT_COLOR,
  applyMarkdownAction,
  applyMarkdownColor,
  normalizeMarkdownColor,
  type ApplyMarkdownActionResult,
  type MarkdownAction,
} from '@/utils/markdownEditor';

const ArticlePreview = lazy(() =>
  import('../ArticlePreview').then((module) => ({ default: module.ArticlePreview }))
);

type EditorMode = 'edit' | 'split' | 'preview';

interface ContentStats {
  characters: number;
  lines: number;
  words: number;
  readingMinutes: number;
}

interface MarkdownWritingStudioProps {
  content: string;
  onContentChange: (content: string) => void;
  textareaRef: RefObject<HTMLTextAreaElement>;
  onPaste: (event: ClipboardEvent<HTMLTextAreaElement>) => void;
  onImageUploadClick: () => void;
  contentStats: ContentStats;
  lastSavedTime: string | null;
  isAutoSaving: boolean;
  isImmersive: boolean;
  onImmersiveChange: (isImmersive: boolean) => void;
}

const markdownToolbarActions: Array<{
  action: MarkdownAction;
  label: string;
  icon: LucideIcon;
}> = [
  { action: 'heading', label: '二级标题', icon: Heading2 },
  { action: 'bold', label: '加粗', icon: Bold },
  { action: 'quote', label: '引用', icon: Quote },
  { action: 'unordered-list', label: '无序列表', icon: List },
  { action: 'ordered-list', label: '有序列表', icon: ListOrdered },
  { action: 'inline-code', label: '行内代码', icon: Code },
  { action: 'code-block', label: '代码块', icon: Pilcrow },
  { action: 'link', label: '链接', icon: LinkIcon },
  { action: 'divider', label: '分割线', icon: Minus },
];

const TEXT_COLOR_PRESETS = [
  { label: '红色', value: '#ef4444' },
  { label: '橙色', value: '#f97316' },
  { label: '琥珀', value: '#f59e0b' },
  { label: '绿色', value: '#10b981' },
  { label: '天蓝', value: '#0ea5e9' },
  { label: '靛蓝', value: '#6366f1' },
  { label: '粉色', value: '#ec4899' },
  { label: '灰色', value: '#525252' },
];

const restoreTextareaSelection = (
  textarea: HTMLTextAreaElement,
  result: ApplyMarkdownActionResult
) => {
  requestAnimationFrame(() => {
    textarea.focus();
    textarea.setSelectionRange(result.selection.start, result.selection.end);
  });
};

const insertMarkdownWithUndoStack = (
  textarea: HTMLTextAreaElement,
  result: ApplyMarkdownActionResult
) => {
  textarea.focus();
  textarea.setSelectionRange(result.edit.start, result.edit.end);

  try {
    const canInsertText =
      !document.queryCommandSupported || document.queryCommandSupported('insertText');
    if (!canInsertText) return false;

    const didInsert = document.execCommand('insertText', false, result.edit.replacement);
    if (!didInsert) return false;

    const inputEvent =
      typeof InputEvent === 'function'
        ? new InputEvent('input', {
            bubbles: true,
            inputType: 'insertText',
            data: result.edit.replacement,
          })
        : new Event('input', { bubbles: true });
    textarea.dispatchEvent(inputEvent);
    restoreTextareaSelection(textarea, result);
    return true;
  } catch {
    return false;
  }
};

const tooltipClassName =
  'pointer-events-none absolute left-1/2 top-full z-30 mt-2 -translate-x-1/2 whitespace-nowrap rounded-md border border-neutral-200 bg-white px-2 py-1 text-[11px] font-medium text-neutral-600 opacity-0 shadow-sm transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200';

export const MarkdownWritingStudio = ({
  content,
  onContentChange,
  textareaRef,
  onPaste,
  onImageUploadClick,
  contentStats,
  lastSavedTime,
  isAutoSaving,
  isImmersive,
  onImmersiveChange,
}: MarkdownWritingStudioProps) => {
  const [editorMode, setEditorMode] = useState<EditorMode>('split');
  const [selectedTextColor, setSelectedTextColor] = useState(DEFAULT_TEXT_COLOR);

  const applyEdit = (result: ApplyMarkdownActionResult) => {
    const textarea = textareaRef.current;
    if (textarea && insertMarkdownWithUndoStack(textarea, result)) return;

    onContentChange(result.text);
    if (textarea) restoreTextareaSelection(textarea, result);
  };

  const handleMarkdownAction = (action: MarkdownAction) => {
    const textarea = textareaRef.current;
    const selectionStart = textarea?.selectionStart ?? content.length;
    const selectionEnd = textarea?.selectionEnd ?? selectionStart;

    applyEdit(
      applyMarkdownAction({
        text: content,
        selectionStart,
        selectionEnd,
        action,
      })
    );
  };

  const handleTextColorApply = (color = selectedTextColor) => {
    const normalizedColor = normalizeMarkdownColor(color, selectedTextColor);
    const textarea = textareaRef.current;
    const selectionStart = textarea?.selectionStart ?? content.length;
    const selectionEnd = textarea?.selectionEnd ?? selectionStart;

    setSelectedTextColor(normalizedColor);
    applyEdit(
      applyMarkdownColor(
        {
          text: content,
          selectionStart,
          selectionEnd,
        },
        normalizedColor
      )
    );
  };

  const panelMinHeightClass = isImmersive ? 'min-h-[calc(100vh-220px)]' : 'min-h-[640px]';
  const textareaMinHeightClass = isImmersive ? 'min-h-[calc(100vh-292px)]' : 'min-h-[580px]';

  return (
    <div className="space-y-3">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <label className="block text-sm font-semibold text-neutral-700 dark:text-neutral-200">内容 (Markdown)</label>
          <p className="mt-1 text-xs text-neutral-400 dark:text-neutral-500">
            支持工具栏插入常用 Markdown，粘贴图片会自动上传。
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {(['edit', 'split', 'preview'] as const).map((mode) => {
            const Icon =
              mode === 'edit' ? PanelLeft : mode === 'split' ? SplitSquareHorizontal : Eye;
            return (
              <button
                key={mode}
                type="button"
                onClick={() => setEditorMode(mode)}
                className={`inline-flex h-9 items-center gap-2 rounded-xl px-3 text-xs font-semibold transition-colors ${
                  editorMode === mode
                    ? 'bg-neutral-900 text-white dark:bg-neutral-100 dark:text-neutral-900'
                    : 'bg-neutral-100 text-neutral-500 hover:text-neutral-900 dark:bg-neutral-800 dark:text-neutral-400 dark:hover:text-neutral-100'
                }`}
              >
                <Icon className="h-4 w-4" />
                {mode === 'edit' ? '编辑' : mode === 'split' ? '分屏' : '预览'}
              </button>
            );
          })}
          <button
            type="button"
            onClick={() => onImmersiveChange(!isImmersive)}
            className={`inline-flex h-9 items-center gap-2 rounded-xl px-3 text-xs font-semibold transition-colors ${
              isImmersive
                ? 'bg-primary-600 text-white hover:bg-primary-700'
                : 'bg-primary-50 text-primary-700 hover:bg-primary-100 dark:bg-primary-900/20 dark:text-primary-300 dark:hover:bg-primary-900/30'
            }`}
          >
            {isImmersive ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
            {isImmersive ? '退出专注' : '专注写作'}
          </button>
        </div>
      </div>

      <div className="rounded-2xl border border-neutral-200 bg-neutral-50/70 p-3 dark:border-neutral-800 dark:bg-neutral-950/40">
        <div className="mb-3 flex flex-wrap items-center gap-2">
          {markdownToolbarActions.map((item) => (
            <button
              key={item.action}
              type="button"
              aria-label={item.label}
              onClick={() => handleMarkdownAction(item.action)}
              className="group relative inline-flex h-9 w-9 items-center justify-center rounded-xl text-neutral-500 transition-colors hover:bg-white hover:text-neutral-900 focus-visible:bg-white focus-visible:text-neutral-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100 dark:focus-visible:bg-neutral-800 dark:focus-visible:text-neutral-100"
            >
              <item.icon className="h-4 w-4" aria-hidden="true" />
              <span className={tooltipClassName}>{item.label}</span>
            </button>
          ))}

          <div className="mx-1 h-6 w-px bg-neutral-200 dark:bg-neutral-800" />

          <button
            type="button"
            aria-label="应用当前字体颜色"
            onClick={() => handleTextColorApply()}
            className="group relative inline-flex h-9 w-9 items-center justify-center rounded-xl text-neutral-500 transition-colors hover:bg-white hover:text-neutral-900 focus-visible:bg-white focus-visible:text-neutral-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100 dark:focus-visible:bg-neutral-800 dark:focus-visible:text-neutral-100"
          >
            <Palette className="h-4 w-4" aria-hidden="true" />
            <span
              className="absolute bottom-1 h-0.5 w-5 rounded-full"
              style={{ backgroundColor: selectedTextColor }}
            />
            <span className={tooltipClassName}>字体颜色</span>
          </button>

          <div className="flex h-9 items-center gap-1 rounded-xl bg-white px-2 shadow-sm dark:bg-neutral-900">
            {TEXT_COLOR_PRESETS.map((color) => (
              <button
                key={color.value}
                type="button"
                aria-label={color.label}
                onClick={() => handleTextColorApply(color.value)}
                className={`h-5 w-5 rounded-full border transition-transform hover:scale-110 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 ${
                  selectedTextColor === color.value
                    ? 'border-neutral-900 ring-2 ring-neutral-900/10 dark:border-white dark:ring-white/20'
                    : 'border-white dark:border-neutral-700'
                }`}
                style={{ backgroundColor: color.value }}
              />
            ))}
            <div className="ml-1 w-[112px]">
              <ColorPicker
                value={selectedTextColor}
                format="HEX"
                colorModes={['monochrome']}
                enableAlpha={false}
                recentColors={false}
                swatchColors={TEXT_COLOR_PRESETS.map((color) => color.value)}
                popupProps={{ placement: 'bottom-left' }}
                onChange={(value) => {
                  setSelectedTextColor(normalizeMarkdownColor(value, selectedTextColor));
                }}
              />
            </div>
          </div>

          <button
            type="button"
            aria-label="插入图片"
            onClick={onImageUploadClick}
            className="group relative inline-flex h-9 items-center gap-2 rounded-xl px-3 text-xs font-semibold text-primary-600 transition-colors hover:bg-white focus-visible:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:text-primary-400 dark:hover:bg-neutral-800 dark:focus-visible:bg-neutral-800"
          >
            <ImagePlus className="h-4 w-4" aria-hidden="true" />
            图片
            <span className={tooltipClassName}>插入图片</span>
          </button>

          <div className="ml-auto flex flex-wrap items-center gap-3 text-[11px] font-medium text-neutral-400 dark:text-neutral-500">
            <span>{contentStats.characters} 字符</span>
            <span>{contentStats.lines} 行</span>
            <span>{contentStats.words} 词</span>
            <span>约 {contentStats.readingMinutes} 分钟</span>
          </div>
        </div>

        <div className={`grid gap-4 ${editorMode === 'split' ? 'lg:grid-cols-2' : 'grid-cols-1'}`}>
          {editorMode !== 'preview' && (
            <section className={`flex ${panelMinHeightClass} flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900`}>
              <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-3 dark:border-neutral-800">
                <div>
                  <div className="text-sm font-semibold text-neutral-700 dark:text-neutral-200">Markdown</div>
                  <div className="text-[11px] text-neutral-400 dark:text-neutral-500">原文编辑</div>
                </div>
                {lastSavedTime && (
                  <div className="flex items-center gap-2 text-[10px] font-bold uppercase tracking-wider text-neutral-400">
                    <div className={`h-1.5 w-1.5 rounded-full ${isAutoSaving ? 'animate-pulse bg-amber-400' : 'bg-emerald-400'}`} />
                    {isAutoSaving ? '正在自动保存' : `已保存 ${lastSavedTime}`}
                  </div>
                )}
              </div>
              <textarea
                ref={textareaRef}
                className={`admin-markdown-editor ${textareaMinHeightClass} flex-1 resize-none border-0 bg-transparent px-5 py-4 text-sm leading-7 text-neutral-800 outline-none dark:text-neutral-100`}
                value={content}
                onChange={(event) => onContentChange(event.target.value)}
                onPaste={onPaste}
                placeholder="使用 Markdown 编写内容..."
              />
            </section>
          )}

          {editorMode !== 'edit' && (
            <section className={`flex ${panelMinHeightClass} flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900`}>
              <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-3 dark:border-neutral-800">
                <div>
                  <div className="text-sm font-semibold text-neutral-700 dark:text-neutral-200">Preview</div>
                  <div className="text-[11px] text-neutral-400 dark:text-neutral-500">编辑时预览</div>
                </div>
              </div>
              <div className="article-reading-body admin-markdown-preview flex-1 overflow-y-auto px-6 py-5">
                {content.trim() ? (
                  <Suspense fallback={<div className="h-full animate-pulse rounded-xl bg-neutral-100 dark:bg-neutral-800" />}>
                    <ArticlePreview content={content} />
                  </Suspense>
                ) : (
                  <div className="flex h-full min-h-[420px] items-center justify-center rounded-xl border border-dashed border-neutral-200 text-sm text-neutral-400 dark:border-neutral-800 dark:text-neutral-500">
                    预览会在这里显示
                  </div>
                )}
              </div>
            </section>
          )}
        </div>
      </div>
    </div>
  );
};
```

- [ ] **Step 2: Run the component source test and verify partial progress**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend && node --test src/views/admin/articles/MarkdownWritingStudio.test.mjs
```

Expected: still FAIL because `ArticleEditor` has not delegated to `MarkdownWritingStudio` yet.

---

### Task 5: Wire ArticleEditor to MarkdownWritingStudio

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/ArticleEditor.tsx`
- Test: `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/MarkdownWritingStudio.test.mjs`

- [ ] **Step 1: Replace editor-specific imports**

In `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/ArticleEditor.tsx`, change the import block:

```tsx
import { type ChangeEvent, useEffect, useMemo, useRef, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation } from '@tanstack/react-query';
import { Select } from 'tdesign-react';
import { articleApi, categoryApi, uploadApi, chatApi } from '@/api';
import { Loading } from '@/components/common';
import { useUIStore } from '@/store';
import { getArticlePrimaryActionLabel } from '@/utils/pageBehavior';
import { MarkdownWritingStudio } from './components/MarkdownWritingStudio';
import 'tdesign-react/es/style/index.css';
```

Remove these now-unused local definitions and helpers from `ArticleEditor.tsx`:

```tsx
const ArticlePreview = lazy(() =>
  import('./ArticlePreview').then((module) => ({ default: module.ArticlePreview }))
);

type EditorMode = 'edit' | 'split' | 'preview';

const markdownToolbarActions = ...
const restoreTextareaSelection = ...
const insertMarkdownWithUndoStack = ...
const handleMarkdownAction = ...
```

- [ ] **Step 2: Add page-level immersive layout state**

After the `isAutoSaving` state, add:

```tsx
const [isWritingFocused, setIsWritingFocused] = useState(false);
```

- [ ] **Step 3: Replace the outer page classes**

Change the outer page wrapper:

```tsx
<div className={`${isWritingFocused ? 'max-w-display' : 'max-w-6xl'} mx-auto pb-12 transition-[max-width] duration-300`}>
```

Change the form shell wrapper:

```tsx
<div
  className={
    isWritingFocused
      ? 'space-y-5'
      : 'space-y-6 rounded-xl border border-neutral-100 bg-white p-8 shadow-sm dark:border-neutral-800 dark:bg-neutral-900'
  }
>
```

- [ ] **Step 4: Make metadata compact in focused mode**

Wrap the metadata grid with a conditional container:

```tsx
<div
  className={
    isWritingFocused
      ? 'rounded-2xl border border-neutral-100 bg-white p-5 shadow-sm dark:border-neutral-800 dark:bg-neutral-900'
      : ''
  }
>
  <div className={`grid grid-cols-1 gap-6 ${isWritingFocused ? 'xl:grid-cols-[minmax(0,1fr)_280px]' : 'lg:grid-cols-3'}`}>
    <div className={`${isWritingFocused ? 'space-y-4' : 'lg:col-span-2 space-y-6'}`}>
      {/* keep the existing title, category/status, and summary controls here */}
    </div>

    <div>
      {/* keep the existing cover upload control here */}
    </div>
  </div>
</div>
```

Keep the existing field markup inside those slots. Change the summary textarea height class to:

```tsx
className={`input w-full py-2 ${isWritingFocused ? 'h-20' : 'h-24'}`}
```

- [ ] **Step 5: Replace the inline Markdown content section**

Replace the current content section beginning at:

```tsx
<div className="space-y-3">
```

and ending after the preview panel with:

```tsx
<MarkdownWritingStudio
  content={formData.content}
  onContentChange={(content) => setFormData((prev) => ({ ...prev, content }))}
  textareaRef={contentInputRef}
  onPaste={handleContentPaste}
  onImageUploadClick={() => contentImageInputRef.current?.click()}
  contentStats={contentStats}
  lastSavedTime={lastSavedTime}
  isAutoSaving={isAutoSaving}
  isImmersive={isWritingFocused}
  onImmersiveChange={setIsWritingFocused}
/>

<input
  ref={contentImageInputRef}
  type="file"
  className="hidden"
  accept="image/*"
  onChange={(event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) handleImageUpload(file, 'content');
    event.target.value = '';
  }}
/>
```

- [ ] **Step 6: Run source tests and verify they pass**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend && node --test src/views/admin/articles/MarkdownWritingStudio.test.mjs src/styles/index.test.mjs
```

Expected: PASS.

- [ ] **Step 7: Commit component extraction**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao && git add frontend/src/views/admin/articles/ArticleEditor.tsx frontend/src/views/admin/articles/components/MarkdownWritingStudio.tsx frontend/src/views/admin/articles/MarkdownWritingStudio.test.mjs frontend/src/styles/index.test.mjs && git commit -m "feat: extract markdown writing studio"
```

Expected: commit succeeds.

---

### Task 6: Build, Lint, and Manual UI Verification

**Files:**
- Verify: `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/ArticleEditor.tsx`
- Verify: `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/components/MarkdownWritingStudio.tsx`
- Verify: `/Users/lizhuang/go/src/wenDao/frontend/src/utils/markdownEditor.ts`

- [ ] **Step 1: Run focused tests**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend && node --test src/utils/markdownEditor.test.mjs src/views/admin/articles/MarkdownWritingStudio.test.mjs src/styles/index.test.mjs
```

Expected: PASS.

- [ ] **Step 2: Run frontend build**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend && npm run build
```

Expected: `tsc` and `vite build` complete successfully.

- [ ] **Step 3: Run frontend lint**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend && npm run lint
```

Expected: ESLint completes with zero warnings.

- [ ] **Step 4: Start Vite for manual verification**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend && npm run dev
```

Expected: Vite starts and prints a local URL, normally `http://localhost:3000/`.

- [ ] **Step 5: Verify editor behavior in the browser**

Open the admin article editor and verify:

- normal mode shows metadata above the writing studio,
- toolbar actions still insert existing Markdown syntax,
- clicking a preset color inserts `<span style="color: #...">...</span>`,
- choosing a custom color in `ColorPicker` changes the active color,
- clicking the palette button applies the active custom color,
- preview renders colored text,
- `专注写作` expands the page and increases editor height,
- `退出专注` returns to the normal layout,
- edit, split, and preview modes still switch cleanly,
- image insertion and pasted image upload still work,
- desktop and mobile widths do not show overlapping toolbar controls.

- [ ] **Step 6: Stop the dev server**

If Vite is still running in the terminal session, stop it with `Ctrl-C`.

- [ ] **Step 7: Commit verification-only fixes if needed**

If build, lint, or manual verification required code adjustments, run:

```bash
cd /Users/lizhuang/go/src/wenDao && git add frontend/src && git commit -m "fix: polish markdown writing studio"
```

Expected: no commit is created if no fixes were needed; otherwise commit succeeds.

---

## Self-Review

Spec coverage:
- Font color presets and custom color are covered by Tasks 1, 2, and 4.
- Safe color span insertion is covered by Tasks 1 and 2.
- Component-first extraction is covered by Tasks 3, 4, and 5.
- Immersive writing mode is covered by Tasks 4 and 5.
- Existing autosave, uploads, preview, and article save behavior are preserved by keeping `ArticleEditor` ownership and passing controlled props in Task 5.
- Build, lint, and manual verification are covered by Task 6.

Type consistency:
- The plan uses `MarkdownWritingStudio`, `applyMarkdownColor`, `normalizeMarkdownColor`, `DEFAULT_TEXT_COLOR`, `isWritingFocused`, and `setIsWritingFocused` consistently across tests and implementation steps.
- `MarkdownWritingStudio` receives controlled content and emits content changes through `onContentChange`.
- Page-level immersive width is owned by `ArticleEditor`; the toggle UI is owned by `MarkdownWritingStudio` through `onImmersiveChange`.
