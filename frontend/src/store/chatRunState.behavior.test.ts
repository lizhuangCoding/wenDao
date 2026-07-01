import { describe, expect, it } from 'vitest';
import { createIdleChatRunState } from './chatRunState';

describe('chat run state helpers', () => {
  it('creates the default idle chat run state', () => {
    expect(createIdleChatRunState()).toEqual({
      isTyping: false,
      isStreaming: false,
      streamingConversationId: null,
      currentStage: null,
      currentStageLabel: null,
      requiresUserInput: false,
      pendingQuestion: null,
      runStatus: 'idle',
      isRecovering: false,
      reconnectAttempts: 0,
      lastHeartbeatAt: null,
    });
  });
});
