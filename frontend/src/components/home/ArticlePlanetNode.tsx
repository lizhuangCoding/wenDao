import type { ThreeEvent } from '@react-three/fiber';
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
  const haloOpacity = isActive ? Math.min(node.visual.haloOpacity + 0.24, 0.62) : node.visual.haloOpacity;
  const ringOpacity = isActive ? Math.min(node.visual.ringOpacity + 0.22, 0.72) : node.visual.ringOpacity;

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
        <sphereGeometry args={[node.visual.haloRadius, 18, 18]} />
        <meshBasicMaterial
          color={node.color}
          depthTest={false}
          depthWrite={false}
          transparent
          opacity={haloOpacity}
        />
      </mesh>
      <mesh>
        <sphereGeometry args={[node.visual.shellRadius, 24, 18]} />
        <meshPhysicalMaterial
          color={node.color}
          depthTest={false}
          depthWrite={false}
          emissive={node.color}
          emissiveIntensity={isActive ? node.emissiveIntensity + 0.75 : node.emissiveIntensity * 0.9}
          metalness={0.12}
          opacity={node.visual.shellOpacity}
          roughness={0.18}
          transparent
          transmission={0.16}
        />
      </mesh>
      <mesh>
        <sphereGeometry args={[node.visual.coreRadius, 20, 16]} />
        <meshBasicMaterial
          color={node.color}
          depthTest={false}
        />
      </mesh>
      <mesh position={[node.visual.coreRadius * -0.32, node.visual.coreRadius * 0.52, node.visual.coreRadius * 0.74]}>
        <sphereGeometry args={[node.visual.glintRadius, 12, 8]} />
        <meshBasicMaterial color="#ffffff" depthTest={false} transparent opacity={isActive ? 0.78 : 0.56} />
      </mesh>
      <mesh rotation={[Math.PI / 2.7 + node.weight * 0.05, node.weight * 0.22, Math.PI / 8]}>
        <torusGeometry args={[node.visual.ringRadius, node.visual.coreRadius * 0.045, 8, 72]} />
        <meshBasicMaterial color={node.color} depthTest={false} depthWrite={false} transparent opacity={ringOpacity} />
      </mesh>
    </group>
  );
};
