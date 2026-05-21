import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadAIChatSource = async () => {
  return readFile(new URL('./AIChat.tsx', import.meta.url), 'utf8');
};

const loadChatComponentSource = async (name) => {
  return readFile(new URL(`../components/chat/${name}.tsx`, import.meta.url), 'utf8');
};

const countMatches = (source, pattern) => source.match(pattern)?.length || 0;

test('AIChat keeps a single desktop chat history collapse control', async () => {
  const source = await loadChatComponentSource('ChatHistorySidebar');

  assert.equal(countMatches(source, /data-chat-history-toggle=/g), 1);
  assert.match(source, /data-chat-history-toggle="sidebar"/);
  assert.doesNotMatch(source, /data-chat-history-toggle="header"/);
});

test('AIChat includes question navigator and multiline composer affordances', async () => {
  const source = await loadAIChatSource();
  const composerSource = await loadChatComponentSource('ChatComposer');

  assert.match(source, /ChatQuestionNavigator/);
  assert.match(source, /buildChatQuestionNavItems/);
  assert.match(source, /scrollToQuestion/);
  assert.match(source, /ChatComposer/);
  assert.match(composerSource, /<textarea/);
  assert.match(composerSource, /Shift/);
  assert.match(source, /scrollToBottom/);
});

test('AIChat empty state exposes starter prompts', async () => {
  const source = await loadAIChatSource();

  assert.match(source, /StarterPrompts/);
  assert.doesNotMatch(source, /const STARTER_PROMPTS/);
  assert.doesNotMatch(source, /调研 K8s/);
  assert.doesNotMatch(source, /分布式系统/);
  assert.match(source, /handleStarterPromptSelect/);
});

test('AIChat message flow uses layout animation and smooth bottom following', async () => {
  const source = await loadAIChatSource();
  const messageSource = await loadChatComponentSource('ChatMessageList');

  assert.match(messageSource, /layout="position"/);
  assert.match(source, /behavior:\s*'smooth'/);
  assert.match(source, /requestAnimationFrame/);
  assert.doesNotMatch(source, /scrollTop\s*=\s*container\.scrollHeight/);
});

test('AIChat uses expressive agent status indicators while processing', async () => {
  const source = await loadAIChatSource();
  const stageSource = await loadChatComponentSource('ChatStageBanner');
  const messageSource = await loadChatComponentSource('ChatMessageList');

  assert.match(source, /AIProcessingHalo/);
  assert.match(stageSource, /AgentMoodIndicator/);
  assert.match(messageSource, /AgentMoodIndicator/);
  assert.match(source, /featuredAgentStep/);
  assert.match(source, /detail={featuredAgentStep\?\.detail}/);
  assert.match(source, /currentStage/);
  assert.doesNotMatch(source, /animate-pulse' : 'bg-neutral-400'/);
});

test('AIChat delegates complex chat sections to focused modules', async () => {
  const source = await loadAIChatSource();
  const messageSource = await loadChatComponentSource('ChatMessageList');

  assert.match(source, /ChatComposer/);
  assert.match(source, /ChatHistorySidebar/);
  assert.match(source, /ChatMessageList/);
  assert.match(source, /ChatStageBanner/);
  assert.match(source, /useProcessingTimer/);
  assert.match(messageSource, /AgentProcessPanel/);
  assert.match(messageSource, /ArticleReferencesPanel/);
  assert.match(messageSource, /parseChatArticleReferences/);
  assert.doesNotMatch(source, /const AgentProcessPanel =/);
  assert.doesNotMatch(source, /const ArticleReferencesPanel =/);
  assert.doesNotMatch(source, /<textarea/);
  assert.doesNotMatch(source, /formatProcessingDuration/);
  assert.doesNotMatch(source, /messages\.map/);
});
