import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadSourceFile = async (relativePath) => {
  return readFile(new URL(`./${relativePath}`, import.meta.url), 'utf8');
};

test('shared surfaces use visible light and dark borders', async () => {
  const [panel, dataTable, adminLayout] = await Promise.all([
    loadSourceFile('components/common/Panel.tsx'),
    loadSourceFile('components/common/DataTable.tsx'),
    loadSourceFile('components/admin/AdminLayout.tsx'),
  ]);

  assert.match(panel, /border-neutral-200/);
  assert.match(panel, /dark:border-neutral-700/);
  assert.doesNotMatch(panel, /default: 'border-neutral-100/);
  assert.match(dataTable, /border-neutral-200/);
  assert.match(dataTable, /divide-neutral-200/);
  assert.match(dataTable, /dark:divide-neutral-700/);
  assert.match(adminLayout, /border-neutral-200/);
  assert.match(adminLayout, /dark:border-neutral-700/);
});

test('navigation and pagination keep readable action contrast in both themes', async () => {
  const [header, pagination] = await Promise.all([
    loadSourceFile('components/common/Header.tsx'),
    loadSourceFile('components/common/Pagination.tsx'),
  ]);

  assert.match(header, /dark:hover:bg-primary-500/);
  assert.match(header, /dark:hover:text-white/);
  assert.doesNotMatch(header, /dark:border-primary-900\/20/);
  assert.match(pagination, /border-neutral-200/);
  assert.match(pagination, /dark:text-neutral-200/);
  assert.match(pagination, /dark:border-neutral-700/);
});
