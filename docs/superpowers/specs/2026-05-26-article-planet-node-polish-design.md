# Article Planet Node Polish Design

## Context

The homepage article planet is functional, but each article currently reads as a plain colored dot. The production deploy also failed during the frontend Docker build with a Vite transform heap OOM after adding the 3D stack.

## Goals

- Make article nodes feel like refined miniature celestial objects instead of flat markers.
- Keep the interaction model unchanged: hover/focus previews the article, click opens the article detail.
- Reduce build-time memory and bundle pressure from the 3D feature.
- Preserve the existing mature 3D foundation with `three` and `@react-three/fiber`.

## Visual Direction

Use the selected "gem article planet" direction:

- Each node has a small luminous core, translucent shell, soft halo, thin angled ring, and highlight glint.
- Active nodes grow their halo and ring, with stronger emission and a cleaner silhouette.
- Category color remains the identity anchor, but node materials use layered opacity and lightness so the result feels less like a pure CSS dot.
- Geometry stays intentionally small and low segment count to avoid making many article nodes expensive.

## Build And Runtime Optimization

Remove `@react-three/drei` from the production dependency graph. The current scene only uses `OrbitControls`, `Stars`, `AdaptiveDpr`, and `Preload`; this imports a much larger dependency surface than the feature needs.

Replacement plan:

- Use `OrbitControls` from `three/examples/jsm/controls/OrbitControls.js` through `extend`.
- Replace `Stars` with a deterministic lightweight `points` field using one memoized `BufferGeometry`.
- Remove `AdaptiveDpr` and set a conservative Canvas DPR of `[1, 1.35]`.
- Remove `Preload all`; the homepage scene is already route-local and lazy-loaded.
- Keep `three-vendor` manual chunk, but remove drei-specific package matching.

## Testing

- Extend the existing layout test to cover node visual profile values: core radius, shell radius, halo radius, ring radius, and active scale.
- Add a dependency boundary test that fails if `@react-three/drei` returns to `package.json` or scene imports.
- Continue running `npm run test:layout`, `npm run lint`, `npm run build`, and backend `go test ./...`.
- Re-run the Docker production build path that failed in GitHub Actions.
