import { type ComponentProps, type ReactNode, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Check, ChevronDown, ChevronUp, Copy } from 'lucide-react';
import { cn } from '@/utils';

const MAX_COLLAPSED_CODE_LINES = 18;
const COLLAPSED_CODE_MAX_HEIGHT = '28rem';
const COPY_RESET_TIMEOUT_MS = 1600;

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
  const [hasCopied, setHasCopied] = useState(false);
  const copyResetTimerRef = useRef<number>();
  const lineNumbers = useMemo(() => {
    return Array.from({ length: Math.max(1, lineCount) }, (_, index) => index + 1);
  }, [lineCount]);

  useEffect(() => {
    setIsExpanded(!shouldCollapse);
  }, [codeText, shouldCollapse]);

  useEffect(() => {
    setHasCopied(false);
    return () => {
      if (copyResetTimerRef.current) {
        window.clearTimeout(copyResetTimerRef.current);
      }
    };
  }, [codeText]);

  const handleCopy = async () => {
    if (!codeText) return;
    try {
      await navigator.clipboard.writeText(codeText);
      setHasCopied(true);
      if (copyResetTimerRef.current) {
        window.clearTimeout(copyResetTimerRef.current);
      }
      copyResetTimerRef.current = window.setTimeout(() => {
        setHasCopied(false);
      }, COPY_RESET_TIMEOUT_MS);
    } catch {
      setHasCopied(false);
    }
  };

  const isCollapsed = shouldCollapse && !isExpanded;
  const buttonLabel = isExpanded ? t('codeBlock.collapse') : t('codeBlock.expand');
  const copyLabel = hasCopied ? t('codeBlock.copied') : t('codeBlock.copy');
  const toggleButtonClassName =
    'inline-flex items-center gap-1.5 rounded-full border border-neutral-700 bg-neutral-900/95 px-2.5 py-1 text-xs font-semibold text-neutral-100 shadow-sm transition-colors hover:border-primary-500 hover:bg-neutral-800 focus:outline-none focus:ring-2 focus:ring-primary-400 focus:ring-offset-2 focus:ring-offset-neutral-950 dark:border-neutral-600 dark:bg-neutral-950/95';
  const toggleButton = (
    <button
      type="button"
      aria-expanded={isExpanded}
      onClick={() => setIsExpanded((value) => !value)}
      className={toggleButtonClassName}
    >
      {isExpanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
      {buttonLabel}
    </button>
  );

  return (
    <div className="relative my-8 overflow-hidden rounded-2xl border border-neutral-800 bg-neutral-950 shadow-elevated dark:border-neutral-700 dark:bg-black">
      <button
        type="button"
        aria-label={copyLabel}
        title={copyLabel}
        onClick={handleCopy}
        className="absolute top-2 right-2 z-20 inline-flex h-8 items-center gap-1.5 rounded-md border border-white/10 bg-neutral-900/85 px-2.5 text-xs font-medium text-neutral-200 shadow-sm backdrop-blur transition-colors hover:bg-neutral-800 hover:text-white focus:outline-none focus:ring-2 focus:ring-primary-400 focus:ring-offset-2 focus:ring-offset-neutral-950 dark:border-white/10 dark:bg-neutral-900/90"
      >
        {hasCopied ? <Check className="h-4 w-4 text-emerald-400" /> : <Copy className="h-4 w-4" />}
        <span>{copyLabel}</span>
      </button>

      <div
        className="grid min-w-0 grid-cols-[2.25rem_minmax(0,1fr)] overflow-hidden"
        style={
          isCollapsed
            ? {
                maxHeight: COLLAPSED_CODE_MAX_HEIGHT,
              }
            : undefined
        }
      >
        <div className="select-none border-r border-neutral-800 bg-neutral-950 px-1 py-2 text-right font-mono text-[11px] leading-5 text-neutral-500 dark:border-neutral-700 dark:bg-black dark:text-neutral-600">
          {lineNumbers.map((lineNumber) => (
            <div key={lineNumber} className="h-5">
              {lineNumber}
            </div>
          ))}
        </div>

        <pre
          {...props}
          className={cn(
            'relative !m-0 !min-w-0 !overflow-x-auto !rounded-none !border-0 !bg-transparent !px-2 !py-2 !text-[13px] !leading-5 !text-neutral-100 !shadow-none dark:!text-neutral-100',
            className
          )}
        >
          {children}
        </pre>
      </div>

      {shouldCollapse && (
        <>
          {isCollapsed && (
            <div className="pointer-events-none absolute inset-x-0 bottom-0 h-16 rounded-b-2xl bg-gradient-to-t from-neutral-950 to-transparent dark:from-black" />
          )}
          <div className="absolute inset-x-0 bottom-2 z-10 flex justify-center">{toggleButton}</div>
        </>
      )}
    </div>
  );
};
