import { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { Layout, ConfirmModal } from '@/components/common';
import { AIProcessingHalo } from '@/components/chat/AgentMoodIndicator';
import { AIChatHeader } from '@/components/chat/AIChatHeader';
import { ChatComposer } from '@/components/chat/ChatComposer';
import { ModelSelector } from '@/components/chat/ModelSelector';
import { ChatHistorySidebar } from '@/components/chat/ChatHistorySidebar';
import { ChatMessageList } from '@/components/chat/ChatMessageList';
import { ChatQuestionNavigator } from '@/components/chat/ChatQuestionNavigator';
import { ChatStageBanner } from '@/components/chat/ChatStageBanner';
import { StarterPrompts } from '@/components/chat/StarterPrompts';
import { useChatStore, useUIStore } from '@/store';
import { useAuth } from '@/hooks';
import { useProcessingTimer } from '@/hooks/useProcessingTimer';
import { useTranslation } from 'react-i18next';
import { motion, AnimatePresence } from 'framer-motion';
import { ArrowDown } from 'lucide-react';
import { buildChatQuestionNavItems } from '@/utils/chatQuestionNavigator';
import { selectFeaturedAgentStep } from '@/utils/agentMood';
import type { ChatMessage } from '@/types';
import { useAIChatPageState } from './useAIChatPageState';

const EMPTY_CHAT_MESSAGES: ChatMessage[] = [];

export const AIChat = () => {
  const { t } = useTranslation();
  const { user } = useAuth();
  const { showToast } = useUIStore();
  const [input, setInput] = useState('');
  const [expandedProcessIds, setExpandedProcessIds] = useState<Set<string>>(new Set());
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const composerRef = useRef<HTMLTextAreaElement>(null);
  const [isNearBottom, setIsNearBottom] = useState(true);
  const [activeQuestionId, setActiveQuestionId] = useState<string | null>(null);
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

  useEffect(() => {
    loadConversations();
  }, [loadConversations]);

  const activeChat = activeId ? conversations[activeId] : null;
  const activeChatTitle = activeChat?.title ?? '';
  const activeChatMessages = activeChat?.messages;
  const messages = activeChatMessages ?? EMPTY_CHAT_MESSAGES;
  const isAssistantProcessing = runStatus === 'running';
  const { elapsedLabel: processingDurationLabel } = useProcessingTimer(isAssistantProcessing);
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

  const featuredAgentStep = useMemo(() => {
    return selectFeaturedAgentStep(latestProcessSteps);
  }, [latestProcessSteps]);

  const {
    activeMenuId,
    deleteId,
    draftTitle,
    handleCopyShareLink,
    handleDeleteConfirm,
    handleExport,
    handleRenameCancel,
    handleRenameSave,
    handleStartRename,
    handleToggleShare,
    hasMessages,
    isDeletingChat,
    isExporting,
    isHistoryDrawerOpen,
    isImmersive,
    isRenaming,
    isShared,
    isSharing,
    isSidebarCollapsed,
    menuRef,
    setActiveMenuId,
    setDeleteId,
    setDraftTitle,
    setIsHistoryDrawerOpen,
    setIsSidebarCollapsed,
    shareCopied,
    toggleImmersive,
  } = useAIChatPageState({
    activeChat,
    deleteChat,
    messageCount: messages.length,
    renameChat,
    setActiveChat,
    showToast,
    t,
    updateConversationShare,
  });

  const updateActiveQuestionFromScroll = useCallback(() => {
    const container = scrollContainerRef.current;
    if (!container || questionNavItems.length === 0) {
      setActiveQuestionId(null);
      return;
    }

    const markerTop = container.scrollTop + 140;
    let nextActiveId = questionNavItems[0].anchorId;

    for (const item of questionNavItems) {
      const element = document.getElementById(item.anchorId);
      if (!element) continue;
      if (element.offsetTop <= markerTop) {
        nextActiveId = item.anchorId;
      } else {
        break;
      }
    }

    setActiveQuestionId((current) => (current === nextActiveId ? current : nextActiveId));
  }, [questionNavItems]);

  const scrollToBottom = useCallback((behavior: ScrollBehavior = 'smooth') => {
    const container = scrollContainerRef.current;
    if (!container) return;

    container.scrollTo({
      top: container.scrollHeight,
      behavior,
    });
    setIsNearBottom(true);
    window.requestAnimationFrame(updateActiveQuestionFromScroll);
  }, [updateActiveQuestionFromScroll]);

  useEffect(() => {
    if (messages.length === 0) return;
    const container = scrollContainerRef.current;
    if (!container || !isNearBottom) return;
    const frame = window.requestAnimationFrame(() => scrollToBottom('smooth'));
    return () => window.cancelAnimationFrame(frame);
  }, [activeChatMessages, isTyping, isStreaming, isNearBottom, messages.length, scrollToBottom]);

  useEffect(() => {
    const frame = window.requestAnimationFrame(updateActiveQuestionFromScroll);
    return () => window.cancelAnimationFrame(frame);
  }, [activeId, updateActiveQuestionFromScroll]);

  useEffect(() => {
    const textarea = composerRef.current;
    if (!textarea) return;
    textarea.style.height = 'auto';
    textarea.style.height = `${Math.min(textarea.scrollHeight, 160)}px`;
  }, [input]);

  const handleScroll = useCallback(() => {
    const container = scrollContainerRef.current;
    if (!container) return;
    const distanceFromBottom = container.scrollHeight - container.scrollTop - container.clientHeight;
    setIsNearBottom(distanceFromBottom <= 80);
    updateActiveQuestionFromScroll();
  }, [updateActiveQuestionFromScroll]);

  const submitMessage = useCallback(async (overrideMessage?: string) => {
    const message = (overrideMessage ?? input).trim();
    if (!message || isTyping) return;

    setInput('');
    await sendMessage(message);
  }, [input, isTyping, sendMessage]);

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    void submitMessage();
  };

  const handleComposerKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key !== 'Enter' || e.shiftKey || e.nativeEvent.isComposing) return;
    e.preventDefault();
    void submitMessage();
  };

  const scrollToQuestion = useCallback((anchorId: string) => {
    const container = scrollContainerRef.current;
    const element = document.getElementById(anchorId);
    if (!container || !element) return;

    container.scrollTo({
      top: Math.max(element.offsetTop - 24, 0),
      behavior: 'smooth',
    });
    setActiveQuestionId(anchorId);
  }, []);

  const handleStarterPromptSelect = useCallback((prompt: string) => {
    void submitMessage(prompt);
  }, [submitMessage]);

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
      .then(() => showToast(t('chat.copySuccess'), 'success'))
      .catch(() => showToast(t('chat.copyFailed'), 'error'));
  };

  const toggleProcessDetail = (id: string) => {
    setExpandedProcessIds((current) => {
      const next = new Set(current);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  return (
    <Layout hideHeader={isImmersive} hideFooter={isImmersive}>
      <div className={`${
        isImmersive
          ? 'w-full h-dvh px-0 py-0'
          : `${isSidebarCollapsed ? 'max-w-[1680px]' : 'max-w-display'} mx-auto px-3 py-3 sm:px-8 sm:py-6 lg:px-10 lg:py-10 h-[calc(100dvh-80px)]`
      } flex min-h-0 gap-3 lg:gap-6`}>
        <ChatHistorySidebar
          activeId={activeId}
          activeMenuId={activeMenuId}
          conversations={conversations}
          isDrawerOpen={isHistoryDrawerOpen}
          isImmersive={isImmersive}
          isSidebarCollapsed={isSidebarCollapsed}
          labels={{
            deleteLabel: t('admin.delete'),
            newSession: t('chat.newSession'),
            rename: t('chat.rename'),
          }}
          menuRef={menuRef}
          onCreateNewChat={() => void createNewChat()}
          onDrawerOpenChange={setIsHistoryDrawerOpen}
          onRequestDelete={setDeleteId}
          onSelectChat={(chatId) => void setActiveChat(chatId)}
          onSidebarCollapsedChange={setIsSidebarCollapsed}
          onStartRename={(chat) => {
            void handleStartRename(chat);
          }}
          setActiveMenuId={setActiveMenuId}
        />

        <main className={`min-w-0 flex-1 flex flex-col h-full bg-white dark:bg-neutral-800 overflow-hidden relative ${
          isImmersive
            ? 'rounded-none border-0 shadow-none'
            : 'rounded-2xl sm:rounded-[32px] border border-neutral-200 dark:border-neutral-700 shadow-soft'
        }`}>
          {/* AIChatHeader retains compact mobile controls: flex flex-col items-stretch gap-3 / overflow-x-auto pb-1 / <span className="hidden sm:inline"> */}
          <AIChatHeader
            canManageConversation={Boolean(activeChat)}
            draftTitle={draftTitle}
            hasMessages={hasMessages}
            isExporting={isExporting}
            isImmersive={isImmersive}
            isRenaming={isRenaming}
            isShared={isShared}
            isSharing={isSharing}
            shareCopied={shareCopied}
            title={activeChatTitle}
            onCopyShareLink={handleCopyShareLink}
            onDraftTitleChange={setDraftTitle}
            onExport={handleExport}
            onOpenHistory={() => setIsHistoryDrawerOpen(true)}
            onRenameCancel={handleRenameCancel}
            onRenameSave={handleRenameSave}
            onRenameStart={() => {
              void handleStartRename();
            }}
            onToggleImmersive={toggleImmersive}
            onToggleShare={handleToggleShare}
          />

          <ChatQuestionNavigator
            activeId={activeQuestionId}
            items={questionNavItems}
            onSelect={scrollToQuestion}
          />

          <div
            ref={scrollContainerRef}
            onScroll={handleScroll}
            className="flex-1 min-h-0 overflow-y-auto px-3 sm:px-8 lg:px-10 py-5 lg:py-10 space-y-6 sm:space-y-8 scrollbar-hide relative bg-neutral-50/30 dark:bg-neutral-800/50"
          >
            <ChatStageBanner
              currentStage={currentStage}
              featuredAgentStep={featuredAgentStep}
              isAssistantProcessing={isAssistantProcessing}
              label={currentStageLabel}
              pendingQuestion={pendingQuestion}
              processingDurationLabel={processingDurationLabel}
              requiresUserInput={requiresUserInput}
            />

            {messages.length === 0 && (
              <StarterPrompts
                disabled={isTyping}
                heading={t('chat.howCanIHelp')}
                subheading={t('chat.askAbout')}
                onSelect={handleStarterPromptSelect}
              />
            )}

            <ChatMessageList
              currentStage={currentStage}
              expandedProcessIds={expandedProcessIds}
              featuredAgentStep={featuredAgentStep}
              isAssistantProcessing={isAssistantProcessing}
              isTyping={isTyping}
              messages={messages}
              onCopy={copyToClipboard}
              onToggleProcessDetail={toggleProcessDetail}
              processingDurationLabel={processingDurationLabel}
              questionAnchorByMessageId={questionAnchorByMessageId}
              userAvatarUrl={user?.avatar_url}
              username={user?.username}
            />
          </div>

          <AnimatePresence>
            {!isNearBottom && (
              <motion.button
                type="button"
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: 8 }}
                onClick={() => scrollToBottom('smooth')}
                className="absolute bottom-28 left-1/2 z-20 hidden -translate-x-1/2 items-center gap-2 rounded-full border border-neutral-200 bg-white/95 px-4 py-2 text-xs font-bold text-neutral-600 shadow-soft backdrop-blur transition-colors hover:border-primary-200 hover:text-primary-600 sm:inline-flex dark:border-neutral-700 dark:bg-neutral-900/95 dark:text-neutral-300 dark:hover:border-primary-800 dark:hover:text-primary-300"
                aria-label={t('chat.backToBottom')}
              >
                <ArrowDown className="h-4 w-4" aria-hidden="true" />
                {t('chat.backToBottom')}
              </motion.button>
            )}
          </AnimatePresence>

          <div className="px-3 sm:px-8 lg:px-10 py-4 lg:py-8 bg-white dark:bg-neutral-800 border-t border-neutral-200 dark:border-neutral-700 rounded-b-2xl sm:rounded-b-[32px]">
            <AnimatePresence>
              {isAssistantProcessing && (
                <AIProcessingHalo
                  agentName={featuredAgentStep?.agent_name}
                  detail={featuredAgentStep?.detail}
                  elapsedLabel={t('chat.elapsed', { duration: processingDurationLabel })}
                  stage={currentStage}
                  stageLabel={currentStageLabel}
                  status={featuredAgentStep?.status || 'running'}
                  summary={featuredAgentStep?.summary}
                />
              )}
            </AnimatePresence>
            <div className="flex justify-end mb-2">
              <ModelSelector
                selectedModel={selectedModel}
                onSelect={setSelectedModel}
              />
            </div>
            <ChatComposer
              disabled={isTyping}
              input={input}
              onChange={setInput}
              onKeyDown={handleComposerKeyDown}
              onSubmit={handleSubmit}
              pendingQuestion={pendingQuestion}
              placeholder={t('chat.messagePlaceholder')}
              poweredBy={t('chat.poweredBy')}
              requiresUserInput={requiresUserInput}
              textareaRef={composerRef}
            />
          </div>
        </main>
      </div>

      <ConfirmModal
        isOpen={deleteId !== null}
        title={t('chat.deleteSession')}
        message={t('chat.deleteConfirm')}
        onConfirm={handleDeleteConfirm}
        onCancel={() => setDeleteId(null)}
        isConfirming={isDeletingChat}
        isDanger
      />
    </Layout>
  );
};
