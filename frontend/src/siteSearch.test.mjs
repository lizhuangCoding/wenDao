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

test('search page provides local search history without popular search shortcuts', async () => {
  const source = await loadSearchSource();

  assert.match(source, /SEARCH_HISTORY_KEY\s*=\s*'wendao-search-history'/);
  assert.match(source, /MAX_SEARCH_HISTORY\s*=\s*8/);
  assert.match(source, /localStorage\.getItem\(SEARCH_HISTORY_KEY\)/);
  assert.match(source, /localStorage\.setItem\(SEARCH_HISTORY_KEY/);
  assert.match(source, /saveSearchHistory/);
  assert.match(source, /搜索历史/);
  assert.match(source, /清空历史/);
  assert.doesNotMatch(source, /POPULAR_SEARCH_TERMS/);
  assert.doesNotMatch(source, /热门搜索/);
});
