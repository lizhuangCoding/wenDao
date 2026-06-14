import { useEffect, useMemo, useRef } from 'react';
import { Canvas, useFrame, useThree } from '@react-three/fiber';
import { AdditiveBlending, BackSide, type BufferGeometry, type Group } from 'three';
import { OrbitControls as ThreeOrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';
import type { ArticleOrbitItem } from '@/types';
import { ArticlePlanetNode } from './ArticlePlanetNode';
import {
  buildArticlePlanetConnections,
  buildArticlePlanetLayout,
  type ArticlePlanetConnection,
} from './articlePlanetLayout';
import { buildArticlePlanetGravityLayout } from './articlePlanetGravity';

interface ArticlePlanetSceneProps {
  activeArticleId?: number;
  articles: ArticleOrbitItem[];
  onArticleFocus: (article: ArticleOrbitItem) => void;
  onArticleOpen: (article: ArticleOrbitItem) => void;
}

interface ArticlePlanetClusterProps extends ArticlePlanetSceneProps {
  nodes: ReturnType<typeof buildArticlePlanetLayout>;
}

const LATITUDE_RINGS = [-0.72, -0.48, -0.24, 0, 0.24, 0.48, 0.72].map((ratio) => ({
  radius: 2.16 * Math.sqrt(1 - ratio * ratio),
  y: 2.16 * ratio,
}));

const CLUSTER_BASE_ROTATION: [number, number, number] = [0.08, -0.24, 0];
const MERIDIAN_RINGS = Array.from({ length: 8 }, (_, index) => (index * Math.PI) / 8);
const PLANET_DESKTOP_DRIFT_AMPLITUDE = 0.18;
const PLANET_MOBILE_DRIFT_AMPLITUDE = 0.07;
const PLANET_DRIFT_SPEED = 0.18;
const PLANET_SELF_ROTATION_SPEED = 0.045;
const GRAVITY_LINE_LERP_SPEED = 3.2;

const PlanetBody = () => (
  <group>
    <pointLight color="#38bdf8" distance={4.8} intensity={9} position={[0.4, 0.5, 1.4]} />
    <mesh>
      <sphereGeometry args={[2.16, 56, 36]} />
      <meshBasicMaterial
        color="#07324a"
        depthWrite={false}
        opacity={0.3}
        transparent
      />
    </mesh>
    <mesh scale={1.04}>
      <sphereGeometry args={[2.16, 48, 30]} />
      <meshBasicMaterial
        color="#7dd3fc"
        depthWrite={false}
        opacity={0.26}
        side={BackSide}
        transparent
      />
    </mesh>
    <mesh scale={1.012}>
      <sphereGeometry args={[2.16, 16, 12]} />
      <meshBasicMaterial color="#a7f3ff" depthWrite={false} opacity={0.04} transparent wireframe />
    </mesh>
    {LATITUDE_RINGS.map((ring) => (
      <mesh key={`lat-${ring.y}`} position={[0, ring.y, 0]} rotation={[Math.PI / 2, 0, 0]}>
        <torusGeometry args={[ring.radius, 0.0045, 6, 128]} />
        <meshBasicMaterial
          blending={AdditiveBlending}
          color="#9be8ff"
          depthWrite={false}
          opacity={0.78}
          transparent
        />
      </mesh>
    ))}
    {MERIDIAN_RINGS.map((rotation) => (
      <mesh key={`meridian-${rotation}`} rotation={[0, rotation, 0]}>
        <torusGeometry args={[2.16, 0.0035, 6, 128]} />
        <meshBasicMaterial
          blending={AdditiveBlending}
          color="#5bd3ff"
          depthWrite={false}
          opacity={0.54}
          transparent
        />
      </mesh>
    ))}
    <mesh rotation={[Math.PI / 2.2, 0.15, 0]}>
      <torusGeometry args={[2.42, 0.008, 8, 144]} />
      <meshBasicMaterial
        blending={AdditiveBlending}
        color="#8ee7ff"
        depthWrite={false}
        transparent
        opacity={0.78}
      />
    </mesh>
    <mesh rotation={[Math.PI / 2.8, 0.6, 0.9]}>
      <torusGeometry args={[2.65, 0.006, 8, 144]} />
      <meshBasicMaterial
        blending={AdditiveBlending}
        color="#a78bfa"
        depthWrite={false}
        transparent
        opacity={0.6}
      />
    </mesh>
  </group>
);

const ArticlePlanetConnectionLine = ({ connection }: { connection: ArticlePlanetConnection }) => {
  const geometryRef = useRef<BufferGeometry>(null);
  const positionsRef = useRef(
    new Float32Array([
      connection.from[0],
      connection.from[1],
      connection.from[2],
      connection.to[0],
      connection.to[1],
      connection.to[2],
    ])
  ).current;
  const positions = useMemo(
    () =>
      new Float32Array([
        connection.from[0],
        connection.from[1],
        connection.from[2],
        connection.to[0],
        connection.to[1],
        connection.to[2],
      ]),
    [connection]
  );

  useFrame((_, delta) => {
    const lerpAmount = Math.min(1, delta * GRAVITY_LINE_LERP_SPEED);
    for (let index = 0; index < positionsRef.length; index += 1) {
      positionsRef[index] += (positions[index] - positionsRef[index]) * lerpAmount;
    }
    const positionAttribute = geometryRef.current?.attributes.position;
    if (positionAttribute) {
      positionAttribute.needsUpdate = true;
    }
  });

  return (
    <line>
      <bufferGeometry ref={geometryRef}>
        <bufferAttribute attach="attributes-position" args={[positionsRef, 3]} />
      </bufferGeometry>
      <lineBasicMaterial
        blending={AdditiveBlending}
        color={connection.color}
        depthWrite={false}
        opacity={connection.opacity}
        transparent
      />
    </line>
  );
};

const SceneControls = () => {
  const { camera, gl } = useThree();
  const controlsRef = useRef<ThreeOrbitControls | null>(null);

  useEffect(() => {
    const controls = new ThreeOrbitControls(camera, gl.domElement);
    controls.autoRotate = false;
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
  const clusterRef = useRef<Group>(null);
  const spinRef = useRef<Group>(null);
  const isCompact = size.width < 768;
  const basePosition: [number, number, number] = isCompact ? [0.16, 1.1, 0] : [1.52, -0.02, 0];
  const driftAmplitude = isCompact ? PLANET_MOBILE_DRIFT_AMPLITUDE : PLANET_DESKTOP_DRIFT_AMPLITUDE;
  const scale = isCompact ? 0.58 : 0.76;
  const gravityNodes = useMemo(
    () => buildArticlePlanetGravityLayout(nodes, activeArticleId),
    [activeArticleId, nodes]
  );
  const connections = useMemo(
    () => buildArticlePlanetConnections(gravityNodes, activeArticleId),
    [activeArticleId, gravityNodes]
  );

  useFrame(({ clock }, delta) => {
    if (clusterRef.current) {
      clusterRef.current.position.x =
        basePosition[0] + Math.sin(clock.elapsedTime * PLANET_DRIFT_SPEED) * driftAmplitude;
    }
    if (spinRef.current) {
      spinRef.current.rotation.y += delta * PLANET_SELF_ROTATION_SPEED;
    }
  });

  return (
    <group ref={clusterRef} position={basePosition} rotation={CLUSTER_BASE_ROTATION} scale={scale}>
      <group ref={spinRef}>
        <PlanetBody />
        {connections.map((connection) => (
          <ArticlePlanetConnectionLine key={connection.key} connection={connection} />
        ))}
        {gravityNodes.map((node) => (
          <ArticlePlanetNode
            key={node.key}
            isActive={node.article.id === activeArticleId}
            node={node}
            onFocus={onArticleFocus}
            onOpen={onArticleOpen}
          />
        ))}
      </group>
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
