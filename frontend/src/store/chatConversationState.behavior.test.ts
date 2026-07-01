import { describe, expect, it } from 'vitest';
import { createConversationMap, removeConversationRecord } from './chatConversationState';

describe('chat conversation state helpers', () => {
  it('removes one conversation without mutating the original map', () => {
    const conversations = {
      1: {
        id: 1,
        title: 'one',
        messages: [],
        steps: [],
        activeRun: null,
        createdAt: 1,
        updatedAt: 1,
        isLoaded: false,
      },
      2: {
        id: 2,
        title: 'two',
        messages: [],
        steps: [],
        activeRun: null,
        createdAt: 2,
        updatedAt: 2,
        isLoaded: false,
      },
    };

    const next = removeConversationRecord(conversations, 1);
    expect(next[1]).toBeUndefined();
    expect(next[2].title).toBe('two');
    expect(conversations[1].title).toBe('one');
  });

  it('creates a normalized conversation map from summaries', () => {
    const next = createConversationMap([
      {
        id: 7,
        title: 'Conversation',
        user_id: 1,
        created_at: '2026-07-01T00:00:00Z',
        updated_at: '2026-07-01T00:00:00Z',
        is_shared: true,
        share_token: 'token',
      },
    ]);

    expect(next[7]).toMatchObject({
      id: 7,
      title: 'Conversation',
      isLoaded: false,
      isShared: true,
      shareToken: 'token',
    });
  });
});
