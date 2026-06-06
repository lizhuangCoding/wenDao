import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadArticleEditor = () => readFile(new URL('./ArticleEditor.tsx', import.meta.url), 'utf8');

test('ArticleEditor uses a component date-time picker for scheduled publishing', async () => {
  const source = await loadArticleEditor();

  assert.match(source, /DatePicker/);
  assert.match(source, /enableTimePicker/);
  assert.match(source, /valueType="YYYY-MM-DD HH:mm"/);
  assert.match(source, /article-schedule-picker/);
  assert.doesNotMatch(source, /type="datetime-local"/);
  assert.doesNotMatch(source, /webkit-calendar-picker-indicator/);
});
