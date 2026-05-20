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
