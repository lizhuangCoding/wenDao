import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import test from 'node:test';

const __dirname = dirname(fileURLToPath(import.meta.url));
const srcRoot = resolve(__dirname, '..');

const filesWithMutationToasts = [
  'components/comment/CommentForm.tsx',
  'views/admin/AIObservability.tsx',
  'views/admin/Settings.tsx',
  'views/admin/articles/ArticleEditor.tsx',
  'views/admin/articles/ArticleList.tsx',
  'views/admin/categories/CategoryList.tsx',
  'views/admin/collections/CollectionList.tsx',
  'views/admin/comments/CommentList.tsx',
  'views/admin/knowledge-documents/KnowledgeDocumentDetail.tsx',
  'views/admin/knowledge-documents/KnowledgeDocumentList.tsx',
  'views/admin/tags/TagList.tsx',
  'views/admin/users/UserManagement.tsx',
  'pages/admin/Broadcast.tsx',
];

test('API mutation error toasts preserve backend error messages before fallback text', () => {
  const violations = [];

  for (const relativePath of filesWithMutationToasts) {
    const source = readFileSync(resolve(srcRoot, relativePath), 'utf8');
    const lines = source.split('\n');

    lines.forEach((line, index) => {
      if (!line.includes('showToast(') || !line.includes("'error'")) return;
      if (!line.includes('onError') && !lines[Math.max(index - 1, 0)].includes('onError')) return;
      if (line.includes('.message')) return;
      violations.push(`${relativePath}:${index + 1}: ${line.trim()}`);
    });
  }

  assert.deepEqual(violations, []);
});
