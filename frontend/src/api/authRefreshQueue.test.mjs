import assert from 'node:assert/strict';
import { rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { build } from 'esbuild';

const tempDir = path.join(tmpdir(), 'wendao-auth-refresh-queue-tests');
const bundlePath = path.join(tempDir, 'authRefreshQueue.test-bundle.mjs');

const loadAuthRefreshQueue = async () => {
  await build({
    entryPoints: [new URL('./authRefreshQueue.ts', import.meta.url).pathname],
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

test('auth refresh queue replays all pending requests after refresh succeeds', async () => {
  const { createAuthRefreshQueue } = await loadAuthRefreshQueue();
  const queue = createAuthRefreshQueue();

  const first = new Promise((resolve, reject) => {
    queue.add(() => resolve('first retried'), reject);
  });
  const second = new Promise((resolve, reject) => {
    queue.add(() => resolve('second retried'), reject);
  });

  assert.equal(queue.size(), 2);

  queue.resolveAll();

  assert.deepEqual(await Promise.all([first, second]), ['first retried', 'second retried']);
  assert.equal(queue.size(), 0);
});

test('auth refresh queue rejects all pending requests after refresh fails', async () => {
  const { createAuthRefreshQueue } = await loadAuthRefreshQueue();
  const queue = createAuthRefreshQueue();
  const refreshError = new Error('refresh expired');

  const first = new Promise((resolve, reject) => {
    queue.add(() => resolve('first retried'), reject);
  });
  const second = new Promise((resolve, reject) => {
    queue.add(() => resolve('second retried'), reject);
  });

  assert.equal(queue.size(), 2);

  queue.rejectAll(refreshError);

  await assert.rejects(first, (error) => error === refreshError);
  await assert.rejects(second, (error) => error === refreshError);
  assert.equal(queue.size(), 0);
});
