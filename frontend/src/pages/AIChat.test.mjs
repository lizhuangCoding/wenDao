import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadAIChatSource = async () => {
  return readFile(new URL('./AIChat.tsx', import.meta.url), 'utf8');
};

const loadChatComponentSource = async (name) => {
  return readFile(new URL(`../components/chat/${name}.tsx`, import.meta.url), 'utf8');
};

const loadApiSource = async (name) => {
  return readFile(new URL(`../api/${name}.ts`, import.meta.url), 'utf8');
};

const loadStoreSource = async (name) => {
  return readFile(new URL(`../store/${name}.ts`, import.meta.url), 'utf8');
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
  assert.match(source, /e\.shiftKey/);
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
  const moodSource = await loadChatComponentSource('AgentMoodIndicator');
  const stageSource = await loadChatComponentSource('ChatStageBanner');
  const messageSource = await loadChatComponentSource('ChatMessageList');

  assert.match(source, /AIProcessingHalo/);
  assert.match(stageSource, /AgentMoodIndicator/);
  assert.match(messageSource, /AgentMoodIndicator/);
  assert.match(moodSource, /useReducedMotion/);
  assert.match(moodSource, /scale:\s*prefersReducedMotion \? 1 : \[1, 1\.04, 1\]/);
  assert.match(source, /featuredAgentStep/);
  assert.match(source, /detail={featuredAgentStep\?\.detail}/);
  assert.match(source, /currentStage/);
  assert.doesNotMatch(source, /animate-pulse' : 'bg-neutral-400'/);
});

test('AIChat keeps assistant answer bubbles clean while user bubbles keep depth', async () => {
  const messageSource = await loadChatComponentSource('ChatMessageList');

  assert.match(messageSource, /bg-gradient-to-br/);
  assert.match(messageSource, /from-neutral-950/);
  assert.match(messageSource, /bg-white text-neutral-800 shadow-sm/);
  assert.match(messageSource, /dark:bg-\[#07111a\]/);
  assert.doesNotMatch(messageSource, /from-primary-50\/90/);
  assert.doesNotMatch(messageSource, /dark:from-primary-950\/30/);
});

test('AIChat starter prompt cards use restrained magnetic CTA motion', async () => {
  const starterSource = await loadChatComponentSource('StarterPrompts');

  assert.match(starterSource, /motion/);
  assert.match(starterSource, /useMotionValue/);
  assert.match(starterSource, /handleMagneticPointerMove/);
  assert.match(starterSource, /whileHover=\{\{ scale: 1\.01 \}\}/);
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

test('AIChat share and export paths use store actions and safe filenames', async () => {
  const [source, apiSource, storeSource] = await Promise.all([
    loadAIChatSource(),
    loadApiSource('chat'),
    loadStoreSource('chatStore'),
  ]);

  assert.match(storeSource, /updateConversationShare/);
  assert.match(source, /updateConversationShare/);
  assert.doesNotMatch(source, /\(useChatStore\.getState\(\) as any\)\.conversations/);
  assert.match(apiSource, /sanitizeConversationExportTitle/);
  assert.match(apiSource, /const safeTitle = sanitizeConversationExportTitle\(title\)/);
  assert.doesNotMatch(apiSource, /\\\*/);
  assert.doesNotMatch(apiSource, /\\\?/);
  assert.doesNotMatch(apiSource, /\\\|/);
});

test('ModelSelector exposes full long model names instead of truncating menu items', async () => {
  const source = await loadChatComponentSource('ModelSelector');

  assert.match(source, /title=\{currentLabel \|\| t\('common\.defaultModel'\)\}/);
  assert.match(source, /w-\[min\(92vw,28rem\)\]/);
  assert.match(source, /whitespace-normal/);
  assert.match(source, /break-words/);
  assert.match(source, /title=\{m\.display_name\}/);
  assert.doesNotMatch(source, /<span className="flex-1 text-left truncate">/);
});
