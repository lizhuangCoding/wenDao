import { useEffect, useRef, useState } from 'react';
import { chatApi } from '@/api';
import type { Conversation } from '@/store/chatNormalizers';

const CHAT_SIDEBAR_STORAGE_KEY = 'wendao.aiChat.sidebar';
const CHAT_IMMERSIVE_STORAGE_KEY = 'wendao.aiChat.immersive';

interface UseAIChatPageStateParams {
  activeChat: Conversation | null;
  messageCount: number;
  deleteChat: (id: number) => Promise<void>;
  renameChat: (id: number, title: string) => Promise<void>;
  setActiveChat: (id: number) => Promise<void>;
  showToast: (message: string, theme: 'success' | 'error' | 'info') => void;
  t: (key: string, options?: Record<string, unknown>) => string;
  updateConversationShare: (id: number, isShared: boolean, shareToken?: string) => void;
}

type RenameTarget = Pick<Conversation, 'id' | 'title'>;

export const useAIChatPageState = ({
  activeChat,
  messageCount,
  deleteChat,
  renameChat,
  setActiveChat,
  showToast,
  t,
  updateConversationShare,
}: UseAIChatPageStateParams) => {
  const [isRenaming, setIsRenaming] = useState(false);
  const [draftTitle, setDraftTitle] = useState('');
  const [activeMenuId, setActiveMenuId] = useState<number | null>(null);
  const [deleteId, setDeleteId] = useState<number | null>(null);
  const [isDeletingChat, setIsDeletingChat] = useState(false);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(() => {
    if (typeof window === 'undefined') return false;
    return window.localStorage.getItem(CHAT_SIDEBAR_STORAGE_KEY) === 'collapsed';
  });
  const [isImmersive, setIsImmersive] = useState(() => {
    if (typeof window === 'undefined') return false;
    return window.localStorage.getItem(CHAT_IMMERSIVE_STORAGE_KEY) === 'immersive';
  });
  const [isHistoryDrawerOpen, setIsHistoryDrawerOpen] = useState(false);
  const [isSharing, setIsSharing] = useState(false);
  const [shareCopied, setShareCopied] = useState(false);
  const [isExporting, setIsExporting] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

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

  useEffect(() => {
    if (activeChat?.title && !isRenaming) {
      setDraftTitle(activeChat.title);
    }
  }, [activeChat?.title, isRenaming]);

  const handleRenameSave = async () => {
    if (!activeChat || !draftTitle.trim()) return;
    await renameChat(activeChat.id, draftTitle.trim());
    setIsRenaming(false);
  };

  const handleRenameCancel = () => {
    setDraftTitle(activeChat?.title || '');
    setIsRenaming(false);
  };

  const handleStartRename = async (chat: RenameTarget | null = activeChat) => {
    if (!chat) return;
    await setActiveChat(chat.id);
    setDraftTitle(chat.title);
    setIsRenaming(true);
  };

  const handleDeleteConfirm = async () => {
    if (deleteId && !isDeletingChat) {
      setIsDeletingChat(true);
      try {
        await deleteChat(deleteId);
        setDeleteId(null);
        showToast(t('chat.deleteSuccess'), 'success');
      } catch (error) {
        const message = error instanceof Error ? error.message : t('chat.deleteFailed');
        showToast(message || t('chat.deleteFailed'), 'error');
      } finally {
        setIsDeletingChat(false);
      }
    }
  };

  const handleToggleShare = async () => {
    if (!activeChat || isSharing) return;

    setIsSharing(true);
    try {
      const result = await chatApi.shareConversation(activeChat.id, !activeChat.isShared);
      updateConversationShare(activeChat.id, result.is_shared, result.share_token);
      showToast(result.is_shared ? t('chat.shareEnabled') : t('chat.shareDisabled'), 'success');
    } catch (error) {
      const message = error instanceof Error ? error.message : t('article.operationFailed');
      showToast(message || t('article.operationFailed'), 'error');
    } finally {
      setIsSharing(false);
    }
  };

  const handleCopyShareLink = async () => {
    const token = activeChat?.shareToken;
    if (!token) return;

    const url = `${window.location.origin}/shared/${token}`;
    try {
      await navigator.clipboard.writeText(url);
      setShareCopied(true);
      showToast(t('chat.copySuccess'), 'success');
      setTimeout(() => setShareCopied(false), 2000);
    } catch {
      showToast(t('chat.copyFailed'), 'error');
    }
  };

  const handleExport = async () => {
    if (!activeChat || isExporting) return;

    setIsExporting(true);
    try {
      await chatApi.exportConversation(activeChat.id, activeChat.title || 'conversation');
      showToast(t('chat.exportSuccess'), 'success');
    } catch (error) {
      const message = error instanceof Error ? error.message : t('chat.exportFailed');
      showToast(message || t('chat.exportFailed'), 'error');
    } finally {
      setIsExporting(false);
    }
  };

  return {
    activeMenuId,
    deleteId,
    draftTitle,
    isDeletingChat,
    isExporting,
    isHistoryDrawerOpen,
    isImmersive,
    isRenaming,
    isShared: Boolean(activeChat?.isShared),
    isSharing,
    isSidebarCollapsed,
    menuRef,
    shareCopied,
    setActiveMenuId,
    setDeleteId,
    setDraftTitle,
    setIsHistoryDrawerOpen,
    setIsSidebarCollapsed,
    toggleImmersive: () => setIsImmersive((value) => !value),
    handleCopyShareLink,
    handleDeleteConfirm,
    handleExport,
    handleRenameCancel,
    handleRenameSave,
    handleStartRename,
    handleToggleShare,
    hasMessages: messageCount > 0,
  };
};
