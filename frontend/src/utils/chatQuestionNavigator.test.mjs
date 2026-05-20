import assert from 'node:assert/strict';
import { rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { build } from 'esbuild';

const tempDir = path.join(tmpdir(), 'wendao-chat-question-navigator-tests');
const bundlePath = path.join(tempDir, 'chatQuestionNavigator.test-bundle.mjs');

const loadNavigator = async () => {
  await build({
    entryPoints: [new URL('./chatQuestionNavigator.ts', import.meta.url).pathname],
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

test('buildChatQuestionNavItems creates anchors from user questions only', async () => {
  const { buildChatQuestionNavItems } = await loadNavigator();

  const items = buildChatQuestionNavItems([
    { id: 'assistant-1', role: 'assistant', content: '我可以帮你排查。', timestamp: 1 },
    { id: 'user:swap', role: 'user', content: '  Swap   的问题\n要怎么排查？ ', timestamp: 2 },
    { id: 'user-empty', role: 'user', content: '   ', timestamp: 3 },
    { id: 'assistant-2', role: 'assistant', content: '先看约束。', timestamp: 4 },
    { id: 'user/actions', role: 'user', content: 'GitHub Actions 失败怎么办？', timestamp: 5 },
  ]);

  assert.deepEqual(
    items.map((item) => ({
      anchorId: item.anchorId,
      index: item.index,
      label: item.label,
      messageId: item.messageId,
      timestamp: item.timestamp,
    })),
    [
      {
        anchorId: 'chat-question-1-user-swap',
        index: 1,
        label: 'Swap 的问题 要怎么排查？',
        messageId: 'user:swap',
        timestamp: 2,
      },
      {
        anchorId: 'chat-question-2-user-actions',
        index: 2,
        label: 'GitHub Actions 失败怎么办？',
        messageId: 'user/actions',
        timestamp: 5,
      },
    ]
  );
});

test('buildChatQuestionNavItems shortens very long labels without losing full text', async () => {
  const { buildChatQuestionNavItems } = await loadNavigator();

  const [item] = buildChatQuestionNavItems(
    [{ id: 'long-question', role: 'user', content: 'abcdefghijklmnopqrstuvwxyz', timestamp: 1 }],
    12
  );

  assert.equal(item.label, 'abcdefghijkl...');
  assert.equal(item.fullText, 'abcdefghijklmnopqrstuvwxyz');
});
