import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadSource = async (path) => readFile(new URL(path, import.meta.url), 'utf8');

test('notification pages refresh unread count and cached notification lists after read actions', async () => {
  const [listSource, bellSource] = await Promise.all([
    loadSource('./pages/NotificationList.tsx'),
    loadSource('./components/common/NotificationBell.tsx'),
  ]);

  assert.match(listSource, /decrementUnread/);
  assert.match(listSource, /fetchUnreadCount/);
  assert.match(listSource, /invalidateQueries\(\{\s*queryKey:\s*\['notifications'\]/);
  assert.doesNotMatch(listSource, /setUnreadCount\(Math\.max\(0,\s*useNotificationStore\.getState\(\)\.unreadCount - 1\)\)/);

  assert.match(bellSource, /useQueryClient/);
  assert.match(bellSource, /decrementUnread/);
  assert.match(bellSource, /invalidateQueries\(\{\s*queryKey:\s*\['notifications'\]/);
  assert.doesNotMatch(bellSource, /setUnreadCount\(Math\.max\(0,\s*unreadCount - 1\)\)/);
});

test('broadcast page refreshes local notification state after sending a broadcast', async () => {
  const source = await loadSource('./pages/admin/Broadcast.tsx');

  assert.match(source, /useQueryClient/);
  assert.match(source, /fetchUnreadCount/);
  assert.match(source, /invalidateQueries\(\{\s*queryKey:\s*\['notifications'\]/);
});

test('notification list provides page navigation and renders message content elegantly', async () => {
  const source = await loadSource('./pages/NotificationList.tsx');

  assert.match(source, /Layout/);
  assert.match(source, /PageHeader/);
  assert.match(source, /SegmentedControl/);
  assert.doesNotMatch(source, /StatusBadge/);
  assert.match(source, /to="\/"/);
  assert.match(source, /notification\.returnHome/);
  assert.match(source, /ArticleContent/);
  assert.match(source, /notification-message-body/);
  assert.doesNotMatch(source, /to=\{notif\.link_url \|\| '\/notifications'\}/);
  assert.match(source, /comment_like/);
  assert.match(source, /notification\.commentLike/);
  assert.doesNotMatch(source, /comment_dislike/);
  assert.doesNotMatch(source, /StatusBadge variant="neutral"/);
  assert.equal((source.match(/notification\.description/g) ?? []).length, 1);
});

test('notification bell renders markdown content as a readable preview', async () => {
  const source = await loadSource('./components/common/NotificationBell.tsx');

  assert.match(source, /markdownToPlainText/);
  assert.match(source, /getNotificationPreview\(notif\.content\)/);
  assert.match(source, /notification\.commentLike/);
  assert.doesNotMatch(source, /\{notif\.content\}/);
});

test('broadcast editor reuses the article markdown writing studio without a link field', async () => {
  const source = await loadSource('./pages/admin/Broadcast.tsx');

  assert.match(source, /MarkdownWritingStudio/);
  assert.match(source, /allowImageUpload=\{false\}/);
  assert.match(source, /contentStats/);
  assert.doesNotMatch(source, /link_url/);
  assert.doesNotMatch(source, /跳转链接/);
  assert.doesNotMatch(source, /<textarea/);
});
