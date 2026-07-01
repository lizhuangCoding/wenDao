import { useMemo } from 'react';
import { useChatStore } from '@/store';
import { buildChatQuestionNavItems } from '@/utils/chatQuestionNavigator';
import { selectFeaturedAgentStep } from '@/utils/agentMood';
import type { ChatMessage } from '@/types';

const EMPTY_CHAT_MESSAGES: ChatMessage[] = [];

export const useAIChatRuntime = () => {
  const {
    conversations,
    activeId,
    currentStage,
    isTyping,
    isStreaming,
    currentStageLabel,
    requiresUserInput,
    pendingQuestion,
    runStatus,
    selectedModel,
    loadConversations,
    sendMessage,
    createNewChat,
    setActiveChat,
    deleteChat,
    renameChat,
    updateConversationShare,
    setSelectedModel,
  } = useChatStore();

  const activeChat = activeId ? conversations[activeId] : null;
  const activeChatTitle = activeChat?.title ?? '';
  const activeChatMessages = activeChat?.messages;
  const messages = activeChatMessages ?? EMPTY_CHAT_MESSAGES;
  const questionNavItems = useMemo(() => buildChatQuestionNavItems(messages), [messages]);
  const questionAnchorByMessageId = useMemo(
    () => new Map(questionNavItems.map((item) => [item.messageId, item.anchorId])),
    [questionNavItems]
  );
  const latestProcessSteps = useMemo(() => {
    for (let index = messages.length - 1; index >= 0; index -= 1) {
      const steps = messages[index].processSteps || [];
      if (steps.length > 0) {
        return steps;
      }
    }

    return [];
  }, [messages]);

  return {
    conversations,
    activeId,
    activeChat,
    activeChatTitle,
    activeChatMessages,
    messages,
    currentStage,
    isTyping,
    isStreaming,
    currentStageLabel,
    requiresUserInput,
    pendingQuestion,
    runStatus,
    selectedModel,
    questionNavItems,
    questionAnchorByMessageId,
    featuredAgentStep: selectFeaturedAgentStep(latestProcessSteps),
    loadConversations,
    sendMessage,
    createNewChat,
    setActiveChat,
    deleteChat,
    renameChat,
    updateConversationShare,
    setSelectedModel,
  };
};
