import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadClientSource = () => readFile(new URL('./client.ts', import.meta.url), 'utf8');

test('api client releases queued auth retries when token refresh resolves or fails', async () => {
  const source = await loadClientSource();

  assert.match(source, /createAuthRefreshQueue/);
  assert.match(source, /refreshQueue\.add/);
  assert.match(source, /new Promise\(\(resolve, reject\)/);
  assert.match(source, /refreshQueue\.resolveAll\(\)/);
  assert.match(source, /refreshQueue\.rejectAll\(refreshError\)/);
  assert.doesNotMatch(source, /let requests:\s*\(\(\)\s*=>\s*void\)\[\]/);
});
