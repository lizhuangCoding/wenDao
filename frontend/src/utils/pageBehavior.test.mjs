import assert from 'node:assert/strict';
import { rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { build } from 'esbuild';

const tempDir = path.join(tmpdir(), 'wendao-page-behavior-tests');
const bundlePath = path.join(tempDir, 'pageBehavior.test-bundle.mjs');

const loadPageBehavior = async () => {
  await build({
    entryPoints: [new URL('./pageBehavior.ts', import.meta.url).pathname],
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

test('shouldFetchCurrentUser probes once without token to recover cookie auth', async () => {
  const { shouldFetchCurrentUser } = await loadPageBehavior();

  assert.equal(shouldFetchCurrentUser(null, false), true);
  assert.equal(shouldFetchCurrentUser('', false), true);
  assert.equal(shouldFetchCurrentUser(null, true), false);
  assert.equal(shouldFetchCurrentUser('access-token'), true);
});

test('shouldAttemptTokenRefresh respects silent auth checks', async () => {
  const { shouldAttemptTokenRefresh } = await loadPageBehavior();

  assert.equal(
    shouldAttemptTokenRefresh({
      status: 401,
      url: '/auth/me',
      alreadyRetried: false,
      skipAuthRedirect: true,
    }),
    false
  );
  assert.equal(
    shouldAttemptTokenRefresh({
      status: 401,
      url: '/articles',
      alreadyRetried: false,
      skipAuthRedirect: false,
    }),
    true
  );
});

test('getArticlePrimaryActionLabel matches the submitted article status', async () => {
  const { getArticlePrimaryActionLabel } = await loadPageBehavior();

  assert.equal(getArticlePrimaryActionLabel({ isEdit: false, status: 'draft' }), '保存草稿');
  assert.equal(getArticlePrimaryActionLabel({ isEdit: false, status: 'published' }), '发布文章');
  assert.equal(getArticlePrimaryActionLabel({ isEdit: true, status: 'draft' }), '更新文章');
});
