import { useEffect, useState } from 'react';

const formatProcessingDuration = (elapsedMs: number) => {
  const totalSeconds = Math.max(0, Math.floor(elapsedMs / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  if (minutes > 0) {
    return `${minutes}分${seconds.toString().padStart(2, '0')}秒`;
  }

  return `${seconds}秒`;
};

export const useProcessingTimer = (isRunning: boolean) => {
  const [startedAt, setStartedAt] = useState<number | null>(null);
  const [elapsedMs, setElapsedMs] = useState(0);

  useEffect(() => {
    if (!isRunning) {
      setStartedAt(null);
      setElapsedMs(0);
      return;
    }

    const startTime = startedAt ?? Date.now();
    if (startedAt === null) {
      setStartedAt(startTime);
      setElapsedMs(0);
    }

    const updateElapsed = () => setElapsedMs(Date.now() - startTime);
    updateElapsed();
    const timer = window.setInterval(updateElapsed, 1000);
    return () => window.clearInterval(timer);
  }, [isRunning, startedAt]);

  return {
    elapsedMs,
    elapsedLabel: formatProcessingDuration(elapsedMs),
  };
};
