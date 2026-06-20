import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import test from 'node:test';

const __dirname = dirname(fileURLToPath(import.meta.url));
const source = readFileSync(resolve(__dirname, 'CommentForm.tsx'), 'utf8');

test('CommentForm shows the API error message when comment submission fails', () => {
  assert.match(source, /onError:\s*\(\s*error:\s*any\s*\)\s*=>/);
  assert.match(source, /showToast\(\s*error\.message\s*\|\|\s*t\('common\.failed'\)\s*,\s*'error'\s*\)/);
});
