import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ChatActiveRun, ChatConversationDetailResponse, ChatStage } from '@/types';
import type { Conversation } from './chatNormalizers';
import { mapConversationDetail } from './chatNormalizers';
import type { SelectedChatModel } from './chatPersistence';
import { resumeConversationStream } from './chatStream';

const chatApiMock = vi.hoisted(() => ({
  getConversation: vi.fn(),
  resumeStream: vi.fn(),
}));

vi.mock('@/api', () => ({
  chatApi: chatApiMock,
}));

type TestState = {
  conversations: Record<number, Conversation>;
  activeId: number | null;
  isTyping: boolean;
  isStreaming: boolean;
  streamingConversationId: number | null;
  currentStage: ChatStage | null;
  currentStageLabel: string | null;
  requiresUserInput: boolean;
  pendingQuestion: string | null;
  runStatus: 'idle' | 'running' | 'waiting_user' | 'completed' | 'failed';
  isRecovering: boolean;
  reconnectAttempts: number;
  lastHeartbeatAt: number | null;
  selectedModel: SelectedChatModel | null;
};

const baseRun: ChatActiveRun = {
  id: 77,
  status: 'running',
  current_stage: 'analyzing',
  last_answer: '',
  can_resume: true,
};

const createConversation = (): Conversation => ({
  id: 1,
  title: 'Conversation',
  messages: [],
  steps: [],
  activeRun: baseRun,
  createdAt: Date.now(),
  updatedAt: Date.now(),
  isLoaded: true,
});

const createState = (): TestState => ({
  conversations: {
    1: createConversation(),
  },
  activeId: 1,
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
  selectedModel: null,
});

const applySet = (state: TestState, partial: Partial<TestState> | ((state: TestState) => Partial<TestState>)) => {
  const next = typeof partial === 'function' ? partial(state) : partial;
  return {
    ...state,
    ...next,
  };
};

describe('chat stream resume behavior', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('keeps the conversation resumable when the backend asks a follow-up question', async () => {
    let state = createState();
    const get = () => state;
    const set = (partial: Partial<TestState> | ((state: TestState) => Partial<TestState>)) => {
      state = applySet(state, partial);
    };

    chatApiMock.resumeStream.mockImplementation(async (_conversationId, _runId, handlers) => {
      handlers.onQuestion?.({ message: '请补充一下你希望输出的格式' });
    });

    await resumeConversationStream({
      applyConversationDetail: vi.fn(),
      conversationId: 1,
      get,
      run: baseRun,
      set,
    });

    expect(state.runStatus).toBe('waiting_user');
    expect(state.requiresUserInput).toBe(true);
    expect(state.pendingQuestion).toBe('请补充一下你希望输出的格式');
    expect(state.isRecovering).toBe(false);
    expect(state.isStreaming).toBe(false);
    const latestMessage = state.conversations[1].messages[state.conversations[1].messages.length - 1];
    expect(latestMessage).toMatchObject({
      role: 'assistant',
      content: '请补充一下你希望输出的格式',
      runId: 77,
    });
  });

  it('refreshes the latest conversation detail after resume completes', async () => {
    let state = createState();
    const get = () => state;
    const set = (partial: Partial<TestState> | ((state: TestState) => Partial<TestState>)) => {
      state = applySet(state, partial);
    };

    const latestDetail: ChatConversationDetailResponse = {
      conversation: {
        id: 1,
        title: 'Conversation',
        user_id: 7,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        is_shared: false,
        share_token: '',
      },
      messages: [
        {
          id: 1001,
          conversation_id: 1,
          role: 'assistant',
          content: '最终回答',
          created_at: new Date().toISOString(),
          run_id: 77,
        },
      ],
      steps: [],
      active_steps: [],
      active_run: undefined,
    };

    const applyConversationDetail = vi.fn((conversationId: number, detail: ChatConversationDetailResponse) => {
      const mapped = mapConversationDetail(detail);
      state = {
        ...state,
        conversations: {
          ...state.conversations,
          [conversationId]: mapped,
        },
      };
      return mapped;
    });

    chatApiMock.resumeStream.mockImplementation(async (_conversationId, _runId, handlers) => {
      handlers.onChunk?.({ message: '恢复中的回答' });
      await handlers.onDone?.({});
    });
    chatApiMock.getConversation.mockResolvedValue(latestDetail);

    await resumeConversationStream({
      applyConversationDetail,
      conversationId: 1,
      get,
      run: baseRun,
      set,
    });

    expect(chatApiMock.getConversation).toHaveBeenCalledWith(1);
    expect(applyConversationDetail).toHaveBeenCalledWith(1, latestDetail);
    expect(state.runStatus).toBe('completed');
    expect(state.isRecovering).toBe(false);
    expect(state.isStreaming).toBe(false);
    expect(state.currentStage).toBe('completed');
  });
});
