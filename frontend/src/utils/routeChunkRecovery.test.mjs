import assert from 'node:assert/strict';
import { rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { build } from 'esbuild';

const tempDir = path.join(tmpdir(), 'wendao-route-chunk-recovery-tests');
const bundlePath = path.join(tempDir, 'routeChunkRecovery.test-bundle.mjs');

const loadRouteChunkRecovery = async () => {
  await build({
    entryPoints: [new URL('./routeChunkRecovery.ts', import.meta.url).pathname],
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

const createMemoryStorage = () => {
  const values = new Map();

  return {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value),
    removeItem: (key) => values.delete(key),
  };
};

test('route chunk recovery recognises mobile module script import failures', async () => {
  const { isRouteModuleImportError } = await loadRouteChunkRecovery();

  assert.equal(isRouteModuleImportError(new Error('Importing a module script failed.')), true);
  assert.equal(isRouteModuleImportError(new Error('Failed to fetch dynamically imported module')), true);
  assert.equal(isRouteModuleImportError(new Error('Unable to preload CSS for /assets/Profile.css')), true);
  assert.equal(isRouteModuleImportError(new Error('Cannot read properties of undefined')), false);
});

test('route chunk recovery reloads each failing path at most once', async () => {
  const { clearRouteChunkReloadAttempt, shouldAttemptRouteChunkReload } = await loadRouteChunkRecovery();
  const storage = createMemoryStorage();
  const error = new Error('Importing a module script failed.');

  assert.equal(shouldAttemptRouteChunkReload(error, '/ai-chat', storage), true);
  assert.equal(shouldAttemptRouteChunkReload(error, '/ai-chat', storage), false);
  assert.equal(shouldAttemptRouteChunkReload(error, '/profile', storage), true);

  clearRouteChunkReloadAttempt('/ai-chat', storage);
  assert.equal(shouldAttemptRouteChunkReload(error, '/ai-chat', storage), true);
});
