# Admin Markdown Editor Enhancement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade the admin article Markdown input and edit-time preview into a lightweight Markdown writing workspace without changing public article rendering.

**Architecture:** Keep `ArticleEditor` as the owner of article form state, auto-save, and upload behavior. Add a pure Markdown editing utility for deterministic toolbar transformations, then replace the content section with a controlled editor workspace that uses the existing textarea and lazy `ArticlePreview`.

**Tech Stack:** React 18, TypeScript, Tailwind CSS, lucide-react, ReactMarkdown, Node `node:test`, esbuild bundling pattern used by existing utility tests.

---

## File Structure

**Create:**
- `/Users/lizhuang/go/src/wenDao/frontend/src/utils/markdownEditor.ts` - pure text transformation helpers for Markdown toolbar actions.
- `/Users/lizhuang/go/src/wenDao/frontend/src/utils/markdownEditor.test.mjs` - Node tests for toolbar transformations.

**Modify:**
- `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/ArticleEditor.tsx` - replace the Markdown content area with toolbar, mode switching, improved editor panel, and improved preview panel.
- `/Users/lizhuang/go/src/wenDao/frontend/src/styles/index.css` - add admin-scoped preview and editor textarea styling.

---

### Task 1: Add Markdown Editor Utility Tests

**Files:**
- Create: `/Users/lizhuang/go/src/wenDao/frontend/src/utils/markdownEditor.test.mjs`

- [ ] **Step 1: Write the failing tests**

Create `/Users/lizhuang/go/src/wenDao/frontend/src/utils/markdownEditor.test.mjs`:

```js
import assert from 'node:assert/strict';
import { rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { build } from 'esbuild';

const tempDir = path.join(tmpdir(), 'wendao-markdown-editor-tests');
const bundlePath = path.join(tempDir, 'markdownEditor.test-bundle.mjs');

const loadMarkdownEditor = async () => {
  await build({
    entryPoints: [new URL('./markdownEditor.ts', import.meta.url).pathname],
    bundle: true,
    format: 'esm',
    outfile: bundlePath,
    platform: 'node',
  });

  return import(`file://${bundlePath}?cache=${Date.now()}`);
};

test.after(async () => {
  await rm(tempDir, { recursive: true, force: true });
});

test('applyMarkdownAction wraps selected text in bold markers', async () => {
  const { applyMarkdownAction } = await loadMarkdownEditor();

  const result = applyMarkdownAction({
    text: 'hello world',
    selectionStart: 6,
    selectionEnd: 11,
    action: 'bold',
  });

  assert.equal(result.text, 'hello **world**');
  assert.deepEqual(result.selection, { start: 8, end: 13 });
});

test('applyMarkdownAction inserts heading marker at the current line', async () => {
  const { applyMarkdownAction } = await loadMarkdownEditor();

  const result = applyMarkdownAction({
    text: 'intro\nsection title',
    selectionStart: 8,
    selectionEnd: 15,
    action: 'heading',
  });

  assert.equal(result.text, 'intro\n## section title');
  assert.deepEqual(result.selection, { start: 11, end: 18 });
});

test('applyMarkdownAction converts selected lines to unordered list items', async () => {
  const { applyMarkdownAction } = await loadMarkdownEditor();

  const result = applyMarkdownAction({
    text: 'alpha\nbeta',
    selectionStart: 0,
    selectionEnd: 10,
    action: 'unordered-list',
  });

  assert.equal(result.text, '- alpha\n- beta');
  assert.deepEqual(result.selection, { start: 2, end: 14 });
});

test('applyMarkdownAction inserts fenced code block with cursor inside', async () => {
  const { applyMarkdownAction } = await loadMarkdownEditor();

  const result = applyMarkdownAction({
    text: 'before\n',
    selectionStart: 7,
    selectionEnd: 7,
    action: 'code-block',
  });

  assert.equal(result.text, 'before\n```text\n\n```');
  assert.deepEqual(result.selection, { start: 15, end: 15 });
});

test('applyMarkdownAction creates a link skeleton when no text is selected', async () => {
  const { applyMarkdownAction } = await loadMarkdownEditor();

  const result = applyMarkdownAction({
    text: '',
    selectionStart: 0,
    selectionEnd: 0,
    action: 'link',
  });

  assert.equal(result.text, '[链接文本](https://example.com)');
  assert.deepEqual(result.selection, { start: 1, end: 5 });
});
```

- [ ] **Step 2: Run the test and verify it fails**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/frontend && node --test src/utils/markdownEditor.test.mjs
```

Expected: FAIL because `frontend/src/utils/markdownEditor.ts` does not exist.

---

### Task 2: Implement Markdown Editor Utility

**Files:**
- Create: `/Users/lizhuang/go/src/wenDao/frontend/src/utils/markdownEditor.ts`
- Test: `/Users/lizhuang/go/src/wenDao/frontend/src/utils/markdownEditor.test.mjs`

- [ ] **Step 1: Add the minimal implementation**

Create `/Users/lizhuang/go/src/wenDao/frontend/src/utils/markdownEditor.ts`:

```ts
export type MarkdownAction =
  | 'heading'
  | 'bold'
  | 'quote'
  | 'unordered-list'
  | 'ordered-list'
  | 'code-block'
  | 'inline-code'
  | 'link'
  | 'divider';

interface ApplyMarkdownActionInput {
  text: string;
  selectionStart: number;
  selectionEnd: number;
  action: MarkdownAction;
}

interface TextSelection {
  start: number;
  end: number;
}

interface ApplyMarkdownActionResult {
  text: string;
  selection: TextSelection;
}

const replaceRange = (
  text: string,
  start: number,
  end: number,
  replacement: string,
  selection: TextSelection
): ApplyMarkdownActionResult => ({
  text: `${text.slice(0, start)}${replacement}${text.slice(end)}`,
  selection,
});

const getSelectedText = ({ text, selectionStart, selectionEnd }: ApplyMarkdownActionInput) =>
  text.slice(selectionStart, selectionEnd);

const getLineBounds = (text: string, selectionStart: number, selectionEnd: number) => {
  const lineStart = text.lastIndexOf('\n', Math.max(0, selectionStart - 1)) + 1;
  const nextBreak = text.indexOf('\n', selectionEnd);
  const lineEnd = nextBreak === -1 ? text.length : nextBreak;
  return { lineStart, lineEnd };
};

const prefixSelectedLines = (
  input: ApplyMarkdownActionInput,
  getPrefix: (index: number) => string
): ApplyMarkdownActionResult => {
  const { lineStart, lineEnd } = getLineBounds(input.text, input.selectionStart, input.selectionEnd);
  const selectedBlock = input.text.slice(lineStart, lineEnd);
  const lines = selectedBlock.split('\n');
  const replacement = lines.map((line, index) => `${getPrefix(index)}${line}`).join('\n');
  const prefixLength = getPrefix(0).length;

  return replaceRange(input.text, lineStart, lineEnd, replacement, {
    start: input.selectionStart + prefixLength,
    end: input.selectionEnd + prefixLength * lines.length,
  });
};

const wrapSelection = (
  input: ApplyMarkdownActionInput,
  before: string,
  after: string,
  fallback: string
): ApplyMarkdownActionResult => {
  const selectedText = getSelectedText(input) || fallback;
  const replacement = `${before}${selectedText}${after}`;
  const start = input.selectionStart + before.length;

  return replaceRange(input.text, input.selectionStart, input.selectionEnd, replacement, {
    start,
    end: start + selectedText.length,
  });
};

export const applyMarkdownAction = (input: ApplyMarkdownActionInput): ApplyMarkdownActionResult => {
  switch (input.action) {
    case 'bold':
      return wrapSelection(input, '**', '**', '加粗文字');
    case 'inline-code':
      return wrapSelection(input, '`', '`', 'code');
    case 'heading': {
      const { lineStart, lineEnd } = getLineBounds(input.text, input.selectionStart, input.selectionEnd);
      const line = input.text.slice(lineStart, lineEnd);
      const replacement = line.startsWith('## ') ? line : `## ${line.replace(/^#{1,6}\s+/, '')}`;
      const delta = replacement.length - line.length;
      return replaceRange(input.text, lineStart, lineEnd, replacement, {
        start: input.selectionStart + Math.max(delta, 0),
        end: input.selectionEnd + Math.max(delta, 0),
      });
    }
    case 'quote':
      return prefixSelectedLines(input, () => '> ');
    case 'unordered-list':
      return prefixSelectedLines(input, () => '- ');
    case 'ordered-list':
      return prefixSelectedLines(input, (index) => `${index + 1}. `);
    case 'code-block': {
      const selectedText = getSelectedText(input);
      const replacement = `\`\`\`text\n${selectedText}\n\`\`\``;
      const cursor = input.selectionStart + '```text\n'.length;
      return replaceRange(input.text, input.selectionStart, input.selectionEnd, replacement, {
        start: selectedText ? cursor : cursor,
        end: selectedText ? cursor + selectedText.length : cursor,
      });
    }
    case 'link': {
      const selectedText = getSelectedText(input) || '链接文本';
      const replacement = `[${selectedText}](https://example.com)`;
      const start = input.selectionStart + 1;
      return replaceRange(input.text, input.selectionStart, input.selectionEnd, replacement, {
        start,
        end: start + selectedText.length,
      });
    }
    case 'divider': {
      const needsLeadingBreak = input.selectionStart > 0 && input.text[input.selectionStart - 1] !== '\n';
      const needsTrailingBreak = input.selectionEnd < input.text.length && input.text[input.selectionEnd] !== '\n';
      const replacement = `${needsLeadingBreak ? '\n' : ''}---${needsTrailingBreak ? '\n' : ''}`;
      const cursor = input.selectionStart + replacement.length;
      return replaceRange(input.text, input.selectionStart, input.selectionEnd, replacement, {
        start: cursor,
        end: cursor,
      });
    }
  }
};
```

- [ ] **Step 2: Run the focused utility tests**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/frontend && node --test src/utils/markdownEditor.test.mjs
```

Expected: PASS.

---

### Task 3: Connect Toolbar and View Modes in ArticleEditor

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/ArticleEditor.tsx`
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/styles/index.css`

- [ ] **Step 1: Update imports**

In `ArticleEditor.tsx`, add `ChangeEvent` and lucide icons, and import the utility:

```ts
import { ChangeEvent, Suspense, lazy, useEffect, useMemo, useRef, useState } from 'react';
import {
  Bold,
  Code,
  Eye,
  Heading2,
  ImagePlus,
  Link as LinkIcon,
  List,
  ListOrdered,
  Minus,
  PanelLeft,
  Pilcrow,
  Quote,
  SplitSquareHorizontal,
  type LucideIcon,
} from 'lucide-react';
import { applyMarkdownAction, type MarkdownAction } from '@/utils/markdownEditor';
```

- [ ] **Step 2: Add editor mode and file input state**

Inside `ArticleEditor`, near `contentInputRef`, add:

```ts
type EditorMode = 'edit' | 'split' | 'preview';
const [editorMode, setEditorMode] = useState<EditorMode>('split');
const contentImageInputRef = useRef<HTMLInputElement>(null);
```

- [ ] **Step 3: Add toolbar action handler**

Inside `ArticleEditor`, before `handleImageUpload`, add:

```ts
const handleMarkdownAction = (action: MarkdownAction) => {
  const textarea = contentInputRef.current;
  const selectionStart = textarea?.selectionStart ?? formData.content.length;
  const selectionEnd = textarea?.selectionEnd ?? selectionStart;

  const result = applyMarkdownAction({
    text: formData.content,
    selectionStart,
    selectionEnd,
    action,
  });

  setFormData((prev) => ({ ...prev, content: result.text }));
  requestAnimationFrame(() => {
    const nextTextarea = contentInputRef.current;
    if (!nextTextarea) return;
    nextTextarea.focus();
    nextTextarea.setSelectionRange(result.selection.start, result.selection.end);
  });
};
```

- [ ] **Step 4: Add content stats**

Inside `ArticleEditor`, before the `if (isEdit && isArticleLoading) return <Loading />;` guard, add:

```ts
const contentStats = useMemo(() => {
  const trimmed = formData.content.trim();
  const lineCount = formData.content ? formData.content.split('\n').length : 0;
  const cjkCount = (trimmed.match(/[\u4e00-\u9fff]/g) || []).length;
  const wordCount = trimmed
    .replace(/[\u4e00-\u9fff]/g, '')
    .split(/\s+/)
    .filter(Boolean).length + cjkCount;
  const readingMinutes = Math.max(1, Math.ceil(wordCount / 450));

  return {
    characters: formData.content.length,
    lines: lineCount,
    words: wordCount,
    readingMinutes,
  };
}, [formData.content]);
```

- [ ] **Step 5: Replace the existing content section JSX**

Replace the current `<div>` that starts with label `内容 (Markdown)` through the end of its two-column grid with a toolbar workspace:

```tsx
<div className="space-y-3">
  <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
    <div>
      <label className="block text-sm font-semibold text-neutral-700 dark:text-neutral-200">内容 (Markdown)</label>
      <p className="mt-1 text-xs text-neutral-400 dark:text-neutral-500">
        支持工具栏插入常用 Markdown，粘贴图片会自动上传。
      </p>
    </div>
    <div className="flex flex-wrap items-center gap-2">
      {(['edit', 'split', 'preview'] as const).map((mode) => (
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
          {mode === 'edit' ? <PanelLeft className="h-4 w-4" /> : mode === 'split' ? <SplitSquareHorizontal className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          {mode === 'edit' ? '编辑' : mode === 'split' ? '分屏' : '预览'}
        </button>
      ))}
    </div>
  </div>

  <div className="rounded-2xl border border-neutral-200 bg-neutral-50/70 p-3 dark:border-neutral-800 dark:bg-neutral-950/40">
    <div className="mb-3 flex flex-wrap items-center gap-2">
      {markdownToolbarActions.map((item) => (
        <button
          key={item.action}
          type="button"
          title={item.label}
          onClick={() => handleMarkdownAction(item.action)}
          className="inline-flex h-9 w-9 items-center justify-center rounded-xl text-neutral-500 transition-colors hover:bg-white hover:text-neutral-900 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100"
        >
          <item.icon className="h-4 w-4" />
        </button>
      ))}
      <button
        type="button"
        title="插入图片"
        onClick={() => contentImageInputRef.current?.click()}
        className="inline-flex h-9 items-center gap-2 rounded-xl px-3 text-xs font-semibold text-primary-600 transition-colors hover:bg-white dark:text-primary-400 dark:hover:bg-neutral-800"
      >
        <ImagePlus className="h-4 w-4" />
        图片
      </button>
      <input
        ref={contentImageInputRef}
        type="file"
        className="hidden"
        accept="image/*"
        onChange={(e: ChangeEvent<HTMLInputElement>) => {
          const file = e.target.files?.[0];
          if (file) handleImageUpload(file, 'content');
          e.target.value = '';
        }}
      />
      <div className="ml-auto flex flex-wrap items-center gap-3 text-[11px] font-medium text-neutral-400 dark:text-neutral-500">
        <span>{contentStats.characters} 字符</span>
        <span>{contentStats.lines} 行</span>
        <span>约 {contentStats.readingMinutes} 分钟</span>
      </div>
    </div>

    <div className={`grid gap-4 ${editorMode === 'split' ? 'lg:grid-cols-2' : 'grid-cols-1'}`}>
      {editorMode !== 'preview' && (
        <section className="flex min-h-[540px] flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
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
            ref={contentInputRef}
            className="admin-markdown-editor min-h-[480px] flex-1 resize-none border-0 bg-transparent px-5 py-4 text-sm leading-7 text-neutral-800 outline-none dark:text-neutral-100"
            value={formData.content}
            onChange={(e) => setFormData({ ...formData, content: e.target.value })}
            onPaste={handleContentPaste}
            placeholder="使用 Markdown 编写内容..."
          />
        </section>
      )}

      {editorMode !== 'edit' && (
        <section className="flex min-h-[540px] flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
          <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-3 dark:border-neutral-800">
            <div>
              <div className="text-sm font-semibold text-neutral-700 dark:text-neutral-200">Preview</div>
              <div className="text-[11px] text-neutral-400 dark:text-neutral-500">编辑时预览</div>
            </div>
          </div>
          <div className="admin-markdown-preview flex-1 overflow-y-auto px-6 py-5">
            {formData.content.trim() ? (
              <Suspense fallback={<div className="h-full animate-pulse rounded-xl bg-neutral-100 dark:bg-neutral-800" />}>
                <ArticlePreview content={formData.content} />
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
```

- [ ] **Step 6: Add toolbar action config**

Place this constant above `export const ArticleEditor`:

```ts
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
```

- [ ] **Step 7: Add admin-scoped styles**

Append to `/Users/lizhuang/go/src/wenDao/frontend/src/styles/index.css` inside `@layer components`:

```css
  .admin-markdown-editor {
    font-family: "JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
    tab-size: 2;
  }

  .admin-markdown-editor::placeholder {
    @apply text-neutral-400;
  }

  .dark .admin-markdown-editor::placeholder {
    @apply text-neutral-500;
  }

  .admin-markdown-preview {
    @apply text-neutral-700 dark:text-neutral-200;
    font-size: 15px;
    line-height: 1.8;
  }

  .admin-markdown-preview h1 {
    @apply mb-4 mt-2 text-3xl font-black text-neutral-900 dark:text-neutral-100;
  }

  .admin-markdown-preview h2 {
    @apply mb-3 mt-7 text-2xl font-bold text-neutral-900 dark:text-neutral-100;
  }

  .admin-markdown-preview h3 {
    @apply mb-2 mt-6 text-xl font-semibold text-neutral-800 dark:text-neutral-100;
  }

  .admin-markdown-preview p {
    @apply my-4;
  }

  .admin-markdown-preview a {
    @apply text-primary-600 underline underline-offset-4 dark:text-primary-400;
  }

  .admin-markdown-preview blockquote {
    @apply my-5 border-l-4 border-primary-500 bg-primary-50/60 py-2 pl-4 pr-3 text-neutral-600 dark:bg-primary-900/10 dark:text-neutral-300;
  }

  .admin-markdown-preview ul,
  .admin-markdown-preview ol {
    @apply my-4 pl-6;
  }

  .admin-markdown-preview ul {
    @apply list-disc;
  }

  .admin-markdown-preview ol {
    @apply list-decimal;
  }

  .admin-markdown-preview li {
    @apply my-1;
  }

  .admin-markdown-preview code {
    @apply rounded-md bg-neutral-100 px-1.5 py-0.5 font-mono text-[0.9em] text-primary-700 dark:bg-neutral-800 dark:text-primary-300;
  }

  .admin-markdown-preview pre {
    @apply my-5 overflow-x-auto rounded-xl bg-neutral-950 p-4 text-neutral-100 dark:bg-black;
  }

  .admin-markdown-preview pre code {
    @apply bg-transparent p-0 text-neutral-100;
  }

  .admin-markdown-preview table {
    @apply my-5 w-full border-collapse overflow-hidden text-sm;
  }

  .admin-markdown-preview th,
  .admin-markdown-preview td {
    @apply border border-neutral-200 px-3 py-2 dark:border-neutral-700;
  }

  .admin-markdown-preview th {
    @apply bg-neutral-100 font-semibold dark:bg-neutral-800;
  }

  .admin-markdown-preview img {
    @apply my-5 max-w-full rounded-xl border border-neutral-100 shadow-sm dark:border-neutral-800;
  }

  .admin-markdown-preview hr {
    @apply my-8 border-neutral-200 dark:border-neutral-700;
  }
```

- [ ] **Step 8: Run TypeScript build**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/frontend && npm run build
```

Expected: PASS.

---

### Task 4: Verification and Commit

**Files:**
- Verify all files changed in previous tasks.

- [ ] **Step 1: Run utility tests**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/frontend && node --test src/utils/markdownEditor.test.mjs
```

Expected: PASS.

- [ ] **Step 2: Run frontend build**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/frontend && npm run build
```

Expected: PASS.

- [ ] **Step 3: Run frontend lint**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/frontend && npm run lint
```

Expected: PASS.

- [ ] **Step 4: Inspect diff**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao && git diff --stat
```

Expected: only the Markdown editor utility, test, `ArticleEditor.tsx`, and `index.css` are changed.

- [ ] **Step 5: Commit implementation**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao && git add frontend/src/utils/markdownEditor.ts frontend/src/utils/markdownEditor.test.mjs frontend/src/views/admin/articles/ArticleEditor.tsx frontend/src/styles/index.css
git commit -m "feat: enhance admin markdown editor"
```

Expected: commit succeeds.
