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
  assert.deepEqual(result.edit, {
    start: 6,
    end: 11,
    replacement: '**world**',
    selection: { start: 8, end: 13 },
  });
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
