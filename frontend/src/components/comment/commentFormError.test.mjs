import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import test from 'node:test';

const __dirname = dirname(fileURLToPath(import.meta.url));
const source = readFileSync(resolve(__dirname, 'CommentForm.tsx'), 'utf8');

test('CommentForm shows the API error message when comment submission fails', () => {
  assert.doesNotMatch(source, /onError:\s*\(\s*error:\s*any\s*\)\s*=>/);
  assert.match(source, /getApiErrorMessage/);
  assert.match(source, /showToast\(\s*getApiErrorMessage\(error,\s*t\('common\.failed'\)\)\s*,\s*'error'\s*\)/);
});
