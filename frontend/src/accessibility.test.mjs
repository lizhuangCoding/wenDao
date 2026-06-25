import assert from 'node:assert/strict';
import { readdir, readFile, stat } from 'node:fs/promises';
import path from 'node:path';
import test from 'node:test';

const sourceRoot = new URL('./', import.meta.url);

const collectSourceFiles = async (directoryUrl) => {
  const entries = await readdir(directoryUrl, { withFileTypes: true });
  const files = await Promise.all(
    entries.map(async (entry) => {
      const entryUrl = new URL(`${entry.name}${entry.isDirectory() ? '/' : ''}`, directoryUrl);
      if (entry.isDirectory()) {
        return collectSourceFiles(entryUrl);
      }
      const filePath = entryUrl.pathname;
      return /\.(tsx|ts)$/.test(filePath) ? [filePath] : [];
    })
  );
  return files.flat();
};

test('native button elements declare an explicit type', async () => {
  await stat(sourceRoot);
  const files = await collectSourceFiles(sourceRoot);
  const missingType = [];

  for (const file of files) {
    const source = await readFile(file, 'utf8');
    for (const match of source.matchAll(/<button\b[\s\S]*?>/g)) {
      if (!/\btype=/.test(match[0])) {
        const line = source.slice(0, match.index).split('\n').length;
        missingType.push(`${path.relative(sourceRoot.pathname, file)}:${line}`);
      }
    }
  }

  assert.deepEqual(missingType, []);
});

test('confirmation modal exposes dialog semantics and keyboard dismissal', async () => {
  const source = await readFile(new URL('./components/common/ConfirmModal.tsx', import.meta.url), 'utf8');

  assert.match(source, /role="dialog"/);
  assert.match(source, /aria-modal="true"/);
  assert.match(source, /aria-labelledby=\{titleId\}/);
  assert.match(source, /aria-describedby=\{messageId\}/);
  assert.match(source, /useId/);
  assert.match(source, /Escape/);
});
