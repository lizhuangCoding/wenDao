import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const source = await readFile(new URL('./CommentItem.tsx', import.meta.url), 'utf8');

test('CommentItem handles comments whose user relation is missing', () => {
  assert.match(source, /const\s+commentUser\s*=/);
  assert.match(source, /t\('common\.deletedUser'\)/);
  assert.doesNotMatch(source, /comment\.user\.(avatar_url|username)/);
});

test('CommentItem renders a blank avatar placeholder for deleted users', () => {
  assert.match(source, /DefaultDeletedUserAvatar/);
  assert.match(source, /isDeletedUser/);
  assert.doesNotMatch(source, /UserCircle/);
  assert.doesNotMatch(source, /deleted-user/);
});

test('CommentItem lets the active like or dislike vote be cancelled', () => {
  assert.match(source, /type\s+CommentVote\s*=\s*'like'\s*\|\s*'dislike'\s*\|\s*null/);
  assert.match(source, /commentApi\.unlikeComment/);
  assert.match(source, /commentApi\.undislikeComment/);
  assert.doesNotMatch(source, /disabled=\{voted\}/);
  assert.doesNotMatch(source, /if\s*\(\s*voted\s*\)\s*return/);
});
