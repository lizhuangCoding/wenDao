import { chatApi } from '@/api';
import type {
  ChatActiveRun,
  ChatConversationDetailResponse,
  ChatConversationMutationResponse,
  ChatMessage,
  ChatRequest,
  ChatStage,
} from '@/types';
import {
  ensureResumableAssistantMessage,
  mapConversationDetail,
  preserveExistingProcessSteps,
  stepEventToStep,
  upsertStep,
  type Conversation,
} from './chatNormalizers';
import { persistActiveChatId, type SelectedChatModel } from './chatPersistence';

type ChatRunStatus = 'idle' | 'running' | 'waiting_user' | 'completed' | 'failed';

interface ChatStreamState {
  conversations: Record<number, Conversation>;
  activeId: number | null;
  isTyping: boolean;
  isStreaming: boolean;
  streamingConversationId: number | null;
  currentStage: ChatStage | null;
  currentStageLabel: string | null;
  requiresUserInput: boolean;
  pendingQuestion: string | null;
  runStatus: ChatRunStatus;
  isRecovering: boolean;
  reconnectAttempts: number;
  lastHeartbeatAt: number | null;
  selectedModel: SelectedChatModel | null;
}

type ChatStreamSet = (
  partial:
    | Partial<ChatStreamState>
    | ((state: ChatStreamState) => Partial<ChatStreamState>)
) => void;

interface ApplyConversationDetail {
  (id: number, detail: ChatConversationDetailResponse): Conversation;
}

interface ResumeConversationParams {
  applyConversationDetail: ApplyConversationDetail;
  conversationId: number;
  get: () => ChatStreamState;
  run: ChatActiveRun | null;
  set: ChatStreamSet;
}

interface StreamConversationMessageParams {
  applyConversationDetail: ApplyConversationDetail;
  content: string;
  get: () => ChatStreamState;
  set: ChatStreamSet;
}

const buildConversationTitle = (content: string) =>
  content.slice(0, 30) + (content.length > 30 ? '...' : '');

const createConversationRecord = (response: ChatConversationMutationResponse): Conversation => ({
  id: response.id,
  title: response.title,
  messages: [],
  steps: [],
  activeRun: null,
  createdAt: new Date(response.created_at).getTime(),
  updatedAt: new Date(response.updated_at).getTime(),
  isLoaded: true,
});

export const applyConversationDetailToState = (
  get: () => ChatStreamState,
  set: ChatStreamSet,
  id: number,
  detail: ChatConversationDetailResponse
) => {
  const mapped = preserveExistingProcessSteps(mapConversationDetail(detail), get().conversations[id]);
  set(() => ({
    conversations: {
      ...get().conversations,
      [id]: mapped,
    },
    isTyping: false,
    isStreaming: false,
    streamingConversationId: null,
    currentStage: mapped.activeRun?.current_stage ?? null,
    currentStageLabel: mapped.activeRun?.status === 'waiting_user' ? '需要你补充一点信息' : null,
    requiresUserInput: mapped.activeRun?.status === 'waiting_user',
    pendingQuestion: mapped.activeRun?.pending_question ?? null,
    runStatus: mapped.activeRun?.status ?? 'idle',
    isRecovering: false,
    reconnectAttempts: 0,
    lastHeartbeatAt: mapped.activeRun?.heartbeat_at ? new Date(mapped.activeRun.heartbeat_at).getTime() : null,
  }));
  return mapped;
};

export const resumeConversationStream = async ({
  applyConversationDetail,
  conversationId,
  get,
  run,
  set,
}: ResumeConversationParams) => {
  if (!run || !run.can_resume) return;
  if (get().streamingConversationId === conversationId && get().isStreaming) return;

  set((state) => ({
    isTyping: run.status === 'running',
    isStreaming: run.status === 'running',
    isRecovering: true,
    streamingConversationId: run.status === 'running' ? conversationId : null,
    currentStage: run.current_stage ?? null,
    currentStageLabel: run.status === 'waiting_user' ? '需要你补充一点信息' : '正在恢复回答',
    requiresUserInput: run.status === 'waiting_user',
    pendingQuestion: run.pending_question ?? null,
    runStatus: run.status,
    reconnectAttempts: state.reconnectAttempts + 1,
    lastHeartbeatAt: Date.now(),
    conversations: {
      ...state.conversations,
      [conversationId]: {
        ...state.conversations[conversationId],
        activeRun: run,
        messages: ensureResumableAssistantMessage(
          state.conversations[conversationId]?.messages || [],
          run,
          state.conversations[conversationId]?.steps || []
        ),
      },
    },
  }));

  let terminalEventReceived = false;

  try {
    await chatApi.resumeStream(conversationId, run.id, {
      onResume: ({ run_id, stage, status }) => {
        set((state) => ({
          lastHeartbeatAt: Date.now(),
          conversations: {
            ...state.conversations,
            [conversationId]: {
              ...state.conversations[conversationId],
              activeRun: state.conversations[conversationId]?.activeRun
                ? {
                    ...state.conversations[conversationId].activeRun!,
                    id: run_id,
                    current_stage: stage ?? state.conversations[conversationId].activeRun!.current_stage,
                    status: status ?? state.conversations[conversationId].activeRun!.status,
                  }
                : run,
            },
          },
        }));
      },
      onSnapshot: ({ run_id, stage, status, message }) => {
        set((state) => ({
          lastHeartbeatAt: Date.now(),
          conversations: {
            ...state.conversations,
            [conversationId]: {
              ...state.conversations[conversationId],
              messages: ensureResumableAssistantMessage(
                state.conversations[conversationId]?.messages || [],
                {
                  ...(state.conversations[conversationId]?.activeRun || run),
                  id: run_id,
                  current_stage: stage ?? state.conversations[conversationId]?.activeRun?.current_stage ?? run.current_stage,
                  status: status ?? state.conversations[conversationId]?.activeRun?.status ?? run.status,
                  last_answer: message ?? state.conversations[conversationId]?.activeRun?.last_answer ?? run.last_answer,
                },
                state.conversations[conversationId]?.steps || []
              ),
              activeRun: {
                ...(state.conversations[conversationId]?.activeRun || run),
                id: run_id,
                current_stage: stage ?? state.conversations[conversationId]?.activeRun?.current_stage ?? run.current_stage,
                status: status ?? state.conversations[conversationId]?.activeRun?.status ?? run.status,
                last_answer: message ?? state.conversations[conversationId]?.activeRun?.last_answer ?? run.last_answer,
              },
              updatedAt: Date.now(),
            },
          },
        }));
      },
      onHeartbeat: ({ stage, status }) => {
        set({
          lastHeartbeatAt: Date.now(),
          currentStage: stage ?? get().currentStage,
          runStatus: status ?? get().runStatus,
        });
      },
      onStage: ({ stage, label }) => {
        set({
          currentStage: stage,
          currentStageLabel: label ?? null,
          runStatus: 'running',
          lastHeartbeatAt: Date.now(),
        });
      },
      onQuestion: ({ message }) => {
        terminalEventReceived = true;
        set((state) => ({
          isTyping: false,
          isStreaming: false,
          isRecovering: false,
          streamingConversationId: null,
          currentStage: 'clarifying',
          currentStageLabel: '需要你补充一点信息',
          requiresUserInput: true,
          pendingQuestion: message,
          runStatus: 'waiting_user',
          conversations: {
            ...state.conversations,
            [conversationId]: {
              ...state.conversations[conversationId],
              activeRun: state.conversations[conversationId]?.activeRun
                ? {
                    ...state.conversations[conversationId].activeRun!,
                    status: 'waiting_user',
                    pending_question: message,
                    last_answer: message,
                  }
                : run,
              messages: ensureResumableAssistantMessage(
                state.conversations[conversationId]?.messages || [],
                {
                  ...(state.conversations[conversationId]?.activeRun || run),
                  status: 'waiting_user',
                  pending_question: message,
                  last_answer: message,
                },
                state.conversations[conversationId]?.steps || []
              ),
            },
          },
        }));
      },
      onStep: (event) => {
        const nextStep = stepEventToStep(event);
        set((state) => {
          const currentSteps = state.conversations[conversationId]?.steps || [];
          const processSteps = upsertStep(currentSteps, nextStep);

          return {
            lastHeartbeatAt: Date.now(),
            conversations: {
              ...state.conversations,
              [conversationId]: {
                ...state.conversations[conversationId],
                steps: processSteps,
                messages: ensureResumableAssistantMessage(
                  (state.conversations[conversationId]?.messages || []).map((message) =>
                    message.runId === run.id || message.id === `resume-${run.id}` ? { ...message, processSteps } : message
                  ),
                  state.conversations[conversationId]?.activeRun || run,
                  processSteps
                ),
              },
            },
          };
        });
      },
      onChunk: ({ message, content }) => {
        const snapshot = message ?? content ?? '';
        set((state) => ({
          lastHeartbeatAt: Date.now(),
          conversations: {
            ...state.conversations,
            [conversationId]: {
              ...state.conversations[conversationId],
              activeRun: state.conversations[conversationId]?.activeRun
                ? { ...state.conversations[conversationId].activeRun!, last_answer: snapshot, status: 'running' }
                : run,
              messages: ensureResumableAssistantMessage(
                state.conversations[conversationId]?.messages || [],
                {
                  ...(state.conversations[conversationId]?.activeRun || run),
                  last_answer: snapshot,
                  status: 'running',
                },
                state.conversations[conversationId]?.steps || []
              ),
              updatedAt: Date.now(),
            },
          },
        }));
      },
      onDone: async () => {
        terminalEventReceived = true;
        try {
          const detail = await chatApi.getConversation(conversationId);
          applyConversationDetail(conversationId, detail);
        } finally {
          set({
            isTyping: false,
            isStreaming: false,
            isRecovering: false,
            streamingConversationId: null,
            currentStage: 'completed',
            currentStageLabel: '回答已生成',
            requiresUserInput: false,
            pendingQuestion: null,
            runStatus: 'completed',
            reconnectAttempts: 0,
          });
        }
      },
      onError: ({ error, message }) => {
        terminalEventReceived = true;
        const finalMessage = error || message || '恢复连接失败，请稍后再试。';
        set((state) => ({
          isTyping: false,
          isStreaming: false,
          isRecovering: false,
          streamingConversationId: null,
          currentStage: 'failed',
          currentStageLabel: '恢复失败',
          requiresUserInput: false,
          pendingQuestion: null,
          runStatus: 'failed',
          conversations: {
            ...state.conversations,
            [conversationId]: {
              ...state.conversations[conversationId],
              activeRun: state.conversations[conversationId]?.activeRun
                ? { ...state.conversations[conversationId].activeRun!, status: 'failed' }
                : run,
              messages: ensureResumableAssistantMessage(
                state.conversations[conversationId]?.messages || [],
                {
                  ...(state.conversations[conversationId]?.activeRun || run),
                  status: 'failed',
                  last_answer: finalMessage,
                },
                state.conversations[conversationId]?.steps || []
              ),
            },
          },
        }));
      },
    });

    if (!terminalEventReceived && get().streamingConversationId === conversationId && get().isRecovering) {
      set((state) => ({
        isTyping: false,
        isStreaming: false,
        isRecovering: false,
        streamingConversationId: null,
        currentStage: 'failed',
        currentStageLabel: '恢复连接已断开',
        requiresUserInput: false,
        pendingQuestion: null,
        runStatus: 'failed',
        conversations: {
          ...state.conversations,
          [conversationId]: {
            ...state.conversations[conversationId],
            activeRun: state.conversations[conversationId]?.activeRun
              ? { ...state.conversations[conversationId].activeRun!, status: 'failed' }
              : run,
          },
        },
      }));
    }
  } catch (error) {
    console.error('Failed to resume conversation stream:', error);
    const attempts = get().reconnectAttempts;
    if (attempts < 3) {
      window.setTimeout(() => {
        const nextRun = get().conversations[conversationId]?.activeRun ?? run;
        if (nextRun?.status === 'running') {
          void resumeConversationStream({ applyConversationDetail, conversationId, get, run: nextRun, set });
        }
      }, 1200);
      return;
    }

    set({
      isTyping: false,
      isStreaming: false,
      isRecovering: false,
      streamingConversationId: null,
      currentStage: 'failed',
      currentStageLabel: '连接已断开，可重新进入会话恢复',
      runStatus: 'failed',
    });
  }
};

export const streamConversationMessage = async ({
  applyConversationDetail,
  content,
  get,
  set,
}: StreamConversationMessageParams) => {
  const { activeId, conversations } = get();
  let currentId = activeId;

  if (!currentId) {
    try {
      const response = await chatApi.createConversation(buildConversationTitle(content));
      currentId = response.id;
      const newChat = createConversationRecord(response);

      persistActiveChatId(currentId);
      set((state) => ({
        conversations: { ...state.conversations, [currentId!]: newChat },
        activeId: currentId,
      }));
    } catch (error) {
      console.error('Failed to create conversation:', error);
      return;
    }
  }

  const currentConversation = conversations[currentId] || get().conversations[currentId];
  const isFirstMessage = !currentConversation || currentConversation.messages.length === 0;
  const nextTitle = isFirstMessage ? buildConversationTitle(content) : currentConversation.title;

  const userMessage: ChatMessage = {
    id: Date.now().toString(),
    role: 'user',
    content,
    timestamp: Date.now(),
  };

  const assistantMessageId = (Date.now() + 1).toString();
  const assistantPlaceholder: ChatMessage = {
    id: assistantMessageId,
    role: 'assistant',
    content: '',
    timestamp: Date.now(),
  };

  set((state) => ({
    conversations: {
      ...state.conversations,
      [currentId]: {
        ...state.conversations[currentId],
        title: nextTitle,
        messages: [...(state.conversations[currentId]?.messages || []), userMessage, assistantPlaceholder],
        steps: [],
        activeRun: null,
        updatedAt: Date.now(),
        isLoaded: true,
      },
    },
    isTyping: true,
    isStreaming: true,
    isRecovering: false,
    reconnectAttempts: 0,
    streamingConversationId: currentId,
    currentStage: 'analyzing',
    currentStageLabel: '正在理解你的问题',
    requiresUserInput: false,
    pendingQuestion: null,
    runStatus: 'running',
    lastHeartbeatAt: Date.now(),
  }));

  try {
    const reqModel = get().selectedModel;
    const requestPayload: ChatRequest = {
      message: content,
      conversation_id: currentId,
      ...(reqModel ? { model_provider: reqModel.provider, model_name: reqModel.model_name } : {}),
    };

    await chatApi.streamMessage(requestPayload, {
      onResume: ({ run_id, stage, status }) => {
        set((state) => ({
          lastHeartbeatAt: Date.now(),
          conversations: {
            ...state.conversations,
            [currentId]: {
              ...state.conversations[currentId],
              activeRun: {
                id: run_id,
                status: status ?? 'running',
                current_stage: stage ?? 'analyzing',
                last_answer: '',
                can_resume: true,
              },
            },
          },
        }));
      },
      onSnapshot: ({ run_id, stage, status, message }) => {
        set((state) => ({
          lastHeartbeatAt: Date.now(),
          conversations: {
            ...state.conversations,
            [currentId]: {
              ...state.conversations[currentId],
              activeRun: {
                id: run_id,
                status: status ?? 'running',
                current_stage: stage ?? 'analyzing',
                last_answer: message ?? '',
                can_resume: true,
              },
              messages: state.conversations[currentId].messages.map((msg) =>
                msg.id === assistantMessageId ? { ...msg, content: message ?? msg.content, runId: run_id } : msg
              ),
            },
          },
        }));
      },
      onHeartbeat: () => {
        set({ lastHeartbeatAt: Date.now() });
      },
      onStage: ({ stage, label }) => {
        set({
          currentStage: stage,
          currentStageLabel: label ?? null,
          runStatus: stage === 'completed' ? 'completed' : 'running',
          requiresUserInput: false,
          lastHeartbeatAt: Date.now(),
        });
      },
      onQuestion: ({ run_id, message }) => {
        set((state) => ({
          currentStage: 'clarifying',
          currentStageLabel: '需要你补充一点信息',
          requiresUserInput: true,
          pendingQuestion: message,
          runStatus: 'waiting_user',
          isTyping: false,
          isStreaming: false,
          streamingConversationId: null,
          conversations: {
            ...state.conversations,
            [currentId]: {
              ...state.conversations[currentId],
              title: nextTitle,
              activeRun: state.conversations[currentId].activeRun
                ? { ...state.conversations[currentId].activeRun!, status: 'waiting_user', pending_question: message, last_answer: message }
                : null,
              messages: state.conversations[currentId].messages.map((msg) =>
                msg.id === assistantMessageId ? { ...msg, content: message, runId: run_id ?? msg.runId } : msg
              ),
              steps: state.conversations[currentId].steps,
              updatedAt: Date.now(),
              isLoaded: true,
            },
          },
        }));
      },
      onStep: (event) => {
        const nextStep = stepEventToStep(event);
        set((state) => {
          const currentSteps =
            state.conversations[currentId]?.messages.find((msg) => msg.id === assistantMessageId)?.processSteps || [];
          const processSteps = upsertStep(currentSteps, nextStep);

          return {
            lastHeartbeatAt: Date.now(),
            conversations: {
              ...state.conversations,
              [currentId]: {
                ...state.conversations[currentId],
                title: nextTitle,
                messages: state.conversations[currentId].messages.map((msg) =>
                  msg.id === assistantMessageId ? { ...msg, processSteps } : msg
                ),
                steps: processSteps,
                updatedAt: Date.now(),
                isLoaded: true,
              },
            },
          };
        });
      },
      onChunk: ({ run_id, message, content: chunkContent }) => {
        const snapshot = message ?? chunkContent ?? '';
        set((state) => ({
          lastHeartbeatAt: Date.now(),
          conversations: {
            ...state.conversations,
            [currentId]: {
              ...state.conversations[currentId],
              title: nextTitle,
              activeRun: state.conversations[currentId].activeRun
                ? { ...state.conversations[currentId].activeRun!, last_answer: snapshot, status: 'running' }
                : null,
              messages: state.conversations[currentId].messages.map((msg) =>
                msg.id === assistantMessageId ? { ...msg, content: snapshot, runId: run_id ?? msg.runId } : msg
              ),
              steps: state.conversations[currentId].steps,
              updatedAt: Date.now(),
              isLoaded: true,
            },
          },
        }));
      },
      onDone: async () => {
        try {
          const detail = await chatApi.getConversation(currentId);
          applyConversationDetail(currentId, detail);
        } finally {
          set({
            isTyping: false,
            isStreaming: false,
            streamingConversationId: null,
            currentStage: 'completed',
            currentStageLabel: '回答已生成',
            requiresUserInput: false,
            pendingQuestion: null,
            runStatus: 'completed',
            reconnectAttempts: 0,
          });
        }
      },
      onError: ({ error, message }) => {
        const finalMessage = error || message || '生成回答失败，请稍后再试。';
        set((state) => ({
          conversations: {
            ...state.conversations,
            [currentId]: {
              ...state.conversations[currentId],
              activeRun: state.conversations[currentId].activeRun
                ? { ...state.conversations[currentId].activeRun!, status: 'failed', last_answer: finalMessage }
                : null,
              messages: state.conversations[currentId].messages.map((msg) =>
                msg.id === assistantMessageId ? { ...msg, content: msg.content || finalMessage } : msg
              ),
              steps: state.conversations[currentId].steps,
              updatedAt: Date.now(),
              isLoaded: true,
            },
          },
          isTyping: false,
          isStreaming: false,
          streamingConversationId: null,
          currentStage: 'failed',
          currentStageLabel: '本次执行失败',
          requiresUserInput: false,
          pendingQuestion: null,
          runStatus: 'failed',
        }));
      },
    });
  } catch (error) {
    const resumableRun = get().conversations[currentId]?.activeRun;
    if (resumableRun?.can_resume) {
      void resumeConversationStream({ applyConversationDetail, conversationId: currentId, get, run: resumableRun, set });
      return;
    }

    set((state) => ({
      conversations: {
        ...state.conversations,
        [currentId]: {
          ...state.conversations[currentId],
          messages: state.conversations[currentId].messages.map((msg) =>
            msg.id === assistantMessageId ? { ...msg, content: '生成回答失败，请稍后再试。' } : msg
          ),
          steps: state.conversations[currentId].steps,
          updatedAt: Date.now(),
          isLoaded: true,
        },
      },
      isTyping: false,
      isStreaming: false,
      streamingConversationId: null,
      currentStage: 'failed',
      currentStageLabel: '本次执行失败',
      requiresUserInput: false,
      pendingQuestion: null,
      runStatus: 'failed',
    }));
  }
};
