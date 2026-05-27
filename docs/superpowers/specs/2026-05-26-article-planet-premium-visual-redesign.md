# Article Planet Premium Visual Redesign

## Context

The current article planet interaction works, but the visual result feels cheap. The main sphere reads like a dense debugging mesh, and article nodes use oversized translucent blobs that overpower the scene instead of feeling like polished 3D objects.

## Direction

Use the selected "高级数字星图" direction:

- Keep the existing Three.js and `@react-three/fiber` interaction model.
- Keep mouse rotation, hover focus, and click-to-open article behavior unchanged.
- Replace the cheap node look with compact premium mini planets: glossy surface, subtle atmosphere, small specular highlights, refined rim glow, and occasional thin ring.
- Replace the heavy solid main planet look with a clean blue holographic wireframe sphere and soft inner glow.
- Keep the scene readable behind the hero title and article card.

## Visual Rules

- Article node halo must stay close to the body. It should frame the planet, not become a large colored circle.
- Node colors use a restrained premium palette: cyan, violet, rose gold, teal, soft amber, and blue.
- Every node gets layered material data: surface, atmosphere, rim, accent, shadow, and glow.
- Main sphere uses transparent shell plus wireframe/rings; no opaque oversized filled sphere.
- Background uses deterministic star dust and a few small lens-like particles, not noisy decorative blobs.

## Performance Rules

- Do not reintroduce `@react-three/drei`.
- Keep geometry segment counts modest.
- Keep Canvas DPR conservative.
- Avoid texture files and dependency-heavy postprocessing. Procedural geometry and materials are enough.

## Verification

- Extend layout tests to assert premium node visual constraints.
- Keep dependency boundary test preventing `@react-three/drei`.
- Run `npm run test:layout`, `npm run lint`, and `npm run build`.
- Use browser screenshots at desktop and mobile sizes to inspect visual composition.
