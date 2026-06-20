import assert from 'node:assert/strict';
import { rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { build } from 'esbuild';

const tempDir = path.join(tmpdir(), 'wendao-article-export-tests');
const bundlePath = path.join(tempDir, 'articleExport.test-bundle.mjs');

const loadArticleExportUtils = async () => {
  await build({
    entryPoints: [new URL('./articleExport.ts', import.meta.url).pathname],
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

const article = {
  title: 'Go/React: 搜索体验优化?',
  summary: '让搜索更容易继续。',
  content: '## 正文\n\n这里是文章正文。',
  category: { name: '工程' },
  author: { username: 'lizhuang' },
  tags: [{ name: 'Go' }, { name: 'React' }],
  published_at: '2026-06-20T08:00:00Z',
  created_at: '2026-06-19T08:00:00Z',
};

test('buildArticleMarkdown exports article metadata and body as markdown', async () => {
  const { buildArticleMarkdown } = await loadArticleExportUtils();

  const markdown = buildArticleMarkdown(article);

  assert.match(markdown, /^# Go\/React: 搜索体验优化\?/);
  assert.match(markdown, /> 让搜索更容易继续。/);
  assert.match(markdown, /- 分类：工程/);
  assert.match(markdown, /- 标签：Go, React/);
  assert.match(markdown, /- 作者：lizhuang/);
  assert.match(markdown, /## 正文/);
});

test('getArticleMarkdownFilename returns a safe markdown filename', async () => {
  const { getArticleMarkdownFilename } = await loadArticleExportUtils();

  assert.equal(getArticleMarkdownFilename(article), 'Go_React_ 搜索体验优化_.md');
});
