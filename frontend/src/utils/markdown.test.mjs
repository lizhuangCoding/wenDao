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

test('markdownToPlainText returns readable notification previews without markdown syntax', async () => {
  const { markdownToPlainText } = await loadMarkdownUtils();

  const preview = markdownToPlainText(`
# 系统通知

请阅读 **重要更新**，并查看 [发布说明](/release)。

- 支持 Markdown
- 不显示语法
`);

  assert.equal(preview, '系统通知 请阅读 重要更新，并查看 发布说明。 支持 Markdown 不显示语法');
  assert.doesNotMatch(preview, /[#*[\]()]/);
});

test('estimateReadingTime gives higher weight to code-heavy articles', async () => {
  const { estimateReadingTime } = await loadMarkdownUtils();

  const proseOnly = estimateReadingTime(`
这是一篇短文，主要是普通正文内容。

It has a short English paragraph as well.
`);

  const codeLines = Array.from(
    { length: 40 },
    (_, index) => `const value${index} = ${index};`
  ).join('\n');

  const codeHeavy = estimateReadingTime(`
这是一篇短文，主要是普通正文内容。

\`\`\`ts
${codeLines}
\`\`\`
`);

  assert.ok(proseOnly >= 1);
  assert.ok(codeHeavy > proseOnly);
});
