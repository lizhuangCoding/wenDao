import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadRouterSource = async () => {
  return readFile(new URL('./router.tsx', import.meta.url), 'utf8');
};

test('router provides a recovery boundary for failed lazy route chunks', async () => {
  const source = await loadRouterSource();

  assert.match(source, /RouteErrorFallback/);
  assert.match(source, /RouteLoadSuccessMarker/);
  assert.match(source, /errorElement:\s*<RouteErrorFallback\s*\/>/);
  assert.match(source, /<RouteLoadSuccessMarker>\s*\{element\}\s*<\/RouteLoadSuccessMarker>/);
});
