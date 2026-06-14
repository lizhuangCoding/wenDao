import { useEffect, useRef, useState } from 'react';

interface TrailPoint {
  age: number;
  life: number;
  speed: number;
  x: number;
  y: number;
}

const COMET_MEDIA_QUERY = '(hover: hover) and (pointer: fine)';
const REDUCED_MOTION_QUERY = '(prefers-reduced-motion: reduce)';
const MAX_TRAIL_POINTS = 34;
const MIN_POINT_DISTANCE = 5;

const supportsCursorCometTrail = () => {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return false;
  }
  return (
    window.matchMedia(COMET_MEDIA_QUERY).matches &&
    !window.matchMedia(REDUCED_MOTION_QUERY).matches
  );
};

export const CursorCometTrail = () => {
  const [isEnabled, setIsEnabled] = useState(false);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const frameRef = useRef<number>();
  const pointsRef = useRef<TrailPoint[]>([]);
  const lastPointerRef = useRef({ x: 0, y: 0, time: 0 });

  useEffect(() => {
    const updateEnabled = () => setIsEnabled(supportsCursorCometTrail());
    updateEnabled();

    const hoverMedia = window.matchMedia(COMET_MEDIA_QUERY);
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

    const canvas = canvasRef.current;
    const context = canvas?.getContext('2d');
    if (!canvas || !context) return undefined;

    const resize = () => {
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      canvas.width = Math.floor(window.innerWidth * ratio);
      canvas.height = Math.floor(window.innerHeight * ratio);
      canvas.style.width = `${window.innerWidth}px`;
      canvas.style.height = `${window.innerHeight}px`;
      context.setTransform(ratio, 0, 0, ratio, 0, 0);
    };

    const addTrailPoint = (event: PointerEvent) => {
      const now = window.performance.now();
      const last = lastPointerRef.current;
      const distance = last.time ? Math.hypot(event.clientX - last.x, event.clientY - last.y) : 0;
      if (last.time && distance < MIN_POINT_DISTANCE) return;

      const elapsed = last.time ? Math.max(now - last.time, 16) : 16;
      const speed = Math.min(1, distance / elapsed / 1.4);
      pointsRef.current.push({
        age: 0,
        life: 420 + speed * 240,
        speed,
        x: event.clientX,
        y: event.clientY,
      });
      if (pointsRef.current.length > MAX_TRAIL_POINTS) {
        pointsRef.current.splice(0, pointsRef.current.length - MAX_TRAIL_POINTS);
      }
      lastPointerRef.current = { x: event.clientX, y: event.clientY, time: now };
    };

    const animate = (time: number) => {
      context.clearRect(0, 0, window.innerWidth, window.innerHeight);

      const points = pointsRef.current;
      for (let index = points.length - 1; index >= 0; index -= 1) {
        const point = points[index];
        point.age += 16.7;
        if (point.age >= point.life) {
          points.splice(index, 1);
        }
      }

      if (points.length > 1) {
        const flameGradient = context.createLinearGradient(
          points[0].x,
          points[0].y,
          points[points.length - 1].x,
          points[points.length - 1].y
        );
        flameGradient.addColorStop(0, 'rgba(14, 165, 233, 0)');
        flameGradient.addColorStop(0.38, 'rgba(34, 211, 238, 0.24)');
        flameGradient.addColorStop(0.72, 'rgba(251, 191, 36, 0.54)');
        flameGradient.addColorStop(1, 'rgba(255, 255, 255, 0.88)');

        context.save();
        context.globalCompositeOperation = 'lighter';
        context.lineCap = 'round';
        context.lineJoin = 'round';
        context.shadowBlur = 18;
        context.shadowColor = 'rgba(251, 146, 60, 0.72)';
        context.strokeStyle = flameGradient;
        context.lineWidth = 2.2;
        context.beginPath();
        context.moveTo(points[0].x, points[0].y);

        for (let index = 1; index < points.length; index += 1) {
          const previous = points[index - 1];
          const current = points[index];
          const midX = (previous.x + current.x) / 2;
          const midY = (previous.y + current.y) / 2;
          context.quadraticCurveTo(previous.x, previous.y, midX, midY);
        }
        context.stroke();

        const head = points[points.length - 1];
        const pulse = 0.88 + Math.sin(time * 0.018) * 0.12;
        const headGradient = context.createRadialGradient(head.x, head.y, 0, head.x, head.y, 22);
        headGradient.addColorStop(0, `rgba(255, 255, 255, ${0.9 * pulse})`);
        headGradient.addColorStop(0.28, `rgba(251, 191, 36, ${0.56 * pulse})`);
        headGradient.addColorStop(1, 'rgba(14, 165, 233, 0)');
        context.fillStyle = headGradient;
        context.beginPath();
        context.arc(head.x, head.y, 22, 0, Math.PI * 2);
        context.fill();
        context.restore();
      }

      frameRef.current = window.requestAnimationFrame(animate);
    };

    resize();
    window.addEventListener('resize', resize);
    window.addEventListener('pointermove', addTrailPoint, { passive: true });
    frameRef.current = window.requestAnimationFrame(animate);

    return () => {
      window.removeEventListener('resize', resize);
      window.removeEventListener('pointermove', addTrailPoint);
      if (frameRef.current) {
        window.cancelAnimationFrame(frameRef.current);
      }
      pointsRef.current = [];
    };
  }, [isEnabled]);

  if (!isEnabled) return null;

  return (
    <canvas
      ref={canvasRef}
      aria-hidden="true"
      className="pointer-events-none fixed inset-0 z-[2]"
      data-cursor-comet-trail=""
    />
  );
};
