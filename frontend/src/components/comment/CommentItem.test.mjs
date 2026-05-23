import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const source = await readFile(new URL('./CommentItem.tsx', import.meta.url), 'utf8');

test('CommentItem handles comments whose user relation is missing', () => {
  assert.match(source, /const\s+commentUser\s*=/);
  assert.match(source, /已注销用户/);
  assert.doesNotMatch(source, /comment\.user\.(avatar_url|username)/);
});
