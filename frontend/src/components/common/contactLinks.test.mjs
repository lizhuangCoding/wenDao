import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadSourceFile = async (relativePath) => {
  return readFile(new URL(`../${relativePath}`, import.meta.url), 'utf8');
};

test('contact links expose the configured mail and GitHub destinations', async () => {
  const source = await loadSourceFile('common/ContactLinks.tsx');
  const dataSource = await loadSourceFile('common/contactLinksData.ts');

  assert.match(source, /links = defaultContactLinks/);
  assert.match(dataSource, /3174285493@qq\.com/);
  assert.match(dataSource, /github\.com\/lizhuangCoding/);
  assert.match(source, /sort_order/);
});

test('footer composes the shared contact links component', async () => {
  const source = await loadSourceFile('common/Footer.tsx');

  assert.match(source, /ContactLinks/);
  assert.match(source, /common\.contactMe/);
  assert.match(source, /contactLinksData/);
});

test('home does not render a separate about-contact section below articles', async () => {
  const source = await loadSourceFile('../pages/Home.tsx');

  assert.doesNotMatch(source, /HomeContactSection/);
  assert.doesNotMatch(source, /关于我/);
});
