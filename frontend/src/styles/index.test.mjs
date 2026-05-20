import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadIndexCss = async () => {
  return readFile(new URL('./index.css', import.meta.url), 'utf8');
};

const getRuleBody = (css, selector) => {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const match = css.match(new RegExp(`${escapedSelector}\\s*\\{([^}]*)\\}`, 'm'));
  return match?.[1] || '';
};

test('public article blockquotes keep body-sized text', async () => {
  const css = await loadIndexCss();
  const ruleBody = getRuleBody(css, '.prose-refined blockquote');

  assert.match(ruleBody, /text-base/);
  assert.doesNotMatch(ruleBody, /text-2xl|text-xl/);
});
