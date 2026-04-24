import { useState, useRef, useEffect } from 'react';
import { Layout, ConfirmModal } from '@/components/common';
import { ArticleContent } from '@/components/article';
import { useChatStore, useUIStore } from '@/store';
import { useAuth } from '@/hooks';
import { useTranslation } from 'react-i18next';
import { motion, AnimatePresence } from 'framer-motion';
import type { ChatArticleReference, ChatReferenceGroups, ChatStep } from '@/types';

const CHAT_SIDEBAR_STORAGE_KEY = 'wendao.aiChat.sidebar';
const CHAT_IMMERSIVE_STORAGE_KEY = 'wendao.aiChat.immersive';

type AgentProcessPanelProps = {
  messageId: string;
  steps: ChatStep[];
  expandedIds: Set<string>;
  onToggle: (id: string) => void;
};

const statusText: Record<ChatStep['status'], string> = {
  running: '进行中',
  completed: '已完成',
  failed: '失败',
};

const AgentProcessPanel = ({ messageId, steps, expandedIds, onToggle }: AgentProcessPanelProps) => {
  if (!steps.length) return null;

  return (
    <div className="mb-4 rounded-lg border border-neutral-200 dark:border-neutral-600 bg-neutral-50 dark:bg-neutral-800/70 overflow-hidden">
      <div className="px-4 py-3 border-b border-neutral-200 dark:border-neutral-600">
        <p className="text-xs font-bold text-neutral-700 dark:text-neutral-200">多 Agent 协作过程</p>
        <p className="text-[11px] text-neutral-500 dark:text-neutral-400 mt-1">默认展示摘要，展开可查看工具调用、返回结果和原始日志。</p>
      </div>

      <div className="divide-y divide-neutral-200 dark:divide-neutral-700">
        {steps.map((step, index) => {
          const key = step.id > 0 ? `${messageId}-${step.id}` : `${messageId}-${step.agent_name}-${index}`;
          const isExpanded = expandedIds.has(key);
          const isRunning = step.status === 'running';
          const isFailed = step.status === 'failed';

          return (
            <div key={key} className="bg-white/70 dark:bg-neutral-800">
              <button
                type="button"
                onClick={() => onToggle(key)}
                className="w-full flex items-start gap-3 px-4 py-3 text-left hover:bg-neutral-50 dark:hover:bg-neutral-700/60 transition-colors"
                aria-expanded={isExpanded}
              >
                <span className={`mt-1 h-2 w-2 flex-shrink-0 rounded-full ${
                  isFailed ? 'bg-red-500' : isRunning ? 'bg-primary-500 animate-pulse' : 'bg-neutral-400'
                }`} />
                <span className="min-w-0 flex-1">
                  <span className="block text-xs font-bold text-neutral-800 dark:text-neutral-100">
                    {step.summary || `${step.agent_name} 正在协作`}
                  </span>
                  <span className="mt-1 block text-[11px] text-neutral-500 dark:text-neutral-400">
                    {step.agent_name} · {statusText[step.status] || step.status}
                  </span>
                </span>
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  className={`h-4 w-4 flex-shrink-0 text-neutral-400 transition-transform ${isExpanded ? 'rotate-90' : ''}`}
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </button>

              {isExpanded && (
                <div className="px-4 pb-4 pl-9">
                  <pre className="max-h-72 overflow-auto rounded-lg bg-neutral-950 text-neutral-100 p-3 text-[11px] leading-relaxed whitespace-pre-wrap">
                    {step.detail || '暂无详细过程日志。'}
                  </pre>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
};

const emptyReferenceGroups = (): ChatReferenceGroups => ({ blog: [], external: [] });

const parseReferenceLinks = (block: string): ChatArticleReference[] => {
  const references: ChatArticleReference[] = [];
  const seen = new Set<string>();
  const linkPattern = /-\s*\[([^\]]+)\]\(([^)]+)\)/g;
  let match: RegExpExecArray | null;

  while ((match = linkPattern.exec(block)) !== null) {
    const title = match[1].trim();
    const url = match[2].trim();
    const key = `${title}|${url}`;
    if (!title || !url || seen.has(key)) continue;
    seen.add(key);
    references.push({ title, url });
  }

  return references;
};

const splitArticleReferences = (content: string): { body: string; references: ChatReferenceGroups } => {
  const markerPattern = /\n{0,2}(参考博主文章|参考外部文章|参考文章)\s*\n/g;
  const markerMatch = markerPattern.exec(content);
  if (!markerMatch || markerMatch.index === undefined) {
    return { body: content, references: emptyReferenceGroups() };
  }

  const body = content.slice(0, markerMatch.index).trimEnd();
  const references = emptyReferenceGroups();
  const markers: Array<{ title: string; start: number; contentStart: number }> = [];

  markers.push({ title: markerMatch[1], start: markerMatch.index, contentStart: markerMatch.index + markerMatch[0].length });
  let match: RegExpExecArray | null;
  while ((match = markerPattern.exec(content)) !== null) {
    markers.push({ title: match[1], start: match.index, contentStart: match.index + match[0].length });
  }

  markers.forEach((marker, index) => {
    const nextStart = markers[index + 1]?.start ?? content.length;
    const links = parseReferenceLinks(content.slice(marker.contentStart, nextStart));
    if (marker.title === '参考外部文章') {
      references.external.push(...links);
    } else {
      references.blog.push(...links);
    }
  });

  if (references.blog.length === 0 && references.external.length === 0) {
    return { body: content, references: emptyReferenceGroups() };
  }
  return { body, references };
};

const ArticleReferencesPanel = ({ references }: { references: ChatReferenceGroups }) => {
  if (!references.blog.length && !references.external.length) return null;

  const renderGroup = (title: string, items: ChatArticleReference[], external = false) => {
    if (!items.length) return null;
    return (
      <div className="space-y-2">
        <p className="text-xs font-bold text-neutral-700 dark:text-neutral-200">{title}</p>
        {items.map((reference) => (
          <a
            key={`${reference.title}-${reference.url}`}
            href={reference.url}
            target={external ? '_blank' : undefined}
            rel={external ? 'noreferrer' : undefined}
            className="block rounded-lg border border-neutral-200 dark:border-neutral-600 px-3 py-2 text-sm font-medium text-primary-700 dark:text-primary-300 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-colors no-underline"
          >
            {reference.title}
          </a>
        ))}
      </div>
    );
  };

  return (
    <div className="mt-4 border-t border-neutral-200 dark:border-neutral-600 pt-4">
      <div className="space-y-4">
        {renderGroup('参考博主文章', references.blog)}
        {renderGroup('参考外部文章', references.external, true)}
      </div>
    </div>
  );
};

const formatProcessingDuration = (elapsedMs: number) => {
  const totalSeconds = Math.max(0, Math.floor(elapsedMs / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes > 0) {
    return `${minutes}分${seconds.toString().padStart(2, '0')}秒`;
  }
  return `${seconds}秒`;
};

export const AIChat = () => {
  const { t } = useTranslation();
  const { user } = useAuth();
  const { showToast } = useUIStore();
  const [input, setInput] = useState('');
  const [isRenaming, setIsRenaming] = useState(false);
  const [draftTitle, setDraftTitle] = useState('');
  const [activeMenuId, setActiveMenuId] = useState<number | null>(null);
  const [deleteId, setDeleteId] = useState<number | null>(null);
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
  const [processingStartedAt, setProcessingStartedAt] = useState<number | null>(null);
  const [processingElapsedMs, setProcessingElapsedMs] = useState(0);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const [isNearBottom, setIsNearBottom] = useState(true);
  const {
    conversations,
    activeId,
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
  const messages = activeChat?.messages || [];
  const isAssistantProcessing = runStatus === 'running';
  const processingDurationLabel = formatProcessingDuration(processingElapsedMs);

  useEffect(() => {
    if (!isAssistantProcessing) {
      setProcessingStartedAt(null);
      setProcessingElapsedMs(0);
      return;
    }

    const startedAt = processingStartedAt ?? Date.now();
    if (processingStartedAt === null) {
      setProcessingStartedAt(startedAt);
      setProcessingElapsedMs(0);
    }

    const updateElapsed = () => setProcessingElapsedMs(Date.now() - startedAt);
    updateElapsed();
    const timer = window.setInterval(updateElapsed, 1000);
    return () => window.clearInterval(timer);
  }, [isAssistantProcessing, processingStartedAt]);

  useEffect(() => {
    if (activeChat && !isRenaming) {
      setDraftTitle(activeChat.title);
    }
  }, [activeChat?.id, activeChat?.title, isRenaming]);

  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container || !isNearBottom) return;
    container.scrollTop = container.scrollHeight;
  }, [messages, isTyping, isStreaming, isNearBottom]);

  const handleScroll = () => {
    const container = scrollContainerRef.current;
    if (!container) return;
    const distanceFromBottom = container.scrollHeight - container.scrollTop - container.clientHeight;
    setIsNearBottom(distanceFromBottom <= 80);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isTyping) return;

    const message = input.trim();
    setInput('');
    await sendMessage(message);
  };

  const handleRenameSave = async () => {
    if (!activeChat || !draftTitle.trim()) return;
    await renameChat(activeChat.id, draftTitle.trim());
    setIsRenaming(false);
  };

  const handleDeleteConfirm = async () => {
    if (deleteId) {
      await deleteChat(deleteId);
      setDeleteId(null);
      showToast(t('chat.deleteSuccess'), 'success');
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

  const isEmptyChat = (chat: (typeof conversations)[number]) => {
    if (chat.isLoaded) return chat.messages.length === 0;
    return chat.updatedAt === chat.createdAt;
  };

  const hasEmptyChat = Object.values(conversations).some(isEmptyChat);
  const sortedConversations = Object.values(conversations).sort((a, b) => {
    const aIsEmpty = isEmptyChat(a);
    const bIsEmpty = isEmptyChat(b);
    if (aIsEmpty && !bIsEmpty) return -1;
    if (!aIsEmpty && bIsEmpty) return 1;
    if (b.updatedAt !== a.updatedAt) {
      return b.updatedAt - a.updatedAt;
    }
    return b.id - a.id;
  });

  const renderNewChatButton = (compact = false) => (
    <button
      onClick={() => {
        void createNewChat();
        setIsHistoryDrawerOpen(false);
      }}
      disabled={hasEmptyChat}
      className={`flex items-center justify-center gap-2 text-xs font-black tracking-widest rounded-2xl transition-all shadow-soft active:scale-95 ${
        compact ? 'w-12 h-12' : 'w-full py-4'
      } ${
        hasEmptyChat
          ? 'bg-neutral-100 dark:bg-neutral-800 text-neutral-400 cursor-not-allowed'
          : 'bg-neutral-900 dark:bg-neutral-800 text-white dark:hover:bg-neutral-700 hover:bg-primary-600'
      }`}
      title={t('chat.newSession')}
    >
      <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M12 4v16m8-8H4" />
      </svg>
      {!compact && t('chat.newSession')}
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
            void setActiveChat(chat.id);
            onSelect?.();
          }}
          title={chat.title}
        >
          <div className={`rounded-full ${compact ? 'w-2.5 h-2.5' : 'w-2 h-2'} ${activeId === chat.id ? 'bg-primary-500' : 'bg-neutral-200 dark:bg-neutral-600'}`}></div>
          {!compact && (
            <>
              <div className="flex-1 min-w-0">
                <p className={`text-sm font-bold truncate ${activeId === chat.id ? 'text-primary-900 dark:text-primary-400' : 'text-neutral-600 dark:text-neutral-300'}`}>
                  {chat.title}
                </p>
                <p className="text-[10px] text-neutral-400 dark:text-neutral-500 font-medium uppercase mt-0.5">
                  {new Date(chat.updatedAt).toLocaleDateString()}
                </p>
              </div>
              <div className="relative">
                <button
                  onClick={(e) => {
                    e.stopPropagation();
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
                        onClick={(e) => {
                          e.stopPropagation();
                          void setActiveChat(chat.id);
                          setDraftTitle(chat.title);
                          setIsRenaming(true);
                          setActiveMenuId(null);
                          onSelect?.();
                        }}
                        className="w-full flex items-center gap-2.5 px-3 py-2 text-xs font-bold text-neutral-600 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-700/50 transition-colors"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                        </svg>
                        {t('chat.rename')}
                      </button>
                      <div className="h-px bg-neutral-100 dark:bg-neutral-700 mx-1.5 my-1"></div>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteId(chat.id);
                          setActiveMenuId(null);
                          onSelect?.();
                        }}
                        className="w-full flex items-center gap-2.5 px-3 py-2 text-xs font-bold text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                        {t('admin.delete')}
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
    <Layout hideHeader={isImmersive} hideFooter={isImmersive}>
      <div className={`${
        isImmersive
          ? 'w-full h-screen px-0 py-0'
          : `${isSidebarCollapsed ? 'max-w-[1680px]' : 'max-w-display'} mx-auto px-4 sm:px-8 lg:px-10 py-6 lg:py-10 h-[calc(100vh-80px)]`
      } flex gap-4 lg:gap-6`}>
        <AnimatePresence>
          {isHistoryDrawerOpen && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="fixed inset-0 z-50 bg-neutral-950/40 backdrop-blur-sm lg:hidden"
              onClick={() => setIsHistoryDrawerOpen(false)}
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
                  <p className="text-sm font-black text-neutral-900 dark:text-neutral-100">会话历史</p>
                  <button
                    onClick={() => setIsHistoryDrawerOpen(false)}
                    className="w-10 h-10 flex items-center justify-center rounded-xl text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-colors"
                    aria-label="关闭会话历史"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
                {renderNewChatButton(false)}
                <div className="flex-1 overflow-y-auto space-y-3 scrollbar-hide">
                  {renderConversationList(false, () => setIsHistoryDrawerOpen(false))}
                </div>
              </motion.aside>
            </motion.div>
          )}
        </AnimatePresence>

        <aside className={`${isSidebarCollapsed ? 'w-16 pr-3' : 'w-80 pr-6'} hidden lg:flex flex-col gap-4 h-full border-r border-neutral-100 dark:border-neutral-800 transition-all duration-200 ${isImmersive ? 'pl-4 py-4 bg-white dark:bg-neutral-900' : ''}`}>
          <button
            onClick={() => setIsSidebarCollapsed((value) => !value)}
            className="w-12 h-12 flex items-center justify-center rounded-2xl border border-neutral-100 dark:border-neutral-700 text-neutral-500 dark:text-neutral-300 hover:text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-colors"
            title={isSidebarCollapsed ? '展开会话历史' : '收起会话历史'}
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

        <main className={`min-w-0 flex-1 flex flex-col h-full bg-white dark:bg-neutral-800 overflow-hidden relative ${
          isImmersive
            ? 'rounded-none border-0 shadow-none'
            : 'rounded-[32px] border border-neutral-100 dark:border-neutral-700 shadow-soft'
        }`}>
          <header className={`px-5 sm:px-8 lg:px-10 py-5 lg:py-6 border-b border-neutral-100 dark:border-neutral-700 flex items-center justify-between bg-white dark:bg-neutral-800 z-10 ${
            isImmersive ? 'rounded-none' : 'rounded-[32px]'
          }`}>
            <div className="flex items-center gap-3 min-w-0">
              <button
                onClick={() => setIsHistoryDrawerOpen(true)}
                className="lg:hidden w-10 h-10 flex items-center justify-center rounded-xl border border-neutral-100 dark:border-neutral-700 text-neutral-500 dark:text-neutral-300 hover:text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-colors"
                aria-label="打开会话历史"
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h10M4 18h16" />
                </svg>
              </button>
              <button
                onClick={() => setIsSidebarCollapsed((value) => !value)}
                className="hidden lg:flex w-10 h-10 items-center justify-center rounded-xl border border-neutral-100 dark:border-neutral-700 text-neutral-500 dark:text-neutral-300 hover:text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-colors"
                aria-label={isSidebarCollapsed ? '展开会话历史' : '收起会话历史'}
              >
                <svg xmlns="http://www.w3.org/2000/svg" className={`h-5 w-5 transition-transform ${isSidebarCollapsed ? 'rotate-180' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                </svg>
              </button>
              <div className="min-w-0">
                {isRenaming && activeChat ? (
                <div className="flex items-center gap-3">
                  <input
                    value={draftTitle}
                    onChange={(e) => setDraftTitle(e.target.value)}
                    placeholder={t('chat.renamePlaceholder')}
                    className="bg-transparent border border-neutral-200 dark:border-neutral-600 rounded-lg px-3 py-2 text-lg font-serif font-black text-neutral-900 dark:text-neutral-100"
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
                  <button onClick={() => void handleRenameSave()} className="text-xs text-primary-600 dark:text-primary-400">
                    {t('chat.saveName')}
                  </button>
                  <button
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
                onClick={() => setIsImmersive((value) => !value)}
                className="inline-flex items-center gap-2 rounded-xl border border-neutral-100 dark:border-neutral-700 px-3 py-2 text-xs font-bold text-neutral-500 dark:text-neutral-300 hover:text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-colors"
                title={isImmersive ? '退出沉浸模式' : '开启沉浸模式'}
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  {isImmersive ? (
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 3H5a2 2 0 00-2 2v3m16 0V5a2 2 0 00-2-2h-3m0 18h3a2 2 0 002-2v-3M5 16v3a2 2 0 002 2h3" />
                  ) : (
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V5a1 1 0 011-1h3m8 0h3a1 1 0 011 1v3m0 8v3a1 1 0 01-1 1h-3M8 20H5a1 1 0 01-1-1v-3" />
                  )}
                </svg>
                <span>{isImmersive ? '退出全屏' : '沉浸模式'}</span>
              </button>
            </div>
          </header>

          <div
            ref={scrollContainerRef}
            onScroll={handleScroll}
            className="flex-1 overflow-y-auto px-4 sm:px-8 lg:px-10 py-6 lg:py-10 space-y-8 scrollbar-hide relative bg-neutral-50/30 dark:bg-neutral-800/50"
          >
            {currentStageLabel && (
              <div className="mb-4 rounded-xl border border-primary-200 bg-primary-50 px-4 py-3 text-sm text-primary-700 dark:border-primary-800 dark:bg-primary-900/20 dark:text-primary-300">
                <div className="flex flex-wrap items-center gap-2 justify-between">
                  <span>
                    {currentStageLabel}
                    {requiresUserInput && pendingQuestion ? `：${pendingQuestion}` : ''}
                  </span>
                  {isAssistantProcessing && (
                    <span className="inline-flex items-center rounded-full bg-white/80 dark:bg-primary-950/40 px-2.5 py-1 text-[11px] font-bold text-primary-700 dark:text-primary-200">
                      已耗时 {processingDurationLabel}
                    </span>
                  )}
                </div>
              </div>
            )}

            {messages.length === 0 && (
              <div className="h-full flex flex-col items-center justify-center text-center max-w-sm mx-auto">
                <div className="w-20 h-20 bg-primary-50 dark:bg-primary-900/30 rounded-full flex items-center justify-center mb-6">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-10 w-10 text-primary-500 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
                  </svg>
                </div>
                <h3 className="text-2xl font-serif font-black text-neutral-900 dark:text-neutral-100 mb-2">{t('chat.howCanIHelp')}</h3>
                <p className="text-sm text-neutral-400 dark:text-neutral-500 font-medium">{t('chat.askAbout')}</p>
              </div>
            )}

            {messages.map((message) => {
              const articleRefs = message.role === 'assistant'
                ? splitArticleReferences(message.content)
                : { body: message.content, references: emptyReferenceGroups() };

              return (
                <motion.div
                  key={message.id}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
                >
                <div className={`flex gap-4 ${message.role === 'user' ? 'max-w-[85%] flex-row-reverse' : 'w-full max-w-5xl flex-row'}`}>
                  {message.role === 'user' ? (
                    <div className="w-8 h-8 rounded-full overflow-hidden flex-shrink-0 border border-neutral-200 dark:border-neutral-600">
                      <img
                        src={user?.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${user?.username}`}
                        alt={user?.username}
                        className="w-full h-full object-cover"
                      />
                    </div>
                  ) : (
                    <div className="w-8 h-8 rounded-full bg-primary-500 flex-shrink-0 flex items-center justify-center">
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                      </svg>
                    </div>
                  )}
                  <div className={`relative group/msg px-6 py-4 rounded-[24px] ${
                    message.role === 'user'
                      ? 'bg-neutral-900 dark:bg-neutral-700 text-white dark:text-neutral-100 rounded-tr-none shadow-elevated'
                      : 'bg-white dark:bg-neutral-700 text-neutral-800 dark:text-neutral-100 border border-neutral-100 dark:border-neutral-600 rounded-tl-none shadow-sm'
                  }`}>
                    {message.role === 'assistant' ? (
                      <div className="prose prose-sm max-w-none dark:prose-invert">
                        <AgentProcessPanel
                          messageId={message.id}
                          steps={message.processSteps || []}
                          expandedIds={expandedProcessIds}
                          onToggle={toggleProcessDetail}
                        />
                        {message.content ? (
                          <>
                            {articleRefs.body && <ArticleContent content={articleRefs.body} />}
                            <ArticleReferencesPanel references={articleRefs.references} />
                          </>
                        ) : (
                          <p className="text-sm text-neutral-500 dark:text-neutral-400">正在生成最终回答...</p>
                        )}
                      </div>
                    ) : (
                      <p className="text-sm leading-relaxed whitespace-pre-wrap font-medium">{message.content}</p>
                    )}
                    <div className="flex items-center justify-between mt-2">
                      <span className={`text-[9px] font-bold uppercase tracking-tighter ${
                        message.role === 'user' ? 'text-neutral-400' : 'text-neutral-400 dark:text-neutral-500'
                      }`}>
                        {new Date(message.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                      </span>
                      <button
                        onClick={() => copyToClipboard(message.content)}
                        className="opacity-0 group-hover/msg:opacity-100 p-1 text-neutral-400 hover:text-primary-500 transition-all ml-4"
                        title="复制内容"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              </motion.div>
              );
            })}

            {isTyping && (
              <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="flex justify-start">
                <div className="flex gap-4">
                  <div className="w-8 h-8 rounded-full bg-primary-500 flex items-center justify-center">
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                    </svg>
                  </div>
                  <div className="bg-white dark:bg-neutral-700 border border-neutral-100 dark:border-neutral-600 px-6 py-4 rounded-[24px] rounded-tl-none shadow-sm">
                    <div className="flex gap-1.5">
                      <motion.div animate={{ scale: [1, 1.2, 1] }} transition={{ repeat: Infinity, duration: 1 }} className="w-1.5 h-1.5 bg-primary-400 rounded-full" />
                      <motion.div animate={{ scale: [1, 1.2, 1] }} transition={{ repeat: Infinity, duration: 1, delay: 0.2 }} className="w-1.5 h-1.5 bg-primary-400 rounded-full" />
                      <motion.div animate={{ scale: [1, 1.2, 1] }} transition={{ repeat: Infinity, duration: 1, delay: 0.4 }} className="w-1.5 h-1.5 bg-primary-400 rounded-full" />
                    </div>
                    <p className="mt-3 text-[11px] font-bold text-neutral-500 dark:text-neutral-400">
                      AI 助手处理中{isAssistantProcessing ? ` · 已耗时 ${processingDurationLabel}` : '...'}
                    </p>
                  </div>
                </div>
              </motion.div>
            )}
          </div>

          <div className="px-4 sm:px-8 lg:px-10 py-5 lg:py-8 bg-white dark:bg-neutral-800 border-t border-neutral-100 dark:border-neutral-700 rounded-b-[32px]">
            <form onSubmit={handleSubmit} className="relative group">
              <input
                type="text"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder={requiresUserInput && pendingQuestion ? pendingQuestion : t('chat.messagePlaceholder')}
                className="w-full bg-neutral-50 dark:bg-neutral-700 border-2 border-neutral-100 dark:border-neutral-600 rounded-2xl py-4 px-6 pr-16 text-sm font-bold text-neutral-900 dark:text-neutral-100 placeholder-neutral-400 dark:placeholder-neutral-500 transition-all focus:outline-none focus:border-primary-500 focus:bg-white dark:focus:bg-neutral-600 focus:shadow-elevated"
                disabled={isTyping}
              />
              <button
                type="submit"
                disabled={isTyping || !input.trim()}
                className="absolute right-3 top-1/2 -translate-y-1/2 w-10 h-10 bg-neutral-900 dark:bg-primary-600 text-white rounded-xl flex items-center justify-center transition-all hover:bg-primary-600 dark:hover:bg-primary-500 disabled:opacity-20 active:scale-90"
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                  <path d="M10.894 2.553a1 1 0 00-1.788 0l-7 14a1 1 0 001.169 1.409l5-1.429A1 1 0 009 15.571V11a1 1 0 112 0v4.571a1 1 0 00.725.962l5 1.428a1 1 0 001.17-1.408l-7-14z" />
                </svg>
              </button>
            </form>
            <p className="text-[10px] text-center text-neutral-300 dark:text-neutral-600 font-bold uppercase tracking-widest mt-4">
              {t('chat.poweredBy')}
            </p>
          </div>
        </main>
      </div>

      <ConfirmModal
        isOpen={deleteId !== null}
        title={t('chat.deleteSession')}
        message={t('chat.deleteConfirm')}
        onConfirm={handleDeleteConfirm}
        onCancel={() => setDeleteId(null)}
        isDanger
      />
    </Layout>
  );
};
