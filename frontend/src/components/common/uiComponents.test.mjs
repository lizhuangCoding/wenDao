import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadCommonSource = async (name) => {
  return readFile(new URL(`./${name}.tsx`, import.meta.url), 'utf8');
};

const loadCommonIndex = async () => {
  return readFile(new URL('./index.ts', import.meta.url), 'utf8');
};

test('common UI primitives centralize professional surface, form, and action styles', async () => {
  const [button, buttonStyles, panel, pageHeader, formControls, dataTable, statusBadge, toggleSwitch, index] =
    await Promise.all([
      loadCommonSource('Button'),
      readFile(new URL('./buttonStyles.ts', import.meta.url), 'utf8'),
      loadCommonSource('Panel'),
      loadCommonSource('PageHeader'),
      loadCommonSource('FormControls'),
      loadCommonSource('DataTable'),
      loadCommonSource('StatusBadge'),
      loadCommonSource('ToggleSwitch'),
      loadCommonIndex(),
    ]);

  assert.match(button, /getButtonClassName/);
  assert.match(button, /type = 'button'/);
  assert.match(buttonStyles, /getButtonClassName/);
  assert.match(buttonStyles, /dark:/);
  assert.match(panel, /dark:border-neutral-800/);
  assert.match(pageHeader, /text-3xl/);
  assert.match(formControls, /TextInput/);
  assert.match(formControls, /SelectInput/);
  assert.match(formControls, /TextArea/);
  assert.match(formControls, /dark:bg-neutral-900/);
  assert.match(dataTable, /DataTableHeaderCell/);
  assert.match(dataTable, /overflow-x-auto/);
  assert.match(statusBadge, /variantClassName/);
  assert.match(toggleSwitch, /role="switch"/);

  assert.match(index, /export \* from '.\/Button'/);
  assert.match(index, /export \* from '.\/Panel'/);
  assert.match(index, /export \* from '.\/PageHeader'/);
  assert.match(index, /export \* from '.\/FormControls'/);
  assert.match(index, /export \* from '.\/DataTable'/);
  assert.match(index, /export \* from '.\/StatusBadge'/);
  assert.match(index, /export \* from '.\/ToggleSwitch'/);
});
