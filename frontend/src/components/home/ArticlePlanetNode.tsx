import { useRef } from 'react';
import { useFrame } from '@react-three/fiber';
import type { ThreeEvent } from '@react-three/fiber';
import { AdditiveBlending, Vector3, type Group } from 'three';
import type { ArticleOrbitItem } from '@/types';
import type { ArticlePlanetNodeLayout } from './articlePlanetLayout';

interface ArticlePlanetNodeProps {
  isActive: boolean;
  node: ArticlePlanetNodeLayout;
  onFocus: (article: ArticleOrbitItem) => void;
  onOpen: (article: ArticleOrbitItem) => void;
}

export const ArticlePlanetNode = ({ isActive, node, onFocus, onOpen }: ArticlePlanetNodeProps) => {
  const groupRef = useRef<Group>(null);
  const initialPositionRef = useRef<[number, number, number]>(node.position);
  const targetPositionRef = useRef(new Vector3(...node.position));
  const isRelated = node.gravityRole === 'related';
  const isDimmed = node.gravityRole === 'dimmed';
  const gravityScore = node.gravityScore ?? 0;
  const activeScale = isActive ? node.visual.activeScale : 1;
  const gravityScale = isRelated ? 1.08 + gravityScore * 0.08 : isDimmed ? 0.88 : 1;
  const haloOpacity = isActive
    ? Math.min(node.visual.haloOpacity + 0.08, 0.24)
    : isRelated
      ? Math.min(node.visual.haloOpacity + 0.09 + gravityScore * 0.05, 0.28)
      : isDimmed
        ? node.visual.haloOpacity * 0.34
        : node.visual.haloOpacity;
  const ringOpacity = isActive
    ? Math.min(node.visual.ringOpacity + 0.16, 0.68)
    : isRelated
      ? Math.min(node.visual.ringOpacity + 0.18, 0.72)
      : isDimmed
        ? node.visual.ringOpacity * 0.38
        : node.visual.ringOpacity;
  const shellOpacity = isActive
    ? Math.min(node.visual.shellOpacity + 0.05, 0.32)
    : isRelated
      ? Math.min(node.visual.shellOpacity + 0.06, 0.34)
      : isDimmed
        ? node.visual.shellOpacity * 0.45
        : node.visual.shellOpacity;
  const coreOpacity = isDimmed ? 0.58 : 1;
  const emissiveMultiplier = isActive ? 0.34 : isRelated ? 0.3 + gravityScore * 0.08 : isDimmed ? 0.08 : 0.18;

  useFrame((_, delta) => {
    targetPositionRef.current.set(node.position[0], node.position[1], node.position[2]);
    groupRef.current?.position.lerp(targetPositionRef.current, Math.min(1, delta * 2.8));
  });

  const handlePointer = (event: ThreeEvent<PointerEvent>) => {
    event.stopPropagation();
    onFocus(node.article);
  };

  const handleClick = (event: ThreeEvent<MouseEvent>) => {
    event.stopPropagation();
    onOpen(node.article);
  };

  return (
    <group
      ref={groupRef}
      onClick={handleClick}
      onPointerOver={handlePointer}
      onPointerMove={handlePointer}
      position={initialPositionRef.current}
      scale={activeScale * gravityScale}
    >
      <mesh scale={isActive ? 1.16 : 1}>
        <sphereGeometry args={[node.visual.haloRadius, 22, 14]} />
        <meshBasicMaterial
          blending={AdditiveBlending}
          color={node.visual.atmosphereColor}
          depthTest={false}
          depthWrite={false}
          opacity={haloOpacity}
          transparent
        />
      </mesh>
      <mesh>
        <sphereGeometry args={[node.visual.shellRadius, 28, 18]} />
        <meshBasicMaterial
          blending={AdditiveBlending}
          color={node.visual.rimColor}
          depthTest={false}
          depthWrite={false}
          opacity={shellOpacity}
          transparent
        />
      </mesh>
      <mesh>
        <sphereGeometry args={[node.visual.coreRadius, 28, 20]} />
        <meshPhysicalMaterial
          clearcoat={0.95}
          clearcoatRoughness={0.18}
          color={node.visual.surfaceColor}
          depthTest={false}
          emissive={node.visual.atmosphereColor}
          emissiveIntensity={node.emissiveIntensity * emissiveMultiplier}
          metalness={0.18}
          opacity={coreOpacity}
          roughness={0.28}
          transparent={isDimmed}
        />
      </mesh>
      <mesh position={[node.visual.coreRadius * -0.34, node.visual.coreRadius * 0.46, node.visual.coreRadius * 0.82]}>
        <sphereGeometry args={[node.visual.glintRadius, 14, 10]} />
        <meshBasicMaterial color="#fff7ed" depthTest={false} transparent opacity={isActive ? 0.86 : 0.64} />
      </mesh>
      <mesh position={[node.visual.coreRadius * 0.26, node.visual.coreRadius * -0.36, node.visual.coreRadius * 0.76]}>
        <sphereGeometry args={[node.visual.glintRadius * 0.52, 10, 8]} />
        <meshBasicMaterial color={node.visual.shadowColor} depthTest={false} transparent opacity={0.28} />
      </mesh>
      <mesh rotation={[Math.PI / 2.7 + node.weight * 0.05, node.weight * 0.22, Math.PI / 8]}>
        <torusGeometry args={[node.visual.ringRadius, node.visual.coreRadius * 0.025, 8, 80]} />
        <meshBasicMaterial
          blending={AdditiveBlending}
          color={node.visual.accentColor}
          depthTest={false}
          depthWrite={false}
          transparent
          opacity={ringOpacity}
        />
      </mesh>
    </group>
  );
};
