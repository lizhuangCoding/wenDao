import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadSourceFile = async (relativePath) => {
  return readFile(new URL(`../${relativePath}`, import.meta.url), 'utf8');
};

test('contact links expose the configured mail and GitHub destinations', async () => {
  const source = await loadSourceFile('common/ContactLinks.tsx');

  assert.match(source, /3174285493@qq\.com/);
  assert.match(source, /github\.com\/lizhuangCoding/);
  assert.match(source, /mailto:3174285493@qq\.com/);
  assert.match(source, /https:\/\/github\.com\/lizhuangCoding/);
  assert.match(source, /Mail/);
  assert.match(source, /GitHub/);
});

test('footer composes the shared contact links component', async () => {
  const source = await loadSourceFile('common/Footer.tsx');

  assert.match(source, /ContactLinks/);
  assert.match(source, /联系我/);
});

test('home places the about-contact section after article pagination', async () => {
  const source = await loadSourceFile('../pages/Home.tsx');

  assert.match(source, /HomeContactSection/);
  assert.match(source, /Pagination[\s\S]*HomeContactSection/);
});
