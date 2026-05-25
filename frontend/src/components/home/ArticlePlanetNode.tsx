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
  const handlePointer = (event: ThreeEvent<PointerEvent>) => {
    event.stopPropagation();
    onFocus(node.article);
  };

  const handleClick = (event: ThreeEvent<MouseEvent>) => {
    event.stopPropagation();
    onOpen(node.article);
  };

  return (
    <group position={node.position}>
      <mesh
        onClick={handleClick}
        onPointerOver={handlePointer}
        onPointerMove={handlePointer}
        scale={isActive ? 1.45 : 1}
      >
        <sphereGeometry args={[node.radius, 24, 24]} />
        <meshStandardMaterial
          color={node.color}
          emissive={node.color}
          emissiveIntensity={isActive ? node.emissiveIntensity + 0.7 : node.emissiveIntensity}
          roughness={0.2}
          metalness={0.15}
        />
      </mesh>
      <mesh scale={isActive ? 1.95 : 1.45}>
        <sphereGeometry args={[node.radius, 16, 16]} />
        <meshBasicMaterial color={node.color} transparent opacity={isActive ? 0.22 : 0.12} />
      </mesh>
    </group>
  );
};
