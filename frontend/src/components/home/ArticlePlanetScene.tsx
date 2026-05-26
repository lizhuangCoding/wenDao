import { useEffect, useMemo, useRef } from 'react';
import { Canvas, useFrame, useThree } from '@react-three/fiber';
import { OrbitControls as ThreeOrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';
import type { ArticleOrbitItem } from '@/types';
import { ArticlePlanetNode } from './ArticlePlanetNode';
import { buildArticlePlanetLayout } from './articlePlanetLayout';

interface ArticlePlanetSceneProps {
  activeArticleId?: number;
  articles: ArticleOrbitItem[];
  onArticleFocus: (article: ArticleOrbitItem) => void;
  onArticleOpen: (article: ArticleOrbitItem) => void;
}

interface ArticlePlanetClusterProps extends ArticlePlanetSceneProps {
  nodes: ReturnType<typeof buildArticlePlanetLayout>;
}

const randomUnit = (seed: number) => {
  const value = Math.sin(seed * 12.9898) * 43758.5453;
  return value - Math.floor(value);
};

const LightweightStarField = () => {
  const positions = useMemo(() => {
    const count = 420;
    const values = new Float32Array(count * 3);

    for (let index = 0; index < count; index += 1) {
      const radius = 10 + randomUnit(index + 1) * 32;
      const theta = randomUnit(index + 11) * Math.PI * 2;
      const phi = Math.acos(2 * randomUnit(index + 23) - 1);
      const offset = index * 3;

      values[offset] = Math.sin(phi) * Math.cos(theta) * radius;
      values[offset + 1] = Math.cos(phi) * radius;
      values[offset + 2] = Math.sin(phi) * Math.sin(theta) * radius;
    }

    return values;
  }, []);

  return (
    <points>
      <bufferGeometry>
        <bufferAttribute attach="attributes-position" args={[positions, 3]} />
      </bufferGeometry>
      <pointsMaterial color="#dbeafe" depthWrite={false} opacity={0.82} size={0.034} transparent />
    </points>
  );
};

const PlanetBody = () => (
  <group>
    <mesh>
      <sphereGeometry args={[2.18, 56, 44]} />
      <meshStandardMaterial
        color="#0f3d56"
        emissive="#0e7490"
        emissiveIntensity={0.3}
        metalness={0.08}
        roughness={0.58}
      />
    </mesh>
    <mesh scale={1.08}>
      <sphereGeometry args={[2.18, 36, 36]} />
      <meshBasicMaterial color="#38bdf8" depthWrite={false} opacity={0.32} transparent />
    </mesh>
    <mesh scale={1.015}>
      <sphereGeometry args={[2.18, 28, 28]} />
      <meshBasicMaterial color="#67e8f9" depthWrite={false} opacity={0.56} transparent wireframe />
    </mesh>
    <mesh rotation={[Math.PI / 2.2, 0.15, 0]}>
      <torusGeometry args={[2.42, 0.018, 8, 112]} />
      <meshBasicMaterial color="#6ee7b7" transparent opacity={0.76} />
    </mesh>
    <mesh rotation={[Math.PI / 2.8, 0.6, 0.9]}>
      <torusGeometry args={[2.64, 0.014, 8, 112]} />
      <meshBasicMaterial color="#bae6fd" transparent opacity={0.6} />
    </mesh>
  </group>
);

const SceneControls = () => {
  const { camera, gl } = useThree();
  const controlsRef = useRef<ThreeOrbitControls | null>(null);

  useEffect(() => {
    const controls = new ThreeOrbitControls(camera, gl.domElement);
    controls.autoRotate = true;
    controls.autoRotateSpeed = 0.42;
    controls.enableDamping = true;
    controls.enablePan = false;
    controls.enableZoom = false;
    controls.maxDistance = 7.2;
    controls.minDistance = 4.25;
    controls.rotateSpeed = 0.52;
    controlsRef.current = controls;

    return () => {
      controls.dispose();
      controlsRef.current = null;
    };
  }, [camera, gl]);

  useFrame(() => {
    controlsRef.current?.update();
  });

  return null;
};

const ArticlePlanetCluster = ({
  activeArticleId,
  nodes,
  onArticleFocus,
  onArticleOpen,
}: ArticlePlanetClusterProps) => {
  const { size } = useThree();
  const isCompact = size.width < 768;
  const position: [number, number, number] = isCompact ? [0.16, 1.1, 0] : [1.88, 0.12, 0];
  const scale = isCompact ? 0.58 : 0.82;

  return (
    <group position={position} rotation={[0.08, -0.24, 0]} scale={scale}>
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
  );
};

export const ArticlePlanetScene = ({
  activeArticleId,
  articles,
  onArticleFocus,
  onArticleOpen,
}: ArticlePlanetSceneProps) => {
  const nodes = useMemo(() => buildArticlePlanetLayout(articles), [articles]);

  return (
    <Canvas
      camera={{ fov: 45, position: [0, 0, 6.8] }}
      dpr={[1, 1.35]}
      gl={{ antialias: false, alpha: true, powerPreference: 'high-performance' }}
    >
      <color attach="background" args={['#020617']} />
      <ambientLight intensity={0.42} />
      <directionalLight color="#dffdf2" intensity={2.1} position={[4, 3, 5]} />
      <pointLight color="#38bdf8" intensity={28} position={[-3, -1, 3]} />
      <LightweightStarField />
      <ArticlePlanetCluster
        activeArticleId={activeArticleId}
        articles={articles}
        nodes={nodes}
        onArticleFocus={onArticleFocus}
        onArticleOpen={onArticleOpen}
      />
      <SceneControls />
    </Canvas>
  );
};
