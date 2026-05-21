import { useEffect, useState } from 'react';

export const useCountdown = () => {
  const [seconds, setSeconds] = useState(0);

  useEffect(() => {
    if (seconds <= 0) return undefined;
    const timer = window.setTimeout(() => {
      setSeconds((current) => Math.max(current - 1, 0));
    }, 1000);
    return () => window.clearTimeout(timer);
  }, [seconds]);

  return {
    seconds,
    isActive: seconds > 0,
    start: setSeconds,
    reset: () => setSeconds(0),
  };
};

