import { useMemo } from 'react';
import { Canvas } from '@react-three/fiber';
import { AdaptiveDpr, OrbitControls, Preload, Stars } from '@react-three/drei';
import type { ArticleOrbitItem } from '@/types';
import { ArticlePlanetNode } from './ArticlePlanetNode';
import { buildArticlePlanetLayout } from './articlePlanetLayout';

interface ArticlePlanetSceneProps {
  activeArticleId?: number;
  articles: ArticleOrbitItem[];
  onArticleFocus: (article: ArticleOrbitItem) => void;
  onArticleOpen: (article: ArticleOrbitItem) => void;
}

const PlanetBody = () => (
  <group>
    <mesh>
      <sphereGeometry args={[2.18, 72, 72]} />
      <meshPhysicalMaterial
        color="#07111f"
        emissive="#064e3b"
        emissiveIntensity={0.1}
        metalness={0.22}
        opacity={0.42}
        roughness={0.48}
        transparent
        transmission={0.18}
      />
    </mesh>
    <mesh rotation={[Math.PI / 2.2, 0.15, 0]}>
      <torusGeometry args={[2.42, 0.006, 12, 160]} />
      <meshBasicMaterial color="#10b981" transparent opacity={0.32} />
    </mesh>
    <mesh rotation={[Math.PI / 2.8, 0.6, 0.9]}>
      <torusGeometry args={[2.64, 0.004, 12, 160]} />
      <meshBasicMaterial color="#38bdf8" transparent opacity={0.22} />
    </mesh>
  </group>
);

export const ArticlePlanetScene = ({
  activeArticleId,
  articles,
  onArticleFocus,
  onArticleOpen,
}: ArticlePlanetSceneProps) => {
  const nodes = useMemo(() => buildArticlePlanetLayout(articles), [articles]);

  return (
    <Canvas
      camera={{ fov: 45, position: [0, 0, 6.3] }}
      dpr={[1, 1.75]}
      gl={{ antialias: true, alpha: true, powerPreference: 'high-performance' }}
    >
      <color attach="background" args={['#020617']} />
      <ambientLight intensity={0.42} />
      <directionalLight color="#dffdf2" intensity={1.8} position={[4, 3, 5]} />
      <pointLight color="#38bdf8" intensity={22} position={[-3, -1, 3]} />
      <Stars count={900} depth={38} factor={3.2} fade radius={42} speed={0.18} />
      <group rotation={[0.08, -0.24, 0]}>
        <PlanetBody />
        {nodes.map((node) => (
          <ArticlePlanetNode
            key={node.key}
            isActive={node.article.id === activeArticleId}
            node={node}
            onFocus={onArticleFocus}
            onOpen={onArticleOpen}
          />
        ))}
      </group>
      <OrbitControls
        autoRotate
        autoRotateSpeed={0.42}
        enableDamping
        enablePan={false}
        maxDistance={7.2}
        minDistance={4.25}
        rotateSpeed={0.52}
        zoomSpeed={0.45}
      />
      <AdaptiveDpr pixelated />
      <Preload all />
    </Canvas>
  );
};
