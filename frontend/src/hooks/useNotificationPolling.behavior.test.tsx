import { render } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useNotificationPolling } from './useNotificationPolling';

const visibilityState = {
  hidden: false,
};

const Harness = ({
  enabled,
  onPoll,
}: {
  enabled: boolean;
  onPoll: () => Promise<void>;
}) => {
  useNotificationPolling({
    enabled,
    intervalMs: 30_000,
    onPoll,
  });
  return null;
};

describe('useNotificationPolling behavior', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    visibilityState.hidden = false;
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      get: () => visibilityState.hidden,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('starts immediately, pauses while the tab is hidden, and stops when disabled', async () => {
    const onPoll = vi.fn().mockResolvedValue(undefined);
    const view = render(<Harness enabled={true} onPoll={onPoll} />);

    expect(onPoll).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(30_000);
    expect(onPoll).toHaveBeenCalledTimes(2);

    visibilityState.hidden = true;
    document.dispatchEvent(new Event('visibilitychange'));

    await vi.advanceTimersByTimeAsync(60_000);
    expect(onPoll).toHaveBeenCalledTimes(2);

    visibilityState.hidden = false;
    document.dispatchEvent(new Event('visibilitychange'));
    expect(onPoll).toHaveBeenCalledTimes(3);

    view.rerender(<Harness enabled={false} onPoll={onPoll} />);
    await vi.advanceTimersByTimeAsync(60_000);
    expect(onPoll).toHaveBeenCalledTimes(3);
  });
});
