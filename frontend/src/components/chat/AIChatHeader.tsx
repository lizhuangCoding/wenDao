import { Check, Copy, Download, Share2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

interface AIChatHeaderProps {
  canManageConversation: boolean;
  draftTitle: string;
  hasMessages: boolean;
  isExporting: boolean;
  isHistoryDrawerOpen?: boolean;
  isImmersive: boolean;
  isRenaming: boolean;
  isShared: boolean;
  isSharing: boolean;
  shareCopied: boolean;
  title: string;
  onCopyShareLink: () => void;
  onDraftTitleChange: (value: string) => void;
  onExport: () => void;
  onOpenHistory: () => void;
  onRenameCancel: () => void;
  onRenameSave: () => Promise<void> | void;
  onRenameStart: () => void;
  onToggleImmersive: () => void;
  onToggleShare: () => void;
}

export const AIChatHeader = ({
  canManageConversation,
  draftTitle,
  hasMessages,
  isExporting,
  isImmersive,
  isRenaming,
  isShared,
  isSharing,
  shareCopied,
  title,
  onCopyShareLink,
  onDraftTitleChange,
  onExport,
  onOpenHistory,
  onRenameCancel,
  onRenameSave,
  onRenameStart,
  onToggleImmersive,
  onToggleShare,
}: AIChatHeaderProps) => {
  const { t } = useTranslation();

  return (
    <header className={`px-4 sm:px-8 lg:px-10 py-4 lg:py-6 border-b border-neutral-200 dark:border-neutral-700 flex flex-col items-stretch gap-3 bg-white dark:bg-neutral-800 z-10 sm:flex-row sm:items-center sm:justify-between ${
      isImmersive ? 'rounded-none' : 'rounded-t-2xl sm:rounded-[32px]'
    }`}>
      <div className="flex items-center gap-3 min-w-0">
        <button
          type="button"
          onClick={onOpenHistory}
          className="lg:hidden h-9 w-9 sm:w-10 sm:h-10 flex shrink-0 items-center justify-center rounded-xl border border-neutral-200 dark:border-neutral-700 text-neutral-500 dark:text-neutral-300 hover:text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-colors"
          aria-label={t('chat.openHistory')}
        >
          <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h10M4 18h16" />
          </svg>
        </button>
        <div className="min-w-0">
          {isRenaming && canManageConversation ? (
            <div className="flex min-w-0 flex-wrap items-center gap-2 sm:gap-3">
              <input
                value={draftTitle}
                onChange={(event) => onDraftTitleChange(event.target.value)}
                placeholder={t('chat.renamePlaceholder')}
                className="min-w-0 flex-1 bg-transparent border border-neutral-200 dark:border-neutral-600 rounded-lg px-3 py-2 text-base sm:text-lg font-serif font-black text-neutral-900 dark:text-neutral-100"
                onKeyDown={async (event) => {
                  if (event.key === 'Enter' && draftTitle.trim()) {
                    await onRenameSave();
                  }
                  if (event.key === 'Escape') {
                    onRenameCancel();
                  }
                }}
                onBlur={() => {
                  if (draftTitle.trim() && draftTitle !== title) {
                    void onRenameSave();
                    return;
                  }
                  onRenameCancel();
                }}
                autoFocus
              />
              <button type="button" onClick={() => void onRenameSave()} className="text-xs text-primary-600 dark:text-primary-400">
                {t('chat.saveName')}
              </button>
              <button
                type="button"
                onClick={onRenameCancel}
                className="text-xs text-neutral-500 dark:text-neutral-400"
              >
                {t('chat.cancelRename')}
              </button>
            </div>
          ) : (
            <div className="flex min-w-0 items-center gap-3">
              <h2 className="text-lg font-serif font-black text-neutral-900 dark:text-neutral-100 truncate">
                {title || t('chat.title')}
              </h2>
              {canManageConversation && (
                <button
                  type="button"
                  onClick={onRenameStart}
                  className="text-xs text-neutral-400 hover:text-primary-600 dark:hover:text-primary-400"
                >
                  {t('chat.rename')}
                </button>
              )}
            </div>
          )}
        </div>
      </div>

      <div className="flex max-w-full items-center gap-2 overflow-x-auto pb-1 sm:justify-end sm:overflow-visible sm:pb-0">
        {canManageConversation && (
          <>
            <button
              type="button"
              onClick={onExport}
              disabled={isExporting || !hasMessages}
              className="inline-flex shrink-0 items-center gap-1.5 rounded-xl border border-neutral-200 dark:border-neutral-700 px-2.5 py-2 sm:px-3 text-xs font-bold text-neutral-500 dark:text-neutral-300 hover:text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-colors disabled:opacity-30"
              title={t('chat.exportConversation')}
            >
              <Download className="h-4 w-4" aria-hidden="true" />
              <span className="hidden sm:inline">{isExporting ? t('chat.exportingConversation') : t('chat.exportConversation')}</span>
            </button>

            {isShared ? (
              <div className="flex items-center gap-1">
                <button
                  type="button"
                  onClick={onCopyShareLink}
                  className="inline-flex shrink-0 items-center gap-1.5 rounded-xl border border-primary-200 dark:border-primary-800 bg-primary-50 dark:bg-primary-900/20 px-2.5 py-2 sm:px-3 text-xs font-bold text-primary-700 dark:text-primary-300 hover:bg-primary-100 dark:hover:bg-primary-900/40 transition-colors"
                  title={t('chat.copyShareLink')}
                >
                  {shareCopied ? (
                    <Check className="h-4 w-4" aria-hidden="true" />
                  ) : (
                    <Copy className="h-4 w-4" aria-hidden="true" />
                  )}
                  <span className="hidden sm:inline">{shareCopied ? t('chat.copiedShareLink') : t('chat.copyShareLink')}</span>
                </button>
                <button
                  type="button"
                  onClick={onToggleShare}
                  disabled={isSharing}
                  className="inline-flex shrink-0 items-center gap-1.5 rounded-xl border border-neutral-200 dark:border-neutral-700 px-2 py-2 sm:px-2.5 text-xs font-bold text-neutral-400 hover:text-red-500 hover:border-red-200 dark:hover:border-red-800 transition-colors disabled:opacity-30"
                  title={t('chat.cancelShare')}
                >
                  <span className="hidden sm:inline">{isSharing ? '...' : t('chat.cancelShare')}</span>
                </button>
              </div>
            ) : (
              <button
                type="button"
                onClick={onToggleShare}
                disabled={isSharing || !hasMessages}
                className="inline-flex shrink-0 items-center gap-1.5 rounded-xl border border-neutral-200 dark:border-neutral-700 px-2.5 py-2 sm:px-3 text-xs font-bold text-neutral-500 dark:text-neutral-300 hover:text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-colors disabled:opacity-30"
                title={t('chat.shareConversation')}
              >
                <Share2 className="h-4 w-4" aria-hidden="true" />
                <span className="hidden sm:inline">{isSharing ? '...' : t('chat.shareConversation')}</span>
              </button>
            )}
          </>
        )}

        <button
          type="button"
          onClick={onToggleImmersive}
          className="inline-flex shrink-0 items-center gap-2 rounded-xl border border-neutral-200 dark:border-neutral-700 px-2.5 py-2 sm:px-3 text-xs font-bold text-neutral-500 dark:text-neutral-300 hover:text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-colors"
          title={isImmersive ? t('chat.exitImmersiveMode') : t('chat.immersiveMode')}
        >
          <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            {isImmersive ? (
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 3H5a2 2 0 00-2 2v3m16 0V5a2 2 0 00-2-2h-3m0 18h3a2 2 0 002-2v-3M5 16v3a2 2 0 002 2h3" />
            ) : (
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V5a1 1 0 011-1h3m8 0h3a1 1 0 011 1v3m0 8v3a1 1 0 01-1 1h-3M8 20H5a1 1 0 01-1-1v-3" />
            )}
          </svg>
          <span className="hidden sm:inline">{isImmersive ? t('chat.exitImmersiveMode') : t('chat.immersiveMode')}</span>
        </button>
      </div>
    </header>
  );
};
