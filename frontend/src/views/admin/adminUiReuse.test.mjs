import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadAdminSource = async (relativePath) => {
  return readFile(new URL(`./${relativePath}`, import.meta.url), 'utf8');
};

const countMatches = (source, pattern) => source.match(pattern)?.length || 0;

test('admin list pages reuse shared layout, form, table, and status primitives', async () => {
  const sources = await Promise.all([
    loadAdminSource('articles/ArticleList.tsx'),
    loadAdminSource('categories/CategoryList.tsx'),
    loadAdminSource('comments/CommentList.tsx'),
    loadAdminSource('knowledge-documents/KnowledgeDocumentList.tsx'),
  ]);

  for (const source of sources) {
    assert.match(source, /PageHeader/);
    assert.match(source, /Panel/);
    assert.match(source, /DataTable/);
    assert.match(source, /DataTableHeaderCell/);
    assert.match(source, /DataTableCell/);
  }

  const searchableSources = [sources[0], sources[2], sources[3]];
  for (const source of searchableSources) {
    assert.match(source, /TextInput/);
    assert.match(source, /SelectInput/);
    assert.match(source, /Button/);
  }

  assert.match(sources[0], /StatusBadge/);
  assert.match(sources[0], /ToggleSwitch/);
  assert.match(sources[2], /StatusBadge/);
  assert.match(sources[3], /StatusBadge/);

  const combined = sources.join('\n');
  assert.equal(countMatches(combined, /rounded-2xl border border-neutral-(?:100|200) bg-white/g), 0);
  assert.ok(countMatches(combined, /px-6 py-4 text-sm font-semibold text-neutral-600/g) < 3);
});

test('admin dashboard and knowledge document detail avoid rough browser defaults', async () => {
  const [dashboard, detail] = await Promise.all([
    loadAdminSource('Dashboard.tsx'),
    loadAdminSource('knowledge-documents/KnowledgeDocumentDetail.tsx'),
  ]);

  assert.match(dashboard, /PageHeader/);
  assert.match(dashboard, /Panel/);
  assert.match(dashboard, /SegmentedControl/);
  assert.doesNotMatch(dashboard, /alert\(/);

  assert.match(detail, /PageHeader/);
  assert.match(detail, /Panel/);
  assert.match(detail, /Button/);
  assert.match(detail, /TextArea/);
  assert.match(detail, /StatusBadge/);
});

test('admin settings and category management expose configurable site and category ordering', async () => {
  const [settings, categories] = await Promise.all([
    loadAdminSource('Settings.tsx'),
    loadAdminSource('categories/CategoryList.tsx'),
  ]);

  assert.match(settings, /contactLinksTitle/);
  assert.match(settings, /saveContactLinks/);
  assert.match(settings, /contactLinksInput/);

  assert.match(categories, /sort_order/);
  assert.match(categories, /sortOrderHint/);
  assert.match(categories, /admin\.sortOrder/);
});

test('admin comments default to normal comments while keeping all-status access', async () => {
  const comments = await loadAdminSource('comments/CommentList.tsx');

  assert.match(comments, /useState<CommentStatusFilter>\('normal'\)/);
  assert.match(comments, /setStatus\('normal'\)/);
  assert.match(comments, /<option value="">\{t\('admin\.allStatus'\)\}<\/option>/);
});

test('admin data tables protect utility columns from long primary text', async () => {
  const [dataTable, articles, categories, comments, documents, users, collections, aiObservability] = await Promise.all([
    readFile(new URL('../../components/common/DataTable.tsx', import.meta.url), 'utf8'),
    loadAdminSource('articles/ArticleList.tsx'),
    loadAdminSource('categories/CategoryList.tsx'),
    loadAdminSource('comments/CommentList.tsx'),
    loadAdminSource('knowledge-documents/KnowledgeDocumentList.tsx'),
    loadAdminSource('users/UserManagement.tsx'),
    loadAdminSource('collections/CollectionList.tsx'),
    loadAdminSource('AIObservability.tsx'),
  ]);

  assert.match(dataTable, /table-fixed/);
  assert.match(dataTable, /layout\?: 'fixed' \| 'auto'/);
  assert.match(dataTable, /minWidth\?: string/);
  assert.match(dataTable, /stretch\?: boolean/);
  assert.match(dataTable, /width\?:/);
  assert.match(dataTable, /nowrap\?:/);
  assert.match(dataTable, /truncate\?:/);

  assert.match(dataTable, /actionsCompact/);
  assert.match(dataTable, /actionsWide/);
  assert.match(articles, /width="actionsWide"/);
  for (const source of [comments, documents, aiObservability]) {
    assert.match(source, /width="actionsCompact"/);
  }
  for (const source of [categories, collections, users]) {
    assert.match(source, /width="actions"/);
  }

  for (const source of [articles, categories, comments, documents]) {
    assert.match(source, /width="select"/);
  }

  for (const source of [articles, categories, comments, documents, users]) {
    assert.match(source, /truncate/);
  }

  for (const source of [categories, collections, aiObservability]) {
    assert.match(source, /minWidth="/);
  }

  assert.doesNotMatch(collections, /layout="auto"/);
  assert.match(collections, /stretch=\{false\}/);
  assert.match(categories, /stretch=\{false\}/);
  assert.match(collections, /<DataTableHeaderCell width="medium">名称<\/DataTableHeaderCell>/);
  assert.match(collections, /<DataTableHeaderCell width="compact">创建时间<\/DataTableHeaderCell>/);
  assert.match(collections, /<DataTableHeaderCell width="actions">操作<\/DataTableHeaderCell>/);
  assert.doesNotMatch(collections, /DataTableCell align="right" nowrap/);
  assert.match(categories, /<DataTableHeaderCell width="medium">\{t\('admin\.name'\)\}<\/DataTableHeaderCell>/);
  assert.doesNotMatch(categories, /DataTableCell align="right" nowrap/);
});
