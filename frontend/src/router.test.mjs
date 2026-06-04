import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadRouterSource = async () => {
  return readFile(new URL('./router.tsx', import.meta.url), 'utf8');
};

const loadRouteSuspenseBoundarySource = async () => {
  return readFile(new URL('./components/common/RouteSuspenseBoundary.tsx', import.meta.url), 'utf8');
};

test('router provides a recovery boundary for failed lazy route chunks', async () => {
  const source = await loadRouterSource();

  assert.match(source, /RouteErrorFallback/);
  assert.match(source, /RouteSuspenseBoundary/);
  assert.match(source, /errorElement:\s*<RouteErrorFallback\s*\/>/);
  assert.match(source, /<RouteSuspenseBoundary>\s*\{element\}\s*<\/RouteSuspenseBoundary>/);
});

test('route suspense boundary resets stale lazy content when the path changes', async () => {
  const source = await loadRouteSuspenseBoundarySource();

  assert.match(source, /useLocation/);
  assert.match(source, /key=\{location\.pathname\}/);
  assert.match(source, /fallback=\{<Loading\s*\/>\}/);
});

test('admin lazy child routes keep the admin shell mounted while children load', async () => {
  const source = await loadRouterSource();

  assert.match(source, /path:\s*'stats',\s*element:\s*withSuspense\(<Dashboard\s*\/>\)/);
  assert.match(source, /path:\s*'articles',\s*element:\s*withSuspense\(<ArticleList\s*\/>\)/);
  assert.match(source, /path:\s*'articles\/new',\s*element:\s*withSuspense\(<ArticleEditor\s*\/>\)/);
  assert.match(source, /path:\s*'categories',\s*element:\s*withSuspense\(<CategoryList\s*\/>\)/);
  assert.match(source, /path:\s*'comments',\s*element:\s*withSuspense\(<CommentList\s*\/>\)/);
});

test('router renders unknown routes through a shared not-found page', async () => {
  const source = await loadRouterSource();

  assert.match(source, /NotFoundPage/);
  assert.doesNotMatch(source, /<a href="\/"/);
  assert.doesNotMatch(source, /text-neutral-700 mb-4/);
});
