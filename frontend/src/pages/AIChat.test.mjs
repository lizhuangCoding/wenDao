import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadAIChatSource = async () => {
  return readFile(new URL('./AIChat.tsx', import.meta.url), 'utf8');
};

const countMatches = (source, pattern) => source.match(pattern)?.length || 0;

test('AIChat keeps a single desktop chat history collapse control', async () => {
  const source = await loadAIChatSource();

  assert.equal(countMatches(source, /data-chat-history-toggle=/g), 1);
  assert.match(source, /data-chat-history-toggle="sidebar"/);
  assert.doesNotMatch(source, /data-chat-history-toggle="header"/);
});

test('AIChat includes question navigator and multiline composer affordances', async () => {
  const source = await loadAIChatSource();

  assert.match(source, /ChatQuestionNavigator/);
  assert.match(source, /buildChatQuestionNavItems/);
  assert.match(source, /scrollToQuestion/);
  assert.match(source, /<textarea/);
  assert.match(source, /Shift/);
  assert.match(source, /scrollToBottom/);
});

test('AIChat empty state exposes starter prompts', async () => {
  const source = await loadAIChatSource();

  assert.match(source, /STARTER_PROMPTS/);
  assert.match(source, /K8s/);
  assert.match(source, /分布式系统/);
  assert.match(source, /handleStarterPromptSelect/);
  assert.match(source, /starterPrompt\.prompt/);
});

test('AIChat message flow uses layout animation and smooth bottom following', async () => {
  const source = await loadAIChatSource();

  assert.match(source, /layout="position"/);
  assert.match(source, /behavior:\s*'smooth'/);
  assert.match(source, /requestAnimationFrame/);
  assert.doesNotMatch(source, /scrollTop\s*=\s*container\.scrollHeight/);
});

test('AIChat uses expressive agent status indicators while processing', async () => {
  const source = await loadAIChatSource();

  assert.match(source, /AgentMoodIndicator/);
  assert.match(source, /AIProcessingHalo/);
  assert.match(source, /activeAgentStep/);
  assert.match(source, /currentStage/);
  assert.doesNotMatch(source, /animate-pulse' : 'bg-neutral-400'/);
});
