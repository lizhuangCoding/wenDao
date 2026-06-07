import type { RefObject } from 'react';
import { useTranslation } from 'react-i18next';
import { AnimatePresence, motion } from 'framer-motion';

export interface ChatHistoryConversation {
  id: number;
  title: string;
  messages: unknown[];
  createdAt: number;
  updatedAt: number;
  isLoaded: boolean;
}

interface ChatHistorySidebarLabels {
  deleteLabel: string;
  newSession: string;
  rename: string;
}

interface ChatHistorySidebarProps {
  activeId: number | null;
  activeMenuId: number | null;
  conversations: Record<number, ChatHistoryConversation>;
  isDrawerOpen: boolean;
  isImmersive: boolean;
  isSidebarCollapsed: boolean;
  labels: ChatHistorySidebarLabels;
  menuRef: RefObject<HTMLDivElement>;
  onCreateNewChat: () => void;
  onDrawerOpenChange: (isOpen: boolean) => void;
  onRequestDelete: (chatId: number) => void;
  onSelectChat: (chatId: number) => void;
  onSidebarCollapsedChange: (isCollapsed: boolean) => void;
  onStartRename: (chat: ChatHistoryConversation) => void;
  setActiveMenuId: (chatId: number | null) => void;
}

const isEmptyChat = (chat: ChatHistoryConversation) => {
  if (chat.isLoaded) return chat.messages.length === 0;
  return chat.updatedAt === chat.createdAt;
};

const sortConversations = (conversations: Record<number, ChatHistoryConversation>) => {
  return Object.values(conversations).sort((a, b) => {
    const aIsEmpty = isEmptyChat(a);
    const bIsEmpty = isEmptyChat(b);
    if (aIsEmpty && !bIsEmpty) return -1;
    if (!aIsEmpty && bIsEmpty) return 1;
    if (b.updatedAt !== a.updatedAt) {
      return b.updatedAt - a.updatedAt;
    }
    return b.id - a.id;
  });
};

export const ChatHistorySidebar = ({
  activeId,
  activeMenuId,
  conversations,
  isDrawerOpen,
  isImmersive,
  isSidebarCollapsed,
  labels,
  menuRef,
  onCreateNewChat,
  onDrawerOpenChange,
  onRequestDelete,
  onSelectChat,
  onSidebarCollapsedChange,
  onStartRename,
  setActiveMenuId,
}: ChatHistorySidebarProps) => {
  const { t, i18n } = useTranslation();
  const sortedConversations = sortConversations(conversations);
  const hasEmptyChat = sortedConversations.some(isEmptyChat);

  const handleCreateNewChat = () => {
    onCreateNewChat();
    onDrawerOpenChange(false);
  };

  const renderNewChatButton = (compact = false) => (
    <button
      type="button"
      onClick={handleCreateNewChat}
      disabled={hasEmptyChat}
      className={`flex items-center justify-center gap-2 text-xs font-black tracking-widest rounded-2xl transition-all shadow-soft active:scale-95 ${
        compact ? 'w-12 h-12' : 'w-full py-4'
      } ${
        hasEmptyChat
          ? 'bg-neutral-100 dark:bg-neutral-800 text-neutral-400 cursor-not-allowed'
          : 'bg-neutral-900 dark:bg-neutral-800 text-white dark:hover:bg-neutral-700 hover:bg-primary-600'
      }`}
      title={labels.newSession}
    >
      <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M12 4v16m8-8H4" />
      </svg>
      {!compact && labels.newSession}
    </button>
  );

  const renderConversationList = (compact = false, onSelect?: () => void) => (
    <AnimatePresence mode="popLayout">
      {sortedConversations.map((chat) => (
        <motion.div
          key={chat.id}
          layout
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          exit={{ opacity: 0, scale: 0.95 }}
          className={`group relative flex items-center gap-3 rounded-2xl cursor-pointer transition-all ${
            compact ? 'justify-center p-3' : 'p-4'
          } ${
            activeId === chat.id
              ? 'bg-primary-50 dark:bg-primary-900/30 ring-1 ring-primary-100 dark:ring-primary-800'
              : 'hover:bg-neutral-50 dark:hover:bg-neutral-800'
          }`}
          onClick={() => {
            onSelectChat(chat.id);
            onSelect?.();
          }}
          title={chat.title}
        >
          <div className={`rounded-full ${compact ? 'w-2.5 h-2.5' : 'w-2 h-2'} ${activeId === chat.id ? 'bg-primary-500' : 'bg-neutral-200 dark:bg-neutral-600'}`} />
          {!compact && (
            <>
              <div className="flex-1 min-w-0">
                <p className={`text-sm font-bold truncate ${activeId === chat.id ? 'text-primary-900 dark:text-primary-400' : 'text-neutral-600 dark:text-neutral-300'}`}>
                  {chat.title}
                </p>
                <p className="text-[10px] text-neutral-400 dark:text-neutral-500 font-medium uppercase mt-0.5">
                  {new Date(chat.updatedAt).toLocaleDateString(i18n.resolvedLanguage?.startsWith('en') ? 'en-US' : 'zh-CN')}
                </p>
              </div>
              <div className="relative">
                <button
                  type="button"
                  onClick={(event) => {
                    event.stopPropagation();
                    setActiveMenuId(activeMenuId === chat.id ? null : chat.id);
                  }}
                  className={`p-1 rounded-lg transition-all ${
                    activeMenuId === chat.id
                      ? 'bg-white dark:bg-neutral-700 shadow-sm text-primary-500'
                      : 'opacity-0 group-hover:opacity-100 text-neutral-400 hover:text-neutral-600 dark:hover:text-neutral-200'
                  }`}
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h.01M12 12h.01M19 12h.01M6 12a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0z" />
                  </svg>
                </button>

                <AnimatePresence>
                  {activeMenuId === chat.id && (
                    <motion.div
                      ref={menuRef}
                      initial={{ opacity: 0, scale: 0.95, y: -10 }}
                      animate={{ opacity: 1, scale: 1, y: 0 }}
                      exit={{ opacity: 0, scale: 0.95, y: -10 }}
                      className="absolute right-0 top-full mt-2 w-36 bg-white dark:bg-neutral-800 rounded-xl shadow-elevated border border-neutral-100 dark:border-neutral-700 py-1.5 z-[100] backdrop-blur-sm"
                    >
                      <button
                        type="button"
                        onClick={(event) => {
                          event.stopPropagation();
                          onStartRename(chat);
                          setActiveMenuId(null);
                          onSelect?.();
                        }}
                        className="w-full flex items-center gap-2.5 px-3 py-2 text-xs font-bold text-neutral-600 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-700/50 transition-colors"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                        </svg>
                        {labels.rename}
                      </button>
                      <div className="h-px bg-neutral-100 dark:bg-neutral-700 mx-1.5 my-1" />
                      <button
                        type="button"
                        onClick={(event) => {
                          event.stopPropagation();
                          onRequestDelete(chat.id);
                          setActiveMenuId(null);
                          onSelect?.();
                        }}
                        className="w-full flex items-center gap-2.5 px-3 py-2 text-xs font-bold text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                        {labels.deleteLabel}
                      </button>
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>
            </>
          )}
        </motion.div>
      ))}
    </AnimatePresence>
  );

  return (
    <>
      <AnimatePresence>
        {isDrawerOpen && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 bg-neutral-950/40 backdrop-blur-sm lg:hidden"
            onClick={() => onDrawerOpenChange(false)}
          >
            <motion.aside
              initial={{ x: -320 }}
              animate={{ x: 0 }}
              exit={{ x: -320 }}
              transition={{ type: 'spring', stiffness: 260, damping: 28 }}
              className="h-full w-[min(88vw,320px)] bg-white dark:bg-neutral-900 border-r border-neutral-100 dark:border-neutral-800 p-5 flex flex-col gap-4 shadow-elevated"
              onClick={(event) => event.stopPropagation()}
            >
              <div className="flex items-center justify-between">
                <p className="text-sm font-black text-neutral-900 dark:text-neutral-100">{t('chat.conversationHistory')}</p>
                <button
                  type="button"
                  onClick={() => onDrawerOpenChange(false)}
                  className="w-10 h-10 flex items-center justify-center rounded-xl text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors"
                  aria-label={t('chat.closeHistory')}
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
              {renderNewChatButton(false)}
              <div className="flex-1 overflow-y-auto space-y-3 scrollbar-hide">
                {renderConversationList(false, () => onDrawerOpenChange(false))}
              </div>
            </motion.aside>
          </motion.div>
        )}
      </AnimatePresence>

      <aside className={`${isSidebarCollapsed ? 'w-16 pr-3' : 'w-80 pr-6'} hidden lg:flex flex-col gap-4 h-full border-r border-neutral-100 dark:border-neutral-800 transition-all duration-200 ${isImmersive ? 'pl-4 py-4 bg-white dark:bg-neutral-900' : ''}`}>
        <button
          type="button"
          onClick={() => onSidebarCollapsedChange(!isSidebarCollapsed)}
          data-chat-history-toggle="sidebar"
          className="w-12 h-12 flex items-center justify-center rounded-2xl border border-neutral-100 dark:border-neutral-700 text-neutral-500 dark:text-neutral-300 hover:text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-colors"
          aria-label={isSidebarCollapsed ? t('chat.expandHistory') : t('chat.collapseHistory')}
          title={isSidebarCollapsed ? t('chat.expandHistory') : t('chat.collapseHistory')}
        >
          <svg xmlns="http://www.w3.org/2000/svg" className={`h-5 w-5 transition-transform ${isSidebarCollapsed ? 'rotate-180' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </button>

        {renderNewChatButton(isSidebarCollapsed)}

        <div className="flex-1 overflow-y-auto space-y-3 scrollbar-hide">
          {renderConversationList(isSidebarCollapsed)}
        </div>
      </aside>
    </>
  );
};
