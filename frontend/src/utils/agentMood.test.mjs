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
    caption: 'Searching the internal knowledge base',
    key: 'librarian',
    label: 'Librarian is searching the library',
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
  assert.equal(mood.label, 'High-match answer found');
});

test('resolveAgentMood celebrates sufficient local coverage from step detail', async () => {
  const { resolveAgentMood } = await loadAgentMood();

  const mood = resolveAgentMood({
    agentName: 'Librarian',
    detail: '检索完成。找到 3 个相关来源。覆盖状态：sufficient\n\n站内知识摘要：站内资料充足',
    status: 'completed',
    summary: '正在检索站内知识',
  });

  assert.equal(mood.key, 'found');
});

test('selectFeaturedAgentStep keeps a recent strong match visible over later running steps', async () => {
  const { selectFeaturedAgentStep } = await loadAgentMood();

  const step = selectFeaturedAgentStep([
    {
      id: 1,
      agent_name: 'Librarian',
      type: 'thinking',
      summary: '正在检索站内知识',
      detail: '检索完成。找到 2 个相关来源。覆盖状态：sufficient',
      status: 'completed',
      created_at: '2026-05-20T10:00:00Z',
    },
    {
      id: 2,
      agent_name: 'Synthesizer',
      type: 'thinking',
      summary: '正在整合专家结果',
      detail: '',
      status: 'running',
      created_at: '2026-05-20T10:00:01Z',
    },
  ]);

  assert.equal(step?.agent_name, 'Librarian');
});
