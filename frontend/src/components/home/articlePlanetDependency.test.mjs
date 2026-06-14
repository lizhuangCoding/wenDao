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

test('article planet keeps a slow horizontal drift animation', async () => {
  const sceneSource = await readProjectFile('src/components/home/ArticlePlanetScene.tsx');

  assert.match(sceneSource, /PLANET_DRIFT_SPEED/);
  assert.match(sceneSource, /PLANET_DESKTOP_DRIFT_AMPLITUDE/);
  assert.match(sceneSource, /PLANET_MOBILE_DRIFT_AMPLITUDE/);
  assert.match(
    sceneSource,
    /clusterRef\.current\.position\.x =[\s\S]*basePosition\[0\] \+ Math\.sin\(clock\.elapsedTime \* PLANET_DRIFT_SPEED\) \* driftAmplitude/
  );
});

test('article planet renders constellation connection lines from layout relationships', async () => {
  const sceneSource = await readProjectFile('src/components/home/ArticlePlanetScene.tsx');

  assert.match(sceneSource, /buildArticlePlanetConnections/);
  assert.match(sceneSource, /buildArticlePlanetGravityLayout/);
  assert.match(sceneSource, /ArticlePlanetConnectionLine/);
  assert.match(sceneSource, /GRAVITY_LINE_LERP_SPEED/);
  assert.match(sceneSource, /<lineBasicMaterial/);
  assert.match(sceneSource, /connections\.map/);
});

test('article planet nodes animate toward gravity targets without replacing the open click action', async () => {
  const [nodeSource, heroSource] = await Promise.all([
    readProjectFile('src/components/home/ArticlePlanetNode.tsx'),
    readProjectFile('src/components/home/ArticlePlanetHero.tsx'),
  ]);

  assert.match(nodeSource, /groupRef\.current\?\.position\.lerp/);
  assert.match(nodeSource, /gravityRole === 'related'/);
  assert.match(nodeSource, /gravityRole === 'dimmed'/);
  assert.match(heroSource, /activeArticleId=\{activeArticleId\}/);
  assert.match(heroSource, /onArticleOpen=\{openArticle\}/);
});
