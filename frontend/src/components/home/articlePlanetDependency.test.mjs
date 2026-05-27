import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const readProjectFile = (filePath) => readFile(new URL(`../../../${filePath}`, import.meta.url), 'utf8');

test('article planet scene avoids the heavy drei dependency surface', async () => {
  const [packageJson, sceneSource, viteConfig] = await Promise.all([
    readProjectFile('package.json'),
    readProjectFile('src/components/home/ArticlePlanetScene.tsx'),
    readProjectFile('vite.config.ts'),
  ]);

  assert.equal(packageJson.includes('@react-three/drei'), false);
  assert.equal(sceneSource.includes('@react-three/drei'), false);
  assert.equal(viteConfig.includes('@react-three/drei'), false);
});

test('article planet self-rotates without camera auto orbit drift', async () => {
  const sceneSource = await readProjectFile('src/components/home/ArticlePlanetScene.tsx');

  assert.match(sceneSource, /PLANET_SELF_ROTATION_SPEED/);
  assert.match(sceneSource, /spinRef\.current\.rotation\.y \+= delta \* PLANET_SELF_ROTATION_SPEED/);
  assert.match(sceneSource, /controls\.autoRotate = false/);
});
