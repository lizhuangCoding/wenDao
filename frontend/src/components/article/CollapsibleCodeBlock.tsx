import { type ComponentProps, type ReactNode, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { cn } from '@/utils';

const MAX_COLLAPSED_CODE_LINES = 18;
const COLLAPSED_CODE_MAX_HEIGHT = '28rem';

const getTextContent = (node: ReactNode): string => {
  if (typeof node === 'string') return node;
  if (typeof node === 'number') return String(node);
  if (!node) return '';
  if (Array.isArray(node)) return node.map(getTextContent).join('');
  if (typeof node === 'object' && 'props' in node) {
    return getTextContent((node as React.ReactElement).props.children);
  }
  return '';
};

const getLineCount = (text: string) => {
  const normalizedText = text.trimEnd();
  if (!normalizedText) return 0;
  return normalizedText.split(/\r?\n/).length;
};

export const CollapsibleCodeBlock = ({ className, children, ...props }: ComponentProps<'pre'>) => {
  const { t } = useTranslation();
  const codeText = getTextContent(children);
  const lineCount = getLineCount(codeText);
  const shouldCollapse = lineCount > MAX_COLLAPSED_CODE_LINES || codeText.length > 1200;
  const [isExpanded, setIsExpanded] = useState(() => !shouldCollapse);
  const lineNumbers = useMemo(() => {
    return Array.from({ length: Math.max(1, lineCount) }, (_, index) => index + 1);
  }, [lineCount]);

  useEffect(() => {
    setIsExpanded(!shouldCollapse);
  }, [codeText, shouldCollapse]);

  const isCollapsed = shouldCollapse && !isExpanded;
  const buttonLabel = isExpanded ? t('codeBlock.collapse') : t('codeBlock.expand');

  return (
    <div className="relative my-8 overflow-hidden rounded-2xl border border-neutral-800 bg-neutral-950 shadow-elevated dark:border-neutral-700 dark:bg-black">
      <div
        className="grid min-w-0 grid-cols-[3.5rem_minmax(0,1fr)] overflow-hidden"
        style={
          isCollapsed
            ? {
                maxHeight: COLLAPSED_CODE_MAX_HEIGHT,
              }
            : undefined
        }
      >
        <div className="select-none border-r border-neutral-800 bg-neutral-950 px-3 py-5 text-right font-mono text-[11px] leading-7 text-neutral-500 dark:border-neutral-700 dark:bg-black dark:text-neutral-600">
          {lineNumbers.map((lineNumber) => (
            <div key={lineNumber} className="h-7">
              {lineNumber}
            </div>
          ))}
        </div>

        <pre
          {...props}
          className={cn(
            'relative !m-0 !min-w-0 !overflow-x-auto !rounded-none !border-0 !bg-transparent !p-5 !text-sm !leading-7 !text-neutral-100 !shadow-none dark:!text-neutral-100',
            className
          )}
        >
          {children}
        </pre>
      </div>

      {shouldCollapse && (
        <>
          {isCollapsed && (
            <div className="pointer-events-none absolute inset-x-0 bottom-0 h-24 rounded-b-2xl bg-gradient-to-t from-neutral-950 to-transparent dark:from-black" />
          )}
          <button
            type="button"
            aria-expanded={isExpanded}
            onClick={() => setIsExpanded((value) => !value)}
            className="absolute bottom-4 right-4 z-10 inline-flex items-center gap-2 rounded-full border border-neutral-700 bg-neutral-900/95 px-3 py-2 text-xs font-bold text-neutral-100 shadow-lg transition-colors hover:border-primary-500 hover:bg-neutral-800 dark:border-neutral-600 dark:bg-neutral-950/95"
          >
            {isExpanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
            {buttonLabel}
          </button>
        </>
      )}
    </div>
  );
};
