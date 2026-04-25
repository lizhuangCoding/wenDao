import { create } from 'zustand';
import type { ChatActiveRun, ChatConversationDetailResponse, ChatMessage, ChatStage, ChatStep, ChatStepEvent } from '@/types';
import { chatApi } from '@/api';

interface Conversation {
  id: number;
  title: string;
  messages: ChatMessage[];
  steps: ChatStep[];
  activeRun: ChatActiveRun | null;
  createdAt: number;
  updatedAt: number;
  isLoaded: boolean;
}

interface ChatState {
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

  loadConversations: () => Promise<void>;
  createNewChat: () => Promise<void>;
  setActiveChat: (id: number) => Promise<void>;
  deleteChat: (id: number) => Promise<void>;
  renameChat: (id: number, title: string) => Promise<void>;
  sendMessage: (content: string) => Promise<void>;
  clearMessages: () => void;
}

const normalizeStep = (step: any): ChatStep => ({
  id: Number(step.id ?? step.step_id ?? 0),
  run_id: step.run_id ? Number(step.run_id) : undefined,
  agent_name: step.agent_name ?? step.agentName ?? 'Agent',
  type: step.type ?? 'thinking',
  summary: step.summary ?? '',
  detail: step.detail ?? '',
  status: step.status ?? 'running',
  created_at: step.created_at ?? new Date().toISOString(),
});

const mapSteps = (steps: any[] = []): ChatStep[] => steps.map(normalizeStep);

const attachStepsToLatestAssistant = (messages: ChatMessage[], steps: ChatStep[]): ChatMessage[] => {
  if (steps.length === 0) return messages;
  const latestAssistantIndex = [...messages].reverse().findIndex((m) => m.role === 'assistant');
  if (latestAssistantIndex === -1) return messages;
  const targetIndex = messages.length - 1 - latestAssistantIndex;
  return messages.map((message, index) =>
    index === targetIndex ? { ...message, processSteps: steps } : message
  );
};

const mapMessages = (messages: any[] = [], steps: any[] = []): ChatMessage[] => {
  const mapped = messages.map((m: any) => ({
    id: String(m.id),
    role: m.role as 'user' | 'assistant',
    content: m.content,
    timestamp: new Date(m.created_at).getTime(),
  }));
  return attachStepsToLatestAssistant(mapped, mapSteps(steps));
};

const stepEventToStep = (event: ChatStepEvent): ChatStep => ({
  id: Number(event.step_id || 0),
  agent_name: event.agent_name || 'Agent',
  type: 'thinking',
  summary: event.summary || '',
  detail: event.detail || '',
  status: event.status || 'running',
  created_at: new Date().toISOString(),
});

const upsertStep = (steps: ChatStep[] = [], next: ChatStep): ChatStep[] => {
  const key = next.id > 0 ? `id:${next.id}` : `agent:${next.agent_name}:${next.summary}`;
  const index = steps.findIndex((step) => {
    const stepKey = step.id > 0 ? `id:${step.id}` : `agent:${step.agent_name}:${step.summary}`;
    return stepKey === key;
  });
  if (index === -1) return [...steps, next];
  return steps.map((step, i) => (i === index ? { ...step, ...next } : step));
};

const mapActiveRun = (run?: ChatConversationDetailResponse['active_run']): ChatActiveRun | null => {
  if (!run) return null;
  if (!run.can_resume || (run.status !== 'running' && run.status !== 'waiting_user')) return null;
  return {
    id: Number(run.id),
    status: run.status,
    current_stage: run.current_stage,
    pending_question: run.pending_question,
    last_answer: run.last_answer ?? '',
    heartbeat_at: run.heartbeat_at,
    can_resume: Boolean(run.can_resume),
  };
};

const ensureResumableAssistantMessage = (
  messages: ChatMessage[],
  activeRun: ChatActiveRun | null,
  activeSteps: ChatStep[] = []
): ChatMessage[] => {
  if (!activeRun) return messages;
  const content = activeRun.pending_question ?? activeRun.last_answer ?? '';
  const latestAssistantIndex = [...messages].reverse().findIndex((m) => m.role === 'assistant');
  if (latestAssistantIndex !== -1) {
    const targetIndex = messages.length - 1 - latestAssistantIndex;
    return messages.map((message, index) =>
      index === targetIndex
        ? {
            ...message,
            content: content || message.content,
            processSteps: activeSteps.length ? activeSteps : message.processSteps,
            runId: activeRun.id,
          }
        : message
    );
  }
  return [
    ...messages,
    {
      id: `resume-${activeRun.id}`,
      role: 'assistant',
      content,
      timestamp: Date.now(),
      processSteps: activeSteps,
      runId: activeRun.id,
    },
  ];
};

const mapConversationDetail = (detail: ChatConversationDetailResponse): Conversation => {
  const steps = mapSteps(detail.steps ?? []);
  const activeSteps = mapSteps(detail.active_steps ?? []);
  const activeRun = mapActiveRun(detail.active_run);
  const messages = ensureResumableAssistantMessage(mapMessages(detail.messages, detail.steps ?? []), activeRun, activeSteps);

  return {
    id: detail.conversation.id,
    title: detail.conversation.title,
    messages,
    steps: activeSteps.length ? activeSteps : steps,
    activeRun,
    createdAt: new Date(detail.conversation.created_at).getTime(),
    updatedAt: new Date(detail.conversation.updated_at).getTime(),
    isLoaded: true,
  };
};

const ACTIVE_CHAT_STORAGE_KEY = 'wendao.aiChat.activeId';

const readStoredActiveId = () => {
  if (typeof window === 'undefined') return null;
  const raw = window.localStorage.getItem(ACTIVE_CHAT_STORAGE_KEY);
  if (!raw) return null;
  const parsed = Number(raw);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
};

const persistActiveChatId = (id: number | null) => {
  if (typeof window === 'undefined') return;
  if (id === null) {
    window.localStorage.removeItem(ACTIVE_CHAT_STORAGE_KEY);
    return;
  }
  window.localStorage.setItem(ACTIVE_CHAT_STORAGE_KEY, String(id));
};

export const useChatStore = create<ChatState>()((set, get) => {
  const applyConversationDetail = (id: number, detail: ChatConversationDetailResponse) => {
    const mapped = mapConversationDetail(detail);
    set((state) => ({
      conversations: {
        ...state.conversations,
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

  const resumeConversation = async (conversationId: number, run: ChatActiveRun | null) => {
    if (!run || !run.can_resume) return;
    if (get().streamingConversationId === conversationId && get().isStreaming) return;

    set((state) => ({
      isTyping: run.status === 'running',
      isStreaming: run.status === 'running',
      isRecovering: true,
      streamingConversationId: run.status === 'running' ? conversationId : null,
      currentStage: (run.current_stage as ChatStage) ?? null,
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
          messages: ensureResumableAssistantMessage(state.conversations[conversationId]?.messages || [], run, state.conversations[conversationId]?.steps || []),
        },
      },
    }));

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
                      current_stage: (stage as ChatStage) ?? state.conversations[conversationId].activeRun!.current_stage,
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
                    current_stage: ((stage as ChatStage) ?? state.conversations[conversationId]?.activeRun?.current_stage ?? run.current_stage),
                    status: status ?? state.conversations[conversationId]?.activeRun?.status ?? run.status,
                    last_answer: message ?? state.conversations[conversationId]?.activeRun?.last_answer ?? run.last_answer,
                  },
                  state.conversations[conversationId]?.steps || []
                ),
                activeRun: {
                  ...(state.conversations[conversationId]?.activeRun || run),
                  id: run_id,
                  current_stage: ((stage as ChatStage) ?? state.conversations[conversationId]?.activeRun?.current_stage ?? run.current_stage),
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
            currentStage: (stage as ChatStage) ?? get().currentStage,
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
                    (state.conversations[conversationId]?.messages || []).map((msg) =>
                      msg.runId === run.id || msg.id === `resume-${run.id}` ? { ...msg, processSteps } : msg
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
    } catch (error) {
      console.error('Failed to resume conversation stream:', error);
      const attempts = get().reconnectAttempts;
      if (attempts < 3) {
        window.setTimeout(() => {
          const nextRun = get().conversations[conversationId]?.activeRun ?? run;
          if (nextRun?.status === 'running') {
            void resumeConversation(conversationId, nextRun);
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

  return {
    conversations: {},
    activeId: readStoredActiveId(),
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

    loadConversations: async () => {
      try {
        const convs = await chatApi.getConversations();
        const conversations: Record<number, Conversation> = {};

        for (const conv of convs) {
          conversations[conv.id] = {
            id: conv.id,
            title: conv.title,
            messages: [],
            steps: [],
            activeRun: null,
            createdAt: new Date(conv.created_at).getTime(),
            updatedAt: new Date(conv.updated_at).getTime(),
            isLoaded: false,
          };
        }

        const currentActiveId = get().activeId ?? readStoredActiveId();
        let nextActiveId = currentActiveId;
        if (currentActiveId && conversations[currentActiveId]) {
          try {
            const detail = await chatApi.getConversation(currentActiveId);
            conversations[currentActiveId] = mapConversationDetail(detail);
          } catch {
            nextActiveId = null;
          }
        }

        set({ conversations, activeId: nextActiveId });
        persistActiveChatId(nextActiveId);

        if (nextActiveId && conversations[nextActiveId]?.activeRun?.can_resume) {
          void resumeConversation(nextActiveId, conversations[nextActiveId].activeRun);
        }
      } catch (error) {
        console.error('Failed to load conversations:', error);
      }
    },

    createNewChat: async () => {
      try {
        const response = await chatApi.createConversation('New Conversation');
        const newChat: Conversation = {
          id: response.id,
          title: response.title,
          messages: [],
          steps: [],
          activeRun: null,
          createdAt: new Date(response.created_at).getTime(),
          updatedAt: new Date(response.updated_at).getTime(),
          isLoaded: true,
        };
        persistActiveChatId(response.id);
        set((state) => ({
          conversations: { ...state.conversations, [response.id]: newChat },
          activeId: response.id,
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
        }));
      } catch (error) {
        console.error('Failed to create conversation:', error);
      }
    },

    setActiveChat: async (id) => {
      persistActiveChatId(id);
      set({
        activeId: id,
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
      try {
        const detail = await chatApi.getConversation(id);
        const mapped = applyConversationDetail(id, detail);
        if (mapped.activeRun?.can_resume) {
          void resumeConversation(id, mapped.activeRun);
        }
      } catch (error) {
        console.error('Failed to load conversation detail:', error);
      }
    },

    deleteChat: async (id) => {
      try {
        await chatApi.deleteConversation(id);
        set((state) => {
          const newConversations = { ...state.conversations };
          delete newConversations[id];
          const nextActiveId = state.activeId === id ? null : state.activeId;
          persistActiveChatId(nextActiveId);
          return {
            conversations: newConversations,
            activeId: nextActiveId,
          };
        });
      } catch (error) {
        console.error('Failed to delete conversation:', error);
      }
    },

    renameChat: async (id, title) => {
      try {
        const response = await chatApi.renameConversation(id, title);
        set((state) => ({
          conversations: {
            ...state.conversations,
            [id]: {
              ...state.conversations[id],
              title: response.title,
              updatedAt: new Date(response.updated_at).getTime(),
            },
          },
        }));
      } catch (error) {
        console.error('Failed to rename conversation:', error);
        throw error;
      }
    },

    sendMessage: async (content) => {
      const { activeId, conversations } = get();
      let currentId = activeId;

      if (!currentId) {
        try {
          const response = await chatApi.createConversation(content.slice(0, 30) + (content.length > 30 ? '...' : ''));
          currentId = response.id;
          const newChat: Conversation = {
            id: currentId,
            title: response.title,
            messages: [],
            steps: [],
            activeRun: null,
            createdAt: new Date(response.created_at).getTime(),
            updatedAt: new Date(response.updated_at).getTime(),
            isLoaded: true,
          };
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

      const currentConversation = conversations[currentId!] || get().conversations[currentId!];
      const isFirstMessage = !currentConversation || currentConversation.messages.length === 0;
      const nextTitle = isFirstMessage ? content.slice(0, 30) + (content.length > 30 ? '...' : '') : currentConversation.title;

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
          [currentId!]: {
            ...state.conversations[currentId!],
            title: nextTitle,
            messages: [...(state.conversations[currentId!]?.messages || []), userMessage, assistantPlaceholder],
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
        streamingConversationId: currentId!,
        currentStage: 'analyzing',
        currentStageLabel: '正在理解你的问题',
        requiresUserInput: false,
        pendingQuestion: null,
        runStatus: 'running',
        lastHeartbeatAt: Date.now(),
      }));

      try {
        await chatApi.streamMessage(
          { message: content, conversation_id: currentId },
          {
            onResume: ({ run_id, stage, status }) => {
              set((state) => ({
                lastHeartbeatAt: Date.now(),
                conversations: {
                  ...state.conversations,
                  [currentId!]: {
                    ...state.conversations[currentId!],
                    activeRun: {
                      id: run_id,
                      status: status ?? 'running',
                      current_stage: ((stage as ChatStage) ?? 'analyzing'),
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
                  [currentId!]: {
                    ...state.conversations[currentId!],
                    activeRun: {
                      id: run_id,
                      status: status ?? 'running',
                      current_stage: ((stage as ChatStage) ?? 'analyzing'),
                      last_answer: message ?? '',
                      can_resume: true,
                    },
                    messages: state.conversations[currentId!].messages.map((msg) =>
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
            onQuestion: ({ message }) => {
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
                  [currentId!]: {
                    ...state.conversations[currentId!],
                    title: nextTitle,
                    activeRun: state.conversations[currentId!].activeRun
                      ? { ...state.conversations[currentId!].activeRun!, status: 'waiting_user', pending_question: message, last_answer: message }
                      : null,
                    messages: state.conversations[currentId!].messages.map((msg) =>
                      msg.id === assistantMessageId ? { ...msg, content: message } : msg
                    ),
                    steps: state.conversations[currentId!].steps,
                    updatedAt: Date.now(),
                    isLoaded: true,
                  },
                },
              }));
            },
            onStep: (event) => {
              const nextStep = stepEventToStep(event);
              set((state) => {
                const currentSteps = state.conversations[currentId!]?.messages.find((msg) => msg.id === assistantMessageId)?.processSteps || [];
                const processSteps = upsertStep(currentSteps, nextStep);
                return {
                  lastHeartbeatAt: Date.now(),
                  conversations: {
                    ...state.conversations,
                    [currentId!]: {
                      ...state.conversations[currentId!],
                      title: nextTitle,
                      messages: state.conversations[currentId!].messages.map((msg) =>
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
            onChunk: ({ message, content: chunkContent }) => {
              const snapshot = message ?? chunkContent ?? '';
              set((state) => ({
                lastHeartbeatAt: Date.now(),
                conversations: {
                  ...state.conversations,
                  [currentId!]: {
                    ...state.conversations[currentId!],
                    title: nextTitle,
                    activeRun: state.conversations[currentId!].activeRun
                      ? { ...state.conversations[currentId!].activeRun!, last_answer: snapshot, status: 'running' }
                      : null,
                    messages: state.conversations[currentId!].messages.map((msg) =>
                      msg.id === assistantMessageId ? { ...msg, content: snapshot } : msg
                    ),
                    steps: state.conversations[currentId!].steps,
                    updatedAt: Date.now(),
                    isLoaded: true,
                  },
                },
              }));
            },
            onDone: async () => {
              try {
                const detail = await chatApi.getConversation(currentId!);
                applyConversationDetail(currentId!, detail);
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
                  [currentId!]: {
                    ...state.conversations[currentId!],
                    activeRun: state.conversations[currentId!].activeRun
                      ? { ...state.conversations[currentId!].activeRun!, status: 'failed', last_answer: finalMessage }
                      : null,
                    messages: state.conversations[currentId!].messages.map((msg) =>
                      msg.id === assistantMessageId ? { ...msg, content: msg.content || finalMessage } : msg
                    ),
                    steps: state.conversations[currentId!].steps,
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
          }
        );
      } catch (error) {
        const resumableRun = get().conversations[currentId!]?.activeRun;
        if (resumableRun?.can_resume) {
          void resumeConversation(currentId!, resumableRun);
          return;
        }
        set((state) => ({
          conversations: {
            ...state.conversations,
            [currentId!]: {
              ...state.conversations[currentId!],
              messages: state.conversations[currentId!].messages.map((msg) =>
                msg.id === assistantMessageId ? { ...msg, content: '生成回答失败，请稍后再试。' } : msg
              ),
              steps: state.conversations[currentId!].steps,
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
    },

    clearMessages: () => {
      const { activeId } = get();
      if (!activeId) return;
      set((state) => ({
        conversations: {
          ...state.conversations,
          [activeId]: { ...state.conversations[activeId], messages: [], steps: [], activeRun: null },
        },
        currentStage: null,
        currentStageLabel: null,
        requiresUserInput: false,
        pendingQuestion: null,
        runStatus: 'idle',
      }));
    },
  };
});
