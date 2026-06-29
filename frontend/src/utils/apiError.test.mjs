import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadSource = () => readFile(new URL('./apiError.ts', import.meta.url), 'utf8');

test('normalizeApiError centralizes axios and generic error normalization', async () => {
  const source = await loadSource();

  assert.match(source, /axios\.isAxiosError/);
  assert.match(source, /ApiResponse<unknown>/);
  assert.match(source, /error instanceof Error/);
  assert.match(source, /export const normalizeApiError/);
  assert.match(source, /export const getApiErrorMessage/);
});
