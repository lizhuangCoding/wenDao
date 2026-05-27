import type { ThreeEvent } from '@react-three/fiber';
import { AdditiveBlending } from 'three';
import type { ArticleOrbitItem } from '@/types';
import type { ArticlePlanetNodeLayout } from './articlePlanetLayout';

interface ArticlePlanetNodeProps {
  isActive: boolean;
  node: ArticlePlanetNodeLayout;
  onFocus: (article: ArticleOrbitItem) => void;
  onOpen: (article: ArticleOrbitItem) => void;
}

export const ArticlePlanetNode = ({ isActive, node, onFocus, onOpen }: ArticlePlanetNodeProps) => {
  const activeScale = isActive ? node.visual.activeScale : 1;
  const haloOpacity = isActive ? Math.min(node.visual.haloOpacity + 0.08, 0.24) : node.visual.haloOpacity;
  const ringOpacity = isActive ? Math.min(node.visual.ringOpacity + 0.16, 0.68) : node.visual.ringOpacity;
  const shellOpacity = isActive ? Math.min(node.visual.shellOpacity + 0.05, 0.32) : node.visual.shellOpacity;

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
      onClick={handleClick}
      onPointerOver={handlePointer}
      onPointerMove={handlePointer}
      position={node.position}
      scale={activeScale}
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
          emissiveIntensity={isActive ? node.emissiveIntensity * 0.34 : node.emissiveIntensity * 0.18}
          metalness={0.18}
          roughness={0.28}
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
