import { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { Layout, ConfirmModal } from '@/components/common';
import { AIProcessingHalo } from '@/components/chat/AgentMoodIndicator';
import { ChatComposer } from '@/components/chat/ChatComposer';
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

const CHAT_SIDEBAR_STORAGE_KEY = 'wendao.aiChat.sidebar';
const CHAT_IMMERSIVE_STORAGE_KEY = 'wendao.aiChat.immersive';
const EMPTY_CHAT_MESSAGES: ChatMessage[] = [];

export const AIChat = () => {
  const { t } = useTranslation();
  const { user } = useAuth();
  const { showToast } = useUIStore();
  const [input, setInput] = useState('');
  const [isRenaming, setIsRenaming] = useState(false);
  const [draftTitle, setDraftTitle] = useState('');
  const [activeMenuId, setActiveMenuId] = useState<number | null>(null);
  const [deleteId, setDeleteId] = useState<number | null>(null);
  const [isDeletingChat, setIsDeletingChat] = useState(false);
  const [expandedProcessIds, setExpandedProcessIds] = useState<Set<string>>(new Set());
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(() => {
    if (typeof window === 'undefined') return false;
    return window.localStorage.getItem(CHAT_SIDEBAR_STORAGE_KEY) === 'collapsed';
  });
  const [isImmersive, setIsImmersive] = useState(() => {
    if (typeof window === 'undefined') return false;
    return window.localStorage.getItem(CHAT_IMMERSIVE_STORAGE_KEY) === 'immersive';
  });
  const [isHistoryDrawerOpen, setIsHistoryDrawerOpen] = useState(false);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const composerRef = useRef<HTMLTextAreaElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
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
    loadConversations,
    sendMessage,
    createNewChat,
    setActiveChat,
    deleteChat,
    renameChat,
  } = useChatStore();

  useEffect(() => {
    loadConversations();
  }, [loadConversations]);

  useEffect(() => {
    window.localStorage.setItem(CHAT_SIDEBAR_STORAGE_KEY, isSidebarCollapsed ? 'collapsed' : 'expanded');
  }, [isSidebarCollapsed]);

  useEffect(() => {
    window.localStorage.setItem(CHAT_IMMERSIVE_STORAGE_KEY, isImmersive ? 'immersive' : 'windowed');
  }, [isImmersive]);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setActiveMenuId(null);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

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
    if (activeChatTitle && !isRenaming) {
      setDraftTitle(activeChatTitle);
    }
  }, [activeChatTitle, isRenaming]);

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

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
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

  const handleRenameSave = async () => {
    if (!activeChat || !draftTitle.trim()) return;
    await renameChat(activeChat.id, draftTitle.trim());
    setIsRenaming(false);
  };

  const handleDeleteConfirm = async () => {
    if (deleteId && !isDeletingChat) {
      setIsDeletingChat(true);
      try {
        await deleteChat(deleteId);
        setDeleteId(null);
        showToast(t('chat.deleteSuccess'), 'success');
      } catch (err: any) {
        showToast(err.message || '删除会话失败，请重试', 'error');
      } finally {
        setIsDeletingChat(false);
      }
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    showToast('已复制到剪贴板', 'success');
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
            void setActiveChat(chat.id);
            setDraftTitle(chat.title);
            setIsRenaming(true);
          }}
          setActiveMenuId={setActiveMenuId}
        />

        <main className={`min-w-0 flex-1 flex flex-col h-full bg-white dark:bg-neutral-800 overflow-hidden relative ${
          isImmersive
            ? 'rounded-none border-0 shadow-none'
            : 'rounded-2xl sm:rounded-[32px] border border-neutral-100 dark:border-neutral-700 shadow-soft'
        }`}>
          <header className={`px-4 sm:px-8 lg:px-10 py-4 lg:py-6 border-b border-neutral-100 dark:border-neutral-700 flex items-center justify-between gap-3 bg-white dark:bg-neutral-800 z-10 ${
            isImmersive ? 'rounded-none' : 'rounded-t-2xl sm:rounded-[32px]'
          }`}>
            <div className="flex items-center gap-3 min-w-0">
              <button
                type="button"
                onClick={() => setIsHistoryDrawerOpen(true)}
                className="lg:hidden h-9 w-9 sm:w-10 sm:h-10 flex shrink-0 items-center justify-center rounded-xl border border-neutral-100 dark:border-neutral-700 text-neutral-500 dark:text-neutral-300 hover:text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-colors"
                aria-label="打开会话历史"
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h10M4 18h16" />
                </svg>
              </button>
              <div className="min-w-0">
                {isRenaming && activeChat ? (
                <div className="flex items-center gap-3">
                  <input
                    value={draftTitle}
                    onChange={(e) => setDraftTitle(e.target.value)}
                    placeholder={t('chat.renamePlaceholder')}
                    className="min-w-0 bg-transparent border border-neutral-200 dark:border-neutral-600 rounded-lg px-3 py-2 text-base sm:text-lg font-serif font-black text-neutral-900 dark:text-neutral-100"
                    onKeyDown={async (e) => {
                      if (e.key === 'Enter' && draftTitle.trim()) {
                        await handleRenameSave();
                      }
                      if (e.key === 'Escape' && activeChat) {
                        setDraftTitle(activeChat.title);
                        setIsRenaming(false);
                      }
                    }}
                    onBlur={() => {
                      if (activeChat && draftTitle !== activeChat.title && draftTitle.trim()) {
                        void handleRenameSave();
                      } else if (activeChat) {
                        setDraftTitle(activeChat.title);
                        setIsRenaming(false);
                      }
                    }}
                    autoFocus
                  />
                  <button type="button" onClick={() => void handleRenameSave()} className="text-xs text-primary-600 dark:text-primary-400">
                    {t('chat.saveName')}
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      if (activeChat) setDraftTitle(activeChat.title);
                      setIsRenaming(false);
                    }}
                    className="text-xs text-neutral-500 dark:text-neutral-400"
                  >
                    {t('chat.cancelRename')}
                  </button>
                </div>
              ) : (
                <div className="flex items-center gap-3">
                  <h2 className="text-lg font-serif font-black text-neutral-900 dark:text-neutral-100 truncate">
                    {activeChat?.title || t('chat.title')}
                  </h2>
                  {activeChat && (
                    <button
                      type="button"
                      onClick={() => setIsRenaming(true)}
                      className="text-xs text-neutral-400 hover:text-primary-600 dark:hover:text-primary-400"
                    >
                      {t('chat.rename')}
                    </button>
                  )}
                </div>
                )}
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setIsImmersive((value) => !value)}
                className="inline-flex shrink-0 items-center gap-2 rounded-xl border border-neutral-100 dark:border-neutral-700 px-2.5 py-2 sm:px-3 text-xs font-bold text-neutral-500 dark:text-neutral-300 hover:text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-colors"
                title={isImmersive ? '退出沉浸模式' : '开启沉浸模式'}
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  {isImmersive ? (
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 3H5a2 2 0 00-2 2v3m16 0V5a2 2 0 00-2-2h-3m0 18h3a2 2 0 002-2v-3M5 16v3a2 2 0 002 2h3" />
                  ) : (
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V5a1 1 0 011-1h3m8 0h3a1 1 0 011 1v3m0 8v3a1 1 0 01-1 1h-3M8 20H5a1 1 0 01-1-1v-3" />
                  )}
                </svg>
                <span className="hidden sm:inline">{isImmersive ? '退出全屏' : '沉浸模式'}</span>
              </button>
            </div>
          </header>

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
                aria-label="回到底部"
              >
                <ArrowDown className="h-4 w-4" aria-hidden="true" />
                回到底部
              </motion.button>
            )}
          </AnimatePresence>

          <div className="px-3 sm:px-8 lg:px-10 py-4 lg:py-8 bg-white dark:bg-neutral-800 border-t border-neutral-100 dark:border-neutral-700 rounded-b-2xl sm:rounded-b-[32px]">
            <AnimatePresence>
              {isAssistantProcessing && (
                <AIProcessingHalo
                  agentName={featuredAgentStep?.agent_name}
                  detail={featuredAgentStep?.detail}
                  elapsedLabel={`已耗时 ${processingDurationLabel}`}
                  stage={currentStage}
                  stageLabel={currentStageLabel}
                  status={featuredAgentStep?.status || 'running'}
                  summary={featuredAgentStep?.summary}
                />
              )}
            </AnimatePresence>
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
