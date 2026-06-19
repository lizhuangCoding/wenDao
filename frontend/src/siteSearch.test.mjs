import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadHomeSource = async () => {
  return readFile(new URL('./pages/Home.tsx', import.meta.url), 'utf8');
};

const loadSearchSource = async () => {
  return readFile(new URL('./pages/Search.tsx', import.meta.url), 'utf8');
};

test('home search form navigates to the dedicated search page with q param', async () => {
  const source = await loadHomeSource();

  assert.match(source, /useNavigate/);
  assert.match(source, /navigate\(`\/search\?q=\$\{encodeURIComponent\(keyword\)\}`\)/);
  assert.doesNotMatch(source, /setSearchKeyword/);
});

test('search page reads URL filters and calls the search API', async () => {
  const source = await loadSearchSource();

  assert.match(source, /useSearchParams/);
  assert.match(source, /searchApi\.searchArticles/);
  assert.match(source, /category_id:\s*categoryID/);
  assert.match(source, /tag_id:\s*tagID/);
  assert.match(source, /dangerouslySetInnerHTML/);
});
