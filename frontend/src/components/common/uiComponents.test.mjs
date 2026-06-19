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
  const [button, buttonStyles, panel, pageShell, pageHeader, formControls, dataTable, statusBadge, toggleSwitch, comet, index] =
    await Promise.all([
      loadCommonSource('Button'),
      readFile(new URL('./buttonStyles.ts', import.meta.url), 'utf8'),
      loadCommonSource('Panel'),
      loadCommonSource('PageShell'),
      loadCommonSource('PageHeader'),
      loadCommonSource('FormControls'),
      loadCommonSource('DataTable'),
      loadCommonSource('StatusBadge'),
      loadCommonSource('ToggleSwitch'),
      loadCommonSource('CursorCometTrail'),
      loadCommonIndex(),
    ]);

  assert.match(button, /getButtonClassName/);
  assert.match(button, /type = 'button'/);
  assert.match(buttonStyles, /getButtonClassName/);
  assert.match(buttonStyles, /dark:/);
  assert.match(panel, /dark:border-neutral-800/);
  assert.match(panel, /variantClassName/);
  assert.match(pageShell, /max-w-6xl/);
  assert.match(pageShell, /PageShell/);
  assert.match(pageHeader, /text-3xl/);
  assert.match(pageHeader, /eyebrow/);
  assert.match(formControls, /TextInput/);
  assert.match(formControls, /SelectInput/);
  assert.match(formControls, /TextArea/);
  assert.match(formControls, /dark:bg-neutral-900/);
  assert.match(dataTable, /DataTableHeaderCell/);
  assert.match(dataTable, /overflow-x-auto/);
  assert.match(statusBadge, /variantClassName/);
  assert.match(toggleSwitch, /role="switch"/);
  assert.match(comet, /requestAnimationFrame/);
  assert.match(comet, /cancelAnimationFrame/);
  assert.match(comet, /pointer-events-none/);
  assert.match(comet, /\(hover: hover\) and \(pointer: fine\)/);
  assert.match(comet, /prefers-reduced-motion: reduce/);
  assert.match(comet, /getContext\('2d'\)/);
  assert.match(comet, /createLinearGradient/);
  assert.match(comet, /createRadialGradient/);
  assert.match(comet, /globalCompositeOperation = 'lighter'/);
  assert.match(comet, /MAX_TRAIL_POINTS/);
  assert.match(comet, /data-cursor-comet-trail/);
  assert.match(comet, /z-\[45\]/);
  assert.doesNotMatch(comet, /Array\.from\(\{ length/);

  assert.match(index, /export \* from '.\/Button'/);
  assert.match(index, /export \* from '.\/Panel'/);
  assert.match(index, /export \* from '.\/PageShell'/);
  assert.match(index, /export \* from '.\/PageHeader'/);
  assert.match(index, /export \* from '.\/FormControls'/);
  assert.match(index, /export \* from '.\/DataTable'/);
  assert.match(index, /export \* from '.\/StatusBadge'/);
  assert.match(index, /export \* from '.\/ToggleSwitch'/);
  assert.match(index, /export \* from '.\/CursorCometTrail'/);
});
