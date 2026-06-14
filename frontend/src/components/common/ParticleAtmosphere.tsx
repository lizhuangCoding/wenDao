import { useEffect, useMemo, useRef, useState } from 'react';

type ParticleTone = 'planet' | 'reading';

interface ParticleAtmosphereProps {
  count?: number;
  tone?: ParticleTone;
}

interface ParticleSeed {
  id: number;
  x: number;
  y: number;
  size: number;
  opacity: number;
  depth: number;
  drift: number;
}

interface ParticleMouseState {
  x: number;
  y: number;
  active: boolean;
  decay: number;
  velocity: number;
  lastMoveTime: number;
}

const PARTICLE_MEDIA_QUERY = '(hover: hover) and (pointer: fine)';
const REDUCED_MOTION_QUERY = '(prefers-reduced-motion: reduce)';

const toneClassNames: Record<ParticleTone, string> = {
  planet: 'from-primary-200 via-cyan-200 to-sky-300 shadow-[0_0_18px_rgba(125,211,252,0.58)]',
  reading: 'from-primary-100 via-sky-100 to-neutral-100 shadow-[0_0_14px_rgba(16,185,129,0.24)]',
};

const clampParticleCount = (count: number) => Math.min(56, Math.max(18, Math.round(count)));

const seededRandom = (seed: number) => {
  const value = Math.sin(seed * 91.771 + 17.13) * 10000;
  return value - Math.floor(value);
};

const buildParticles = (count: number): ParticleSeed[] =>
  Array.from({ length: clampParticleCount(count) }, (_, index) => ({
    id: index,
    x: 4 + seededRandom(index + 1) * 92,
    y: 6 + seededRandom(index + 41) * 88,
    size: 2 + seededRandom(index + 89) * 4,
    opacity: 0.18 + seededRandom(index + 127) * 0.34,
    depth: 0.35 + seededRandom(index + 173) * 0.9,
    drift: 0.5 + seededRandom(index + 211) * 1.4,
  }));

const supportsParticleAtmosphere = () => {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return false;
  }
  return (
    window.matchMedia(PARTICLE_MEDIA_QUERY).matches &&
    !window.matchMedia(REDUCED_MOTION_QUERY).matches
  );
};

export const ParticleAtmosphere = ({ count = 42, tone = 'planet' }: ParticleAtmosphereProps) => {
  const [isEnabled, setIsEnabled] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const mouseRef = useRef<ParticleMouseState>({
    x: 0.5,
    y: 0.5,
    active: false,
    decay: 0,
    velocity: 0,
    lastMoveTime: 0,
  });
  const frameRef = useRef<number>();
  const particles = useMemo(() => buildParticles(count), [count]);

  useEffect(() => {
    const updateEnabled = () => setIsEnabled(supportsParticleAtmosphere());
    updateEnabled();

    const hoverMedia = window.matchMedia(PARTICLE_MEDIA_QUERY);
    const motionMedia = window.matchMedia(REDUCED_MOTION_QUERY);
    hoverMedia.addEventListener('change', updateEnabled);
    motionMedia.addEventListener('change', updateEnabled);

    return () => {
      hoverMedia.removeEventListener('change', updateEnabled);
      motionMedia.removeEventListener('change', updateEnabled);
    };
  }, []);

  useEffect(() => {
    if (!isEnabled) return undefined;

    const root = rootRef.current;
    if (!root) return undefined;

    const particleNodes = Array.from(root.querySelectorAll<HTMLElement>('[data-particle]'));
    const toneMotion = tone === 'reading' ? 0.76 : 1;
    const twinkleStrength = tone === 'reading' ? 0.045 : 0.07;
    const maxOpacity = tone === 'reading' ? 0.72 : 0.9;

    const handlePointerMove = (event: PointerEvent) => {
      const currentMouse = mouseRef.current;
      const nextX = event.clientX / Math.max(window.innerWidth, 1);
      const nextY = event.clientY / Math.max(window.innerHeight, 1);
      const now = window.performance.now();
      const elapsedMove = currentMouse.lastMoveTime ? Math.max(now - currentMouse.lastMoveTime, 16) : 0;
      const travel = Math.hypot(nextX - currentMouse.x, nextY - currentMouse.y);
      const rawVelocity = elapsedMove ? Math.min(3, travel / (elapsedMove / 1000)) : 0;

      mouseRef.current = {
        x: nextX,
        y: nextY,
        active: true,
        decay: 1,
        velocity: currentMouse.velocity * 0.68 + rawVelocity * 0.32,
        lastMoveTime: now,
      };
    };

    const handlePointerLeave = () => {
      mouseRef.current.active = false;
      mouseRef.current.decay = Math.max(mouseRef.current.decay, 1);
    };

    const animate = (time: number) => {
      const elapsed = time / 1000;
      const mouse = mouseRef.current;
      const activeInfluence = mouse.active ? 1 : Math.max(0, mouse.decay);
      const velocityInfluence = Math.min(1, mouse.velocity * 0.48);

      mouse.decay = mouse.active ? 1 : mouse.decay * 0.92;
      mouse.velocity *= mouse.active ? 0.9 : 0.86;

      particleNodes.forEach((node, index) => {
        const particle = particles[index];
        if (!particle) return;

        const dx = mouse.x - particle.x / 100;
        const dy = mouse.y - particle.y / 100;
        const distance = Math.sqrt(dx * dx + dy * dy);
        const baseInfluence = Math.max(0, 1 - distance * 3.4) * activeInfluence;
        const influence = Math.min(1, baseInfluence * (0.62 + velocityInfluence * 0.48));
        const driftX =
          (Math.sin(elapsed * particle.drift * 0.36 + particle.id) * 1 +
            Math.sin(elapsed * 0.68 + particle.id * 2.3) * 0.46 +
            Math.sin(elapsed * 0.22 + particle.id * 4.1) * 0.22) *
          7.6 *
          particle.depth *
          toneMotion;
        const driftY =
          (Math.cos(elapsed * particle.drift * 0.31 + particle.id) * 1 +
            Math.sin(elapsed * 0.52 + particle.id * 1.7) * 0.4 +
            Math.cos(elapsed * 0.18 + particle.id * 3.4) * 0.24) *
          5.8 *
          particle.depth *
          toneMotion;
        const followX = dx * 54 * influence * particle.depth * toneMotion;
        const followY = dy * 42 * influence * particle.depth * toneMotion;
        const scale = 1 + influence * 1.55;
        const twinkle =
          1 +
          Math.sin(elapsed * 0.58 + particle.id * 7.7) * twinkleStrength +
          Math.sin(elapsed * 0.19 + particle.id * 3.9) * twinkleStrength * 0.36;
        const opacity = Math.min(maxOpacity, (particle.opacity + influence * 0.44) * twinkle);

        node.style.transform = `translate3d(${driftX + followX}px, ${driftY + followY}px, 0) scale(${scale})`;
        node.style.opacity = String(opacity);
      });

      frameRef.current = window.requestAnimationFrame(animate);
    };

    window.addEventListener('pointermove', handlePointerMove, { passive: true });
    window.addEventListener('pointerleave', handlePointerLeave);
    frameRef.current = window.requestAnimationFrame(animate);

    return () => {
      window.removeEventListener('pointermove', handlePointerMove);
      window.removeEventListener('pointerleave', handlePointerLeave);
      if (frameRef.current) {
        window.cancelAnimationFrame(frameRef.current);
      }
    };
  }, [isEnabled, particles, tone]);

  if (!isEnabled) return null;

  return (
    <div
      ref={rootRef}
      aria-hidden="true"
      className="pointer-events-none fixed inset-0 z-[1] overflow-hidden"
      data-particle-atmosphere={tone}
    >
      {particles.map((particle) => (
        <span
          key={particle.id}
          data-particle=""
          className={`absolute rounded-full bg-gradient-to-br blur-[0.2px] will-change-transform ${toneClassNames[tone]}`}
          style={{
            left: `${particle.x}%`,
            top: `${particle.y}%`,
            width: `${particle.size}px`,
            height: `${particle.size}px`,
            opacity: particle.opacity,
          }}
        />
      ))}
    </div>
  );
};
