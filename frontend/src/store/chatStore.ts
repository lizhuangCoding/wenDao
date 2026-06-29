import { create } from 'zustand';
import type { ChatConversationDetailResponse, ChatStage } from '@/types';
import { chatApi } from '@/api';
import type { Conversation } from './chatNormalizers';
import {
  persistActiveChatId,
  persistSelectedModel,
  readStoredActiveId,
  readStoredModel,
  type SelectedChatModel,
} from './chatPersistence';
import {
  applyConversationDetailToState,
  resumeConversationStream,
  streamConversationMessage,
} from './chatStream';

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
  selectedModel: SelectedChatModel | null;

  setSelectedModel: (model: SelectedChatModel | null) => void;
  loadConversations: () => Promise<void>;
  createNewChat: () => Promise<void>;
  setActiveChat: (id: number) => Promise<void>;
  deleteChat: (id: number) => Promise<void>;
  renameChat: (id: number, title: string) => Promise<void>;
  updateConversationShare: (id: number, isShared: boolean, shareToken?: string) => void;
  sendMessage: (content: string) => Promise<void>;
  clearMessages: () => void;
}

export const useChatStore = create<ChatState>()((set, get) => {
  const applyConversationDetail = (id: number, detail: ChatConversationDetailResponse) => {
    return applyConversationDetailToState(get, set, id, detail);
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
    selectedModel: readStoredModel(),

    setSelectedModel: (model) => {
      persistSelectedModel(model);
      set({ selectedModel: model });
    },

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
            isShared: conv.is_shared,
            shareToken: conv.share_token,
          };
        }

        const currentActiveId = get().activeId ?? readStoredActiveId();
        let nextActiveId = currentActiveId;

        if (currentActiveId && conversations[currentActiveId]) {
          try {
            const detail = await chatApi.getConversation(currentActiveId);
            conversations[currentActiveId] = applyConversationDetail(currentActiveId, detail);
          } catch {
            nextActiveId = null;
          }
        }

        set({ conversations, activeId: nextActiveId });
        persistActiveChatId(nextActiveId);

        if (nextActiveId && conversations[nextActiveId]?.activeRun?.can_resume) {
          void resumeConversationStream({
            applyConversationDetail,
            conversationId: nextActiveId,
            get,
            run: conversations[nextActiveId].activeRun,
            set,
          });
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
          void resumeConversationStream({
            applyConversationDetail,
            conversationId: id,
            get,
            run: mapped.activeRun,
            set,
          });
        }
      } catch (error) {
        console.error('Failed to load conversation detail:', error);
      }
    },

    deleteChat: async (id) => {
      try {
        await chatApi.deleteConversation(id);
        set((state) => {
          const nextConversations = { ...state.conversations };
          delete nextConversations[id];
          const nextActiveId = state.activeId === id ? null : state.activeId;
          persistActiveChatId(nextActiveId);

          return {
            conversations: nextConversations,
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

    updateConversationShare: (id, isShared, shareToken) => {
      set((state) => {
        const conversation = state.conversations[id];
        if (!conversation) return state;

        return {
          conversations: {
            ...state.conversations,
            [id]: {
              ...conversation,
              isShared,
              shareToken,
            },
          },
        };
      });
    },

    sendMessage: async (content) => {
      await streamConversationMessage({
        applyConversationDetail,
        content,
        get,
        set,
      });
    },

    clearMessages: () => {
      const { activeId } = get();
      if (!activeId) return;

      set((state) => ({
        conversations: {
          ...state.conversations,
          [activeId]: {
            ...state.conversations[activeId],
            messages: [],
            steps: [],
            activeRun: null,
          },
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
