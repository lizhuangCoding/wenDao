import { create } from 'zustand';
import type { ChatMessage, ChatStage, ChatStep, ChatStepEvent } from '@/types';
import { chatApi } from '@/api';

interface Conversation {
  id: number;
  title: string;
  messages: ChatMessage[];
  steps: ChatStep[];
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

export const useChatStore = create<ChatState>()((set, get) => ({
  conversations: {},
  activeId: null,
  isTyping: false,
  isStreaming: false,
  streamingConversationId: null,
  currentStage: null,
  currentStageLabel: null,
  requiresUserInput: false,
  pendingQuestion: null,
  runStatus: 'idle',

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
          createdAt: new Date(conv.created_at).getTime(),
          updatedAt: new Date(conv.updated_at).getTime(),
          isLoaded: false,
        };
      }

      const currentActiveId = get().activeId;
      let nextActiveId = currentActiveId;
      if (currentActiveId && conversations[currentActiveId]) {
        try {
          const detail = await chatApi.getConversation(currentActiveId);
          conversations[currentActiveId] = {
            id: detail.conversation.id,
            title: detail.conversation.title,
            messages: mapMessages(detail.messages, detail.steps),
            steps: mapSteps(detail.steps),
            createdAt: new Date(detail.conversation.created_at).getTime(),
            updatedAt: new Date(detail.conversation.updated_at).getTime(),
            isLoaded: true,
          };
        } catch {
          nextActiveId = null;
        }
      }

      set({ conversations, activeId: nextActiveId });
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
        createdAt: new Date(response.created_at).getTime(),
        updatedAt: new Date(response.updated_at).getTime(),
        isLoaded: true,
      };
      set((state) => ({
        conversations: { ...state.conversations, [response.id]: newChat },
        activeId: response.id,
      }));
    } catch (error) {
      console.error('Failed to create conversation:', error);
    }
  },

  setActiveChat: async (id) => {
    set({ activeId: id });
    try {
      const detail = await chatApi.getConversation(id);
      set((state) => ({
        conversations: {
          ...state.conversations,
          [id]: {
            id: detail.conversation.id,
            title: detail.conversation.title,
            messages: mapMessages(detail.messages, detail.steps),
            steps: mapSteps(detail.steps),
            createdAt: new Date(detail.conversation.created_at).getTime(),
            updatedAt: new Date(detail.conversation.updated_at).getTime(),
            isLoaded: true,
          },
        },
      }));
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
        return {
          conversations: newConversations,
          activeId: state.activeId === id ? null : state.activeId,
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
          createdAt: new Date(response.created_at).getTime(),
          updatedAt: new Date(response.updated_at).getTime(),
          isLoaded: true,
        };
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
          steps: state.conversations[currentId!]?.steps || [],
          updatedAt: Date.now(),
          isLoaded: true,
        },
      },
      isTyping: true,
      isStreaming: true,
      streamingConversationId: currentId!,
      currentStage: 'analyzing',
      currentStageLabel: '正在理解你的问题',
      requiresUserInput: false,
      pendingQuestion: null,
      runStatus: 'running',
    }));

    try {
      await chatApi.streamMessage(
        { message: content, conversation_id: currentId },
        {
          onStage: ({ stage, label }) => {
            set({
              currentStage: stage,
              currentStageLabel: label ?? null,
              runStatus: stage === 'completed' ? 'completed' : 'running',
              requiresUserInput: false,
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
              conversations: {
                ...state.conversations,
                [currentId!]: {
                  ...state.conversations[currentId!],
                  title: nextTitle,
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
              set((state) => ({
                conversations: {
                  ...state.conversations,
                  [currentId!]: {
                    id: detail.conversation.id,
                    title: detail.conversation.title,
                    messages: mapMessages(detail.messages, detail.steps),
                    steps: mapSteps(detail.steps),
                    createdAt: new Date(detail.conversation.created_at).getTime(),
                    updatedAt: new Date(detail.conversation.updated_at).getTime(),
                    isLoaded: true,
                  },
                },
                isTyping: false,
                isStreaming: false,
                streamingConversationId: null,
                currentStage: 'completed',
                currentStageLabel: '回答已生成',
                requiresUserInput: false,
                pendingQuestion: null,
                runStatus: 'completed',
              }));
            } catch {
              set({
                isTyping: false,
                isStreaming: false,
                streamingConversationId: null,
                currentStage: 'completed',
                currentStageLabel: '回答已生成',
                requiresUserInput: false,
                pendingQuestion: null,
                runStatus: 'completed',
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
        [activeId]: { ...state.conversations[activeId], messages: [], steps: [] },
      },
      currentStage: null,
      currentStageLabel: null,
      requiresUserInput: false,
      pendingQuestion: null,
      runStatus: 'idle',
    }));
  },
}));
