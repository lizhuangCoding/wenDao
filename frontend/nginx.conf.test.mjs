import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadNginxConfig = () => readFile(new URL('./nginx.conf', import.meta.url), 'utf8');

test('nginx serves SPA html without long-lived cache headers', async () => {
  const config = await loadNginxConfig();

  assert.match(config, /location = \/index\.html \{[\s\S]*Cache-Control "no-cache/);
  assert.match(config, /location \/ \{[\s\S]*try_files \$uri \$uri\/ \/index\.html;[\s\S]*Cache-Control "no-cache/);
});

test('nginx returns missing hashed assets as 404 instead of SPA html', async () => {
  const config = await loadNginxConfig();

  assert.match(config, /location ~\* \\\.\(js\|css\|png[\s\S]*\{[\s\S]*try_files \$uri =404;/);
});
