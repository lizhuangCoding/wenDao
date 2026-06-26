import assert from 'node:assert/strict';
import { readFile, readdir } from 'node:fs/promises';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import vm from 'node:vm';

const srcRoot = fileURLToPath(new URL('./', import.meta.url));

const loadResources = async () => {
  const source = await readFile(new URL('./i18n.ts', import.meta.url), 'utf8');
  const match = source.match(/const resources = ([\s\S]*?);\n\ni18n/);
  assert.ok(match, 'i18n resources object should be parseable');

  const context = {};
  vm.runInNewContext(`resources = ${match[1]}`, context);
  return context.resources;
};

const flattenKeys = (value, prefix = '', keys = new Set()) => {
  for (const [key, child] of Object.entries(value)) {
    const fullKey = prefix ? `${prefix}.${key}` : key;
    if (child && typeof child === 'object' && !Array.isArray(child)) {
      flattenKeys(child, fullKey, keys);
    } else {
      keys.add(fullKey);
    }
  }
  return keys;
};

const readSourceFiles = async (directory) => {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = await Promise.all(entries.map(async (entry) => {
    const fullPath = path.join(directory, entry.name);
    if (entry.isDirectory()) return readSourceFiles(fullPath);
    return /\.(ts|tsx)$/.test(entry.name) ? [fullPath] : [];
  }));
  return files.flat();
};

test('English and Chinese i18n resources expose the same keys', async () => {
  const resources = await loadResources();
  const englishKeys = flattenKeys(resources.en.translation);
  const chineseKeys = flattenKeys(resources.zh.translation);

  assert.deepEqual([...englishKeys].sort(), [...chineseKeys].sort());
});

test('all static translation calls resolve to known resource keys', async () => {
  const resources = await loadResources();
  const availableKeys = new Set([
    ...flattenKeys(resources.en.translation),
    ...flattenKeys(resources.zh.translation),
  ]);
  const sourceFiles = await readSourceFiles(srcRoot);
  const usedKeys = new Set();

  for (const file of sourceFiles) {
    const source = await readFile(file, 'utf8');
    for (const match of source.matchAll(/\bt\(\s*['"]([^'"]+)['"]/g)) {
      usedKeys.add(match[1]);
    }
  }

  const missingKeys = [...usedKeys].filter((key) => !availableKeys.has(key)).sort();
  assert.deepEqual(missingKeys, []);
});

test('article editor chrome labels are translated in the correct language', async () => {
  const resources = await loadResources();

  assert.equal(resources.en.translation.articleEditor.previewTitle, 'Preview');
  assert.equal(resources.zh.translation.articleEditor.previewTitle, '预览');
  assert.equal(resources.en.translation.articleEditor.scheduledSet, 'Scheduled');
  assert.equal(resources.zh.translation.articleEditor.scheduledSet, '已设置');
  assert.equal(resources.en.translation.nav.openMenu, 'Open menu');
  assert.equal(resources.zh.translation.nav.openMenu, '打开菜单');
});
