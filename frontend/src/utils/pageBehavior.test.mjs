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

test('current user failure does not clear auth created after the request started', async () => {
  const { shouldClearAuthAfterCurrentUserFailure } = await loadPageBehavior();

  assert.equal(shouldClearAuthAfterCurrentUserFailure(null, 'new-login-token'), false);
  assert.equal(shouldClearAuthAfterCurrentUserFailure('old-token', 'new-login-token'), false);
  assert.equal(shouldClearAuthAfterCurrentUserFailure(null, null), true);
  assert.equal(shouldClearAuthAfterCurrentUserFailure('expired-token', 'expired-token'), true);
});

test('current user response only applies to the auth state that requested it', async () => {
  const { shouldApplyCurrentUserResult } = await loadPageBehavior();

  assert.equal(shouldApplyCurrentUserResult(null, null), true);
  assert.equal(shouldApplyCurrentUserResult('same-token', 'same-token'), true);
  assert.equal(shouldApplyCurrentUserResult(null, 'new-login-token'), false);
  assert.equal(shouldApplyCurrentUserResult('old-token', 'new-login-token'), false);
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
});
