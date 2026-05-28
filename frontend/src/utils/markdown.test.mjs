import assert from 'node:assert/strict';
import { rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { build } from 'esbuild';

const tempDir = path.join(tmpdir(), 'wendao-markdown-tests');
const bundlePath = path.join(tempDir, 'markdown.test-bundle.mjs');

const loadMarkdownUtils = async () => {
  await build({
    entryPoints: [new URL('./markdown.ts', import.meta.url).pathname],
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

test('extractHeadings uses visible text from inline HTML headings', async () => {
  const { extractHeadings } = await loadMarkdownUtils();

  const headings = extractHeadings('# <span style="color: #f97316">你好</span>');

  assert.deepEqual(headings, [
    {
      id: '你好',
      text: '你好',
      level: 1,
    },
  ]);
});
