import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadSource = async (relativePath) => {
  return readFile(new URL(`../../${relativePath}`, import.meta.url), 'utf8');
};

test('visible dropdowns use the shared custom SelectInput instead of native selects', async () => {
  const [pagination, collectionList] = await Promise.all([
    loadSource('components/common/Pagination.tsx'),
    loadSource('views/admin/collections/CollectionList.tsx'),
  ]);

  assert.match(pagination, /SelectInput/);
  assert.doesNotMatch(pagination, /<select\b/);
  assert.match(collectionList, /SelectInput/);
  assert.doesNotMatch(collectionList, /<select\b/);
});
