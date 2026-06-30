import assert from 'node:assert/strict';
import { readFile, rm } from 'node:fs/promises';
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

test('shouldFetchCurrentUser only probes before cookie-backed auth has been resolved', async () => {
  const { shouldFetchCurrentUser } = await loadPageBehavior();

  assert.equal(shouldFetchCurrentUser(false), true);
  assert.equal(shouldFetchCurrentUser(null), true);
  assert.equal(shouldFetchCurrentUser(undefined), true);
  assert.equal(shouldFetchCurrentUser(true), false);
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

test('current user failure does not clear auth created after the request started', async () => {
  const { shouldClearAuthAfterCurrentUserFailure } = await loadPageBehavior();

  assert.equal(shouldClearAuthAfterCurrentUserFailure(0, 1), false);
  assert.equal(shouldClearAuthAfterCurrentUserFailure(3, 4), false);
  assert.equal(shouldClearAuthAfterCurrentUserFailure(0, 0), true);
  assert.equal(shouldClearAuthAfterCurrentUserFailure(7, 7), true);
});

test('current user response only applies to the auth state that requested it', async () => {
  const { shouldApplyCurrentUserResult } = await loadPageBehavior();

  assert.equal(shouldApplyCurrentUserResult(0, 0), true);
  assert.equal(shouldApplyCurrentUserResult(2, 2), true);
  assert.equal(shouldApplyCurrentUserResult(1, 2), false);
  assert.equal(shouldApplyCurrentUserResult(9, 10), false);
});

test('getArticlePrimaryActionLabel matches the submitted article status', async () => {
  const { getArticlePrimaryActionLabel } = await loadPageBehavior();

  assert.equal(getArticlePrimaryActionLabel({ isEdit: false, status: 'draft' }), 'Save Draft');
  assert.equal(getArticlePrimaryActionLabel({ isEdit: false, status: 'published' }), 'Publish Article');
  assert.equal(getArticlePrimaryActionLabel({ isEdit: true, status: 'draft' }), 'Update Article');
});

test('api client attaches CSRF header for unsafe cookie-backed requests', async () => {
  const source = await readFile(new URL('../api/client.ts', import.meta.url), 'utf8');

  assert.match(source, /readCookie\('csrf_token'\)/);
  assert.match(source, /config\.headers\['X-CSRF-Token'\] = csrfToken/);
  assert.match(source, /!\['get', 'head', 'options'\]\.includes\(normalized\)/);
  assert.doesNotMatch(source, /localStorage\.getItem\('access_token'\)/);
  assert.doesNotMatch(source, /Authorization = `Bearer/);
});

test('chat stream fetch requests attach CSRF header for cookie auth fallback', async () => {
  const source = await readFile(new URL('../api/chat.ts', import.meta.url), 'utf8');

  assert.match(source, /readCookie\('csrf_token'\)/);
  assert.match(source, /'X-CSRF-Token': csrfToken/);
  assert.doesNotMatch(source, /localStorage\.getItem\('access_token'\)/);
  assert.doesNotMatch(source, /Authorization: `Bearer/);
});
