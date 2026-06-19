import { useTranslation } from 'react-i18next';
import type { ChatQuestionNavItem } from '@/utils/chatQuestionNavigator';

interface ChatQuestionNavigatorProps {
  activeId: string | null;
  items: ChatQuestionNavItem[];
  onSelect: (anchorId: string) => void;
}

const formatQuestionTime = (timestamp?: number) => {
  if (!timestamp) return '';
  return new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
};

export const ChatQuestionNavigator = ({ activeId, items, onSelect }: ChatQuestionNavigatorProps) => {
  const { t } = useTranslation();
  if (items.length < 2) return null;

  return (
    <nav
      aria-label={t('chat.questionDirectory')}
      className="group absolute bottom-36 right-3 top-28 z-20 hidden xl:flex items-stretch justify-end"
    >
      <div className="relative h-full w-8 transition-[width] duration-200 ease-out group-hover:w-80 group-focus-within:w-80">
        <div className="pointer-events-none absolute inset-y-6 right-3 flex w-2 flex-col items-center justify-between">
          <div className="absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-neutral-200 dark:bg-neutral-700" />
          {items.slice(0, 14).map((item) => {
            const isActive = item.anchorId === activeId;
            return (
              <span
                key={item.anchorId}
                className={`relative h-1.5 w-1.5 rounded-full transition-colors ${
                  isActive ? 'bg-primary-500' : 'bg-neutral-300 dark:bg-neutral-600'
                }`}
              />
            );
          })}
        </div>

        <div className="absolute inset-y-0 right-0 w-80 translate-x-4 rounded-2xl border border-neutral-200 bg-white/95 p-3 opacity-0 shadow-elevated backdrop-blur transition-all duration-200 group-hover:translate-x-0 group-hover:opacity-100 group-focus-within:translate-x-0 group-focus-within:opacity-100 dark:border-neutral-700 dark:bg-neutral-900/95">
          <div className="mb-3 flex items-center justify-between px-1">
            <p className="text-xs font-black tracking-wider text-neutral-800 dark:text-neutral-100">
              {t('chat.questionDirectory')}
            </p>
            <span className="rounded-full bg-neutral-100 px-2 py-0.5 text-[10px] font-bold text-neutral-500 dark:bg-neutral-800 dark:text-neutral-400">
              {items.length}
            </span>
          </div>

          <div className="h-[calc(100%-2.25rem)] space-y-1 overflow-y-auto pr-1 scrollbar-hide">
            {items.map((item) => {
              const isActive = item.anchorId === activeId;
              const time = formatQuestionTime(item.timestamp);

              return (
                <button
                  key={item.anchorId}
                  type="button"
                  title={item.fullText}
                  aria-current={isActive ? 'true' : undefined}
                  onClick={() => onSelect(item.anchorId)}
                  className={`group/item flex w-full items-start gap-3 rounded-xl px-3 py-2.5 text-left transition-colors ${
                    isActive
                      ? 'bg-primary-50 text-primary-900 ring-1 ring-primary-100 dark:bg-primary-900/20 dark:text-primary-100 dark:ring-primary-800/60'
                      : 'text-neutral-500 hover:bg-neutral-50 hover:text-neutral-900 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100'
                  }`}
                >
                  <span className={`mt-1 flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full text-[10px] font-black ${
                    isActive
                      ? 'bg-primary-500 text-white'
                      : 'bg-neutral-100 text-neutral-400 group-hover/item:bg-neutral-200 dark:bg-neutral-800 dark:text-neutral-500'
                  }`}>
                    {item.index}
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="line-clamp-2 text-xs font-bold leading-5">
                      {item.label}
                    </span>
                    {time && (
                      <span className="mt-0.5 block text-[10px] font-semibold text-neutral-400 dark:text-neutral-500">
                        {time}
                      </span>
                    )}
                  </span>
                </button>
              );
            })}
          </div>
        </div>
      </div>
    </nav>
  );
};
