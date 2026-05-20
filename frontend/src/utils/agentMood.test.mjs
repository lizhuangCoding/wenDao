import assert from 'node:assert/strict';
import { rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { build } from 'esbuild';

const tempDir = path.join(tmpdir(), 'wendao-agent-mood-tests');
const bundlePath = path.join(tempDir, 'agentMood.test-bundle.mjs');

const loadAgentMood = async () => {
  await build({
    entryPoints: [new URL('./agentMood.ts', import.meta.url).pathname],
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

test('resolveAgentMood maps Librarian to a library micro-expression', async () => {
  const { resolveAgentMood } = await loadAgentMood();

  assert.deepEqual(resolveAgentMood({ agentName: 'Librarian', status: 'running' }), {
    caption: '戴上眼镜翻找站内知识',
    key: 'librarian',
    label: 'Librarian 正在查图书馆',
    tone: 'emerald',
  });
});

test('resolveAgentMood maps research and synthesis agents to distinct moods', async () => {
  const { resolveAgentMood } = await loadAgentMood();

  assert.equal(resolveAgentMood({ agentName: 'Journalist' }).key, 'journalist');
  assert.equal(resolveAgentMood({ agentName: 'Synthesizer' }).key, 'synthesizer');
  assert.equal(resolveAgentMood({ agentName: 'planner' }).key, 'planner');
  assert.equal(resolveAgentMood({ agentName: 'executor' }).key, 'executor');
  assert.equal(resolveAgentMood({ agentName: 'replanner' }).key, 'replanner');
});

test('resolveAgentMood celebrates strong matches on completed steps', async () => {
  const { resolveAgentMood } = await loadAgentMood();

  const mood = resolveAgentMood({
    agentName: 'Librarian',
    status: 'completed',
    summary: '站内资料充足，可直接回答',
  });

  assert.equal(mood.key, 'found');
  assert.equal(mood.label, '发现高匹配答案');
});
