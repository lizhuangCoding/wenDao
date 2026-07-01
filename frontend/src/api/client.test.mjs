import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadClientSource = () => readFile(new URL('./client.ts', import.meta.url), 'utf8');
const loadTypesSource = () => readFile(new URL('../types/index.ts', import.meta.url), 'utf8');
const loadApiTypesSource = () => readFile(new URL('../types/api.ts', import.meta.url), 'utf8');

test('api client releases queued auth retries when token refresh resolves or fails', async () => {
  const source = await loadClientSource();

  assert.match(source, /createAuthRefreshQueue/);
  assert.match(source, /refreshQueue\.add/);
  assert.match(source, /new Promise\(\(resolve, reject\)/);
  assert.match(source, /refreshQueue\.resolveAll\(\)/);
  assert.match(source, /refreshQueue\.rejectAll\(refreshError\)/);
  assert.doesNotMatch(source, /let requests:\s*\(\(\)\s*=>\s*void\)\[\]/);
});

test('api request helpers default to unknown instead of any', async () => {
  const [source, typesSource, apiTypesSource] = await Promise.all([
    loadClientSource(),
    loadTypesSource(),
    loadApiTypesSource(),
  ]);

  assert.match(source, /get:\s*<T = unknown>/);
  assert.match(source, /post:\s*<T = unknown,\s*D = unknown>/);
  assert.match(source, /put:\s*<T = unknown,\s*D = unknown>/);
  assert.match(source, /patch:\s*<T = unknown,\s*D = unknown>/);
  assert.match(typesSource, /export \* from '\.\/api'/);
  assert.match(apiTypesSource, /export interface ApiResponse<T = unknown>/);
  assert.match(source, /normalizeApiError/);
  assert.doesNotMatch(source, /<T = any>/);
  assert.doesNotMatch(apiTypesSource, /ApiResponse<T = any>/);
  assert.doesNotMatch(source, /data\?: any/);
});
