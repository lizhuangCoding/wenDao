import { useEffect, useRef } from 'react';

interface UseNotificationPollingOptions {
  enabled: boolean;
  intervalMs?: number;
  onPoll: () => Promise<void>;
}

export const useNotificationPolling = ({
  enabled,
  intervalMs = 30_000,
  onPoll,
}: UseNotificationPollingOptions) => {
  const pollRef = useRef(onPoll);

  useEffect(() => {
    pollRef.current = onPoll;
  }, [onPoll]);

  useEffect(() => {
    if (!enabled) return;

    let timer: ReturnType<typeof window.setInterval> | null = null;

    const clearTimer = () => {
      if (timer !== null) {
        window.clearInterval(timer);
        timer = null;
      }
    };

    const runPoll = () => {
      void pollRef.current();
    };

    const startPolling = () => {
      clearTimer();
      if (document.hidden) return;
      runPoll();
      timer = window.setInterval(runPoll, intervalMs);
    };

    const handleVisibilityChange = () => {
      if (document.hidden) {
        clearTimer();
        return;
      }
      startPolling();
    };

    startPolling();
    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      clearTimer();
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [enabled, intervalMs]);
};
