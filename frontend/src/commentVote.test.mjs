import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadSource = async (path) => readFile(new URL(path, import.meta.url), 'utf8');

test('comment vote state is scoped by current user so multiple accounts do not share one localStorage key', async () => {
  const source = await loadSource('./components/comment/CommentItem.tsx');

  assert.match(source, /useAuthStore/);
  assert.match(source, /const votedKey = `comment_vote_\$\{userId \|\| 'anon'\}_\$\{comment\.id\}`;/);
  assert.doesNotMatch(source, /const votedKey = `comment_vote_\$\{comment\.id\}`;/);
  assert.match(source, /useEffect\(\(\) => \{/);
});
