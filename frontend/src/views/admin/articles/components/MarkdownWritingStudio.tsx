import {
  type ClipboardEvent,
  type RefObject,
  Suspense,
  lazy,
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react';
import { ColorPicker } from 'tdesign-react';
import {
  Bold,
  ChevronDown,
  Code,
  Eye,
  Heading2,
  Heading3,
  Heading4,
  ImagePlus,
  Link as LinkIcon,
  List,
  ListOrdered,
  Maximize2,
  Minimize2,
  Minus,
  Palette,
  PanelLeft,
  Pilcrow,
  Quote,
  SplitSquareHorizontal,
  Sparkles,
  type LucideIcon,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { AIWritingAction } from '@/api/chat';
import {
  DEFAULT_TEXT_COLOR,
  applyMarkdownAction,
  applyMarkdownColor,
  getSynchronizedScrollTop,
  normalizeMarkdownColor,
  type ApplyMarkdownActionResult,
  type MarkdownAction,
} from '@/utils/markdownEditor';

const ArticlePreview = lazy(() =>
  import('../ArticlePreview').then((module) => ({ default: module.ArticlePreview }))
);

type EditorMode = 'edit' | 'split' | 'preview';

interface ContentStats {
  characters: number;
  lines: number;
  words: number;
  readingMinutes: number;
}

interface MarkdownWritingStudioProps {
  content: string;
  onContentChange: (content: string) => void;
  textareaRef: RefObject<HTMLTextAreaElement>;
  onPaste: (event: ClipboardEvent<HTMLTextAreaElement>) => void;
  onSelect?: () => void;
  onKeyUp?: () => void;
  onMouseUp?: () => void;
  onImageUploadClick: () => void;
  allowImageUpload?: boolean;
  helperText?: string;
  placeholder?: string;
  contentStats: ContentStats;
  lastSavedTime: string | null;
  isAutoSaving: boolean;
  isImmersive: boolean;
  onImmersiveChange: (isImmersive: boolean) => void;
  aiSummaryPanel?: {
    isGenerating: boolean;
    result: string;
  } | null;
  aiWritingPanel?: {
    action: AIWritingAction;
    isGenerating: boolean;
    result: string;
    suggestions: string[];
  } | null;
  onGenerateSummary?: () => void;
  onApplySummary?: () => void;
  onGenerateWritingAction?: (action: AIWritingAction) => void;
  onApplyWritingResult?: (result: string) => void;
}

const markdownToolbarActions: Array<{
  action: MarkdownAction;
  labelKey: string;
  icon: LucideIcon;
}> = [
  { action: 'heading-2', labelKey: 'articleEditor.toolbarHeading2', icon: Heading2 },
  { action: 'heading-3', labelKey: 'articleEditor.toolbarHeading3', icon: Heading3 },
  { action: 'heading-4', labelKey: 'articleEditor.toolbarHeading4', icon: Heading4 },
  { action: 'bold', labelKey: 'articleEditor.toolbarBold', icon: Bold },
  { action: 'quote', labelKey: 'articleEditor.toolbarQuote', icon: Quote },
  { action: 'unordered-list', labelKey: 'articleEditor.toolbarUnorderedList', icon: List },
  { action: 'unordered-list-indented', labelKey: 'articleEditor.toolbarNestedUnorderedList', icon: List },
  { action: 'ordered-list', labelKey: 'articleEditor.toolbarOrderedList', icon: ListOrdered },
  { action: 'inline-code', labelKey: 'articleEditor.toolbarInlineCode', icon: Code },
  { action: 'code-block', labelKey: 'articleEditor.toolbarCodeBlock', icon: Pilcrow },
  { action: 'link', labelKey: 'articleEditor.toolbarLink', icon: LinkIcon },
  { action: 'divider', labelKey: 'articleEditor.toolbarDivider', icon: Minus },
];

const TEXT_COLOR_PRESETS = [
  { labelKey: 'articleEditor.colorRed', value: '#ef4444' },
  { labelKey: 'articleEditor.colorOrange', value: '#f97316' },
  { labelKey: 'articleEditor.colorAmber', value: '#f59e0b' },
  { labelKey: 'articleEditor.colorGreen', value: '#10b981' },
  { labelKey: 'articleEditor.colorSky', value: '#0ea5e9' },
  { labelKey: 'articleEditor.colorIndigo', value: '#6366f1' },
  { labelKey: 'articleEditor.colorPink', value: '#ec4899' },
  { labelKey: 'articleEditor.colorGray', value: '#525252' },
];

const restoreTextareaSelection = (
  textarea: HTMLTextAreaElement,
  result: ApplyMarkdownActionResult
) => {
  requestAnimationFrame(() => {
    textarea.focus();
    textarea.setSelectionRange(result.selection.start, result.selection.end);
  });
};

const insertMarkdownWithUndoStack = (
  textarea: HTMLTextAreaElement,
  result: ApplyMarkdownActionResult
) => {
  textarea.focus();
  textarea.setSelectionRange(result.edit.start, result.edit.end);

  try {
    const canInsertText =
      !document.queryCommandSupported || document.queryCommandSupported('insertText');
    if (!canInsertText) return false;

    const didInsert = document.execCommand('insertText', false, result.edit.replacement);
    if (!didInsert) return false;

    const inputEvent =
      typeof InputEvent === 'function'
        ? new InputEvent('input', {
            bubbles: true,
            inputType: 'insertText',
            data: result.edit.replacement,
          })
        : new Event('input', { bubbles: true });
    textarea.dispatchEvent(inputEvent);
    restoreTextareaSelection(textarea, result);
    return true;
  } catch {
    return false;
  }
};

const tooltipClassName =
  'pointer-events-none absolute left-1/2 top-full z-30 mt-2 -translate-x-1/2 whitespace-nowrap rounded-md border border-neutral-200 bg-white px-2 py-1 text-[11px] font-medium text-neutral-600 opacity-0 shadow-sm transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200';

const parseAnchorLine = (element: Element) => {
  const line = Number.parseInt(element.getAttribute('data-md-line') || '', 10);
  return Number.isFinite(line) ? line : undefined;
};

const getPreviewAnchors = (preview: HTMLElement) => {
  const previewRect = preview.getBoundingClientRect();
  const anchorsByLine = new Map<number, number>();

  Array.from(preview.querySelectorAll<HTMLElement>('[data-md-line]')).forEach((element) => {
    const line = parseAnchorLine(element);
    if (typeof line !== 'number') return;

    const top = element.getBoundingClientRect().top - previewRect.top + preview.scrollTop;
    const currentTop = anchorsByLine.get(line);
    anchorsByLine.set(line, currentTop === undefined ? top : Math.min(currentTop, top));
  });

  return Array.from(anchorsByLine.entries())
    .map(([line, top]) => ({ line, top }))
    .sort((a, b) => a.line - b.line);
};

const findLastAnchor = <T,>(items: T[], predicate: (item: T) => boolean) => {
  for (let index = items.length - 1; index >= 0; index -= 1) {
    if (predicate(items[index])) return items[index];
  }
  return undefined;
};

const clampScrollTop = (scrollTop: number, target: HTMLElement) => {
  const maxScrollTop = Math.max(0, target.scrollHeight - target.clientHeight);
  return Math.min(maxScrollTop, Math.max(0, scrollTop));
};

const escapeHtml = (text: string) => {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
};

const getMarkdownScrollAnchorLines = (content: string) => {
  const lines = content.split(/\r\n|\r|\n/);
  const anchors = new Set<number>([1]);
  let isInFence = false;

  lines.forEach((line, index) => {
    const lineNumber = index + 1;
    const trimmed = line.trim();
    const previousTrimmed = lines[index - 1]?.trim() ?? '';

    if (/^(```|~~~)/.test(trimmed)) {
      anchors.add(lineNumber);
      isInFence = !isInFence;
      return;
    }

    if (isInFence || !trimmed) return;

    if (
      /^#{1,6}\s+/.test(trimmed) ||
      /^([-*+]|\d+[.)])\s+/.test(trimmed) ||
      /^>\s?/.test(trimmed) ||
      /^!\[[^\]]*]\([^)]+\)/.test(trimmed) ||
      /^([-*_]\s*){3,}$/.test(trimmed) ||
      /^\|.*\|$/.test(trimmed) ||
      previousTrimmed === ''
    ) {
      anchors.add(lineNumber);
    }
  });

  anchors.add(lines.length);
  return Array.from(anchors).sort((a, b) => a - b);
};

const syncEditorMirrorStyles = (
  textarea: HTMLTextAreaElement,
  mirror: HTMLDivElement
) => {
  const styles = window.getComputedStyle(textarea);

  mirror.style.position = 'absolute';
  mirror.style.left = '-10000px';
  mirror.style.top = '0';
  mirror.style.visibility = 'hidden';
  mirror.style.pointerEvents = 'none';
  mirror.style.width = `${textarea.clientWidth}px`;
  mirror.style.boxSizing = styles.boxSizing;
  mirror.style.padding = styles.padding;
  mirror.style.border = styles.border;
  mirror.style.font = styles.font;
  mirror.style.lineHeight = styles.lineHeight;
  mirror.style.letterSpacing = styles.letterSpacing;
  mirror.style.whiteSpace = 'pre-wrap';
  mirror.style.overflowWrap = 'break-word';
  mirror.style.tabSize = styles.getPropertyValue('tab-size');
};

const getEditorMarkdownAnchors = (
  textarea: HTMLTextAreaElement,
  mirror: HTMLDivElement,
  content: string
) => {
  const anchorLines = new Set(getMarkdownScrollAnchorLines(content));
  const html = content
    .split(/\r\n|\r|\n/)
    .map((line, index) => {
      const lineNumber = index + 1;
      const marker = anchorLines.has(lineNumber)
        ? `<span data-editor-md-line="${lineNumber}"></span>`
        : '';
      return `${marker}${escapeHtml(line)}`;
    })
    .join('\n');

  syncEditorMirrorStyles(textarea, mirror);
  mirror.innerHTML = html || '<span data-editor-md-line="1"></span>';
  const mirrorRect = mirror.getBoundingClientRect();

  return Array.from(mirror.querySelectorAll<HTMLElement>('[data-editor-md-line]'))
    .map((element) => ({
      line: Number.parseInt(element.getAttribute('data-editor-md-line') || '', 10),
      top: element.getBoundingClientRect().top - mirrorRect.top,
    }))
    .filter((item) => Number.isFinite(item.line))
    .sort((a, b) => a.top - b.top);
};

const getInterpolatedTopForLine = (
  anchors: Array<{ line: number; top: number }>,
  sourceLine: number
) => {
  const previousAnchor = findLastAnchor(anchors, (item) => item.line <= sourceLine) ?? anchors[0];
  const nextAnchor = anchors.find((item) => item.line > sourceLine);

  if (!nextAnchor || nextAnchor.line === previousAnchor.line) {
    return previousAnchor.top;
  }

  const progress = Math.min(
    1,
    Math.max(0, (sourceLine - previousAnchor.line) / (nextAnchor.line - previousAnchor.line))
  );

  return previousAnchor.top + progress * (nextAnchor.top - previousAnchor.top);
};

const getScrollMap = (
  editor: HTMLTextAreaElement,
  preview: HTMLElement,
  content: string,
  editorMirror: HTMLDivElement
) => {
  const editorAnchors = getEditorMarkdownAnchors(editor, editorMirror, content);
  const previewAnchors = getPreviewAnchors(preview);
  if (editorAnchors.length === 0 || previewAnchors.length === 0) return [];

  const editorMaxScrollTop = Math.max(0, editor.scrollHeight - editor.clientHeight);
  const previewMaxScrollTop = Math.max(0, preview.scrollHeight - preview.clientHeight);
  const map = editorAnchors.map((editorAnchor) => ({
    editorTop: clampScrollTop(editorAnchor.top, editor),
    previewTop: clampScrollTop(getInterpolatedTopForLine(previewAnchors, editorAnchor.line) - 16, preview),
  }));

  map.unshift({ editorTop: 0, previewTop: 0 });
  map.push({ editorTop: editorMaxScrollTop, previewTop: previewMaxScrollTop });

  return map
    .sort((a, b) => a.editorTop - b.editorTop)
    .filter((item, index, items) => index === 0 || item.editorTop !== items[index - 1].editorTop);
};

const interpolateScrollMap = (
  scrollTop: number,
  map: Array<{ sourceTop: number; targetTop: number }>
) => {
  if (map.length === 0) return undefined;

  const previousPoint = findLastAnchor(map, (item) => item.sourceTop <= scrollTop) ?? map[0];
  const nextPoint = map.find((item) => item.sourceTop > scrollTop);

  if (!nextPoint || nextPoint.sourceTop === previousPoint.sourceTop) {
    return previousPoint.targetTop;
  }

  const progress = Math.min(
    1,
    Math.max(0, (scrollTop - previousPoint.sourceTop) / (nextPoint.sourceTop - previousPoint.sourceTop))
  );

  return previousPoint.targetTop + progress * (nextPoint.targetTop - previousPoint.targetTop);
};

const syncPreviewScrollToEditorAnchor = (
  editor: HTMLTextAreaElement,
  preview: HTMLElement,
  content: string,
  editorMirror: HTMLDivElement
): number | undefined => {
  const scrollMap = getScrollMap(editor, preview, content, editorMirror).map((item) => ({
    sourceTop: item.editorTop,
    targetTop: item.previewTop,
  }));

  return interpolateScrollMap(editor.scrollTop, scrollMap);
};

const syncEditorScrollToPreviewAnchor = (
  preview: HTMLElement,
  editor: HTMLTextAreaElement,
  content: string,
  editorMirror: HTMLDivElement
): number | undefined => {
  const scrollMap = getScrollMap(editor, preview, content, editorMirror).map((item) => ({
    sourceTop: item.previewTop,
    targetTop: item.editorTop,
  }));

  return interpolateScrollMap(preview.scrollTop, scrollMap);
};

export const MarkdownWritingStudio = ({
  content,
  onContentChange,
  textareaRef,
  onPaste,
  onSelect,
  onKeyUp,
  onMouseUp,
  onImageUploadClick,
  allowImageUpload = true,
  helperText,
  placeholder,
  contentStats,
  lastSavedTime,
  isAutoSaving,
  isImmersive,
  onImmersiveChange,
  aiSummaryPanel = null,
  aiWritingPanel = null,
  onGenerateSummary = () => {},
  onApplySummary = () => {},
  onGenerateWritingAction = () => {},
  onApplyWritingResult = () => {},
}: MarkdownWritingStudioProps) => {
  const { t } = useTranslation();
  const [editorMode, setEditorMode] = useState<EditorMode>('split');
  const [selectedTextColor, setSelectedTextColor] = useState(DEFAULT_TEXT_COLOR);
  const [isAIPanelOpen, setIsAIPanelOpen] = useState(false);
  const previewScrollRef = useRef<HTMLDivElement>(null);
  const editorScrollMirrorRef = useRef<HTMLDivElement>(null);
  const scrollSyncFrameRef = useRef<number>();
  const isSyncingScrollRef = useRef(false);

  useEffect(() => {
    return () => {
      if (scrollSyncFrameRef.current) {
        cancelAnimationFrame(scrollSyncFrameRef.current);
      }
    };
  }, []);

  const syncMarkdownScroll = useCallback(
    (source: HTMLElement | null, target: HTMLElement | null) => {
      if (editorMode !== 'split' || !source || !target || isSyncingScrollRef.current) return;

      if (scrollSyncFrameRef.current) {
        cancelAnimationFrame(scrollSyncFrameRef.current);
      }

      scrollSyncFrameRef.current = requestAnimationFrame(() => {
        const editorMirror = editorScrollMirrorRef.current;
        if (!editorMirror) return;

        const anchorScrollTop =
          source === textareaRef.current
            ? syncPreviewScrollToEditorAnchor(source as HTMLTextAreaElement, target, content, editorMirror)
            : syncEditorScrollToPreviewAnchor(source, target as HTMLTextAreaElement, content, editorMirror);
        const nextScrollTop = anchorScrollTop ?? getSynchronizedScrollTop({
          sourceScrollTop: source.scrollTop,
          sourceScrollHeight: source.scrollHeight,
          sourceClientHeight: source.clientHeight,
          targetScrollHeight: target.scrollHeight,
          targetClientHeight: target.clientHeight,
        });

        if (nextScrollTop === undefined) return;

        if (Math.abs(target.scrollTop - nextScrollTop) < 1) return;

        isSyncingScrollRef.current = true;
        target.scrollTop = nextScrollTop;

        requestAnimationFrame(() => {
          isSyncingScrollRef.current = false;
        });
      });
    },
    [content, editorMode, textareaRef]
  );

  const handleEditorScroll = () => {
    syncMarkdownScroll(textareaRef.current, previewScrollRef.current);
  };

  const handlePreviewScroll = () => {
    syncMarkdownScroll(previewScrollRef.current, textareaRef.current);
  };

  const applyEdit = (result: ApplyMarkdownActionResult) => {
    const textarea = textareaRef.current;
    if (textarea && insertMarkdownWithUndoStack(textarea, result)) return;

    onContentChange(result.text);
    if (textarea) restoreTextareaSelection(textarea, result);
  };

  const handleMarkdownAction = (action: MarkdownAction) => {
    const textarea = textareaRef.current;
    const selectionStart = textarea?.selectionStart ?? content.length;
    const selectionEnd = textarea?.selectionEnd ?? selectionStart;

    applyEdit(
      applyMarkdownAction({
        text: content,
        selectionStart,
        selectionEnd,
        action,
      })
    );
  };

  const handleTextColorApply = (color = selectedTextColor) => {
    const normalizedColor = normalizeMarkdownColor(color, selectedTextColor);
    const textarea = textareaRef.current;
    const selectionStart = textarea?.selectionStart ?? content.length;
    const selectionEnd = textarea?.selectionEnd ?? selectionStart;

    setSelectedTextColor(normalizedColor);
    applyEdit(
      applyMarkdownColor(
        {
          text: content,
          selectionStart,
          selectionEnd,
        },
        normalizedColor
      )
    );
  };

  const rootClassName = isImmersive
    ? 'fixed inset-0 z-50 flex flex-col overflow-hidden bg-neutral-50 p-3 dark:bg-neutral-950 sm:p-4'
    : 'space-y-3';
  const headerClassName = isImmersive
    ? 'mb-3 flex shrink-0 flex-col gap-3 rounded-2xl border border-neutral-200 bg-white px-4 py-3 shadow-sm dark:border-neutral-800 dark:bg-neutral-900 lg:flex-row lg:items-center lg:justify-between'
    : 'flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between';
  const studioShellClassName = isImmersive
    ? 'flex min-h-0 flex-1 flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-neutral-50/90 p-3 dark:border-neutral-800 dark:bg-neutral-950/90'
    : 'rounded-2xl border border-neutral-200 bg-neutral-50/70 p-3 dark:border-neutral-800 dark:bg-neutral-950/40';
  const toolbarClassName = isImmersive
    ? 'mb-3 flex shrink-0 flex-wrap items-center gap-2'
    : 'mb-3 flex flex-wrap items-center gap-2';
  const panelGridClassName = isImmersive
    ? `grid min-h-0 flex-1 gap-4 overflow-y-auto lg:overflow-hidden ${
        editorMode === 'split' ? 'lg:grid-cols-2' : 'grid-cols-1'
      }`
    : `grid gap-4 ${editorMode === 'split' ? 'lg:grid-cols-2' : 'grid-cols-1'}`;
  const panelSizeClass = isImmersive ? 'h-full min-h-0' : 'min-h-[640px]';
  const textareaSizeClass = isImmersive ? 'min-h-0' : 'min-h-[580px]';
  const previewBodyClassName = isImmersive
    ? 'article-reading-body admin-markdown-preview min-h-0 flex-1 overflow-y-auto px-6 py-5'
    : 'article-reading-body admin-markdown-preview flex-1 overflow-y-auto px-6 py-5';
  const resolvedHelperText = helperText ?? t('articleEditor.helperText');
  const resolvedPlaceholder = placeholder ?? t('articleEditor.placeholder');

  return (
    <div className={rootClassName}>
      <div ref={editorScrollMirrorRef} aria-hidden="true" />
      <div className={headerClassName}>
        <div>
          <label className="block text-sm font-semibold text-neutral-700 dark:text-neutral-200">
            {t('articleEditor.contentLabel')}
          </label>
          <p className="mt-1 text-xs text-neutral-400 dark:text-neutral-500">
            {resolvedHelperText}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {(['edit', 'split', 'preview'] as const).map((mode) => {
            const Icon =
              mode === 'edit' ? PanelLeft : mode === 'split' ? SplitSquareHorizontal : Eye;
            return (
              <button
                key={mode}
                type="button"
                onClick={() => setEditorMode(mode)}
                className={`inline-flex h-9 items-center gap-2 rounded-xl px-3 text-xs font-semibold transition-colors ${
                  editorMode === mode
                    ? 'bg-neutral-900 text-white dark:bg-neutral-100 dark:text-neutral-900'
                    : 'bg-neutral-100 text-neutral-500 hover:text-neutral-900 dark:bg-neutral-800 dark:text-neutral-400 dark:hover:text-neutral-100'
                }`}
              >
                <Icon className="h-4 w-4" />
                {mode === 'edit'
                  ? t('articleEditor.modeEdit')
                  : mode === 'split'
                    ? t('articleEditor.modeSplit')
                    : t('articleEditor.modePreview')}
              </button>
            );
          })}
          <button
            type="button"
            onClick={() => onImmersiveChange(!isImmersive)}
            className={`inline-flex h-9 items-center gap-2 rounded-xl px-3 text-xs font-semibold transition-colors ${
              isImmersive
                ? 'bg-primary-600 text-white hover:bg-primary-700'
                : 'bg-primary-50 text-primary-700 hover:bg-primary-100 dark:bg-primary-900/20 dark:text-primary-300 dark:hover:bg-primary-900/30'
            }`}
          >
            {isImmersive ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
            {isImmersive ? t('articleEditor.focusExit') : t('articleEditor.focusEnter')}
          </button>
        </div>
      </div>

      <div className={studioShellClassName}>
        <div className={toolbarClassName}>
          {markdownToolbarActions.map((item) => (
            <button
              key={item.action}
              type="button"
              aria-label={t(item.labelKey)}
              onClick={() => handleMarkdownAction(item.action)}
              className="group relative inline-flex h-9 w-9 items-center justify-center rounded-xl text-neutral-500 transition-colors hover:bg-white hover:text-neutral-900 focus-visible:bg-white focus-visible:text-neutral-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100 dark:focus-visible:bg-neutral-800 dark:focus-visible:text-neutral-100"
            >
              <item.icon className="h-4 w-4" aria-hidden="true" />
              <span className={tooltipClassName}>{t(item.labelKey)}</span>
            </button>
          ))}

          <div className="mx-1 h-6 w-px bg-neutral-200 dark:bg-neutral-800" />

          <button
            type="button"
            aria-label={t('articleEditor.textColorApply')}
            onClick={() => handleTextColorApply()}
            className="group relative inline-flex h-9 w-9 items-center justify-center rounded-xl text-neutral-500 transition-colors hover:bg-white hover:text-neutral-900 focus-visible:bg-white focus-visible:text-neutral-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100 dark:focus-visible:bg-neutral-800 dark:focus-visible:text-neutral-100"
          >
            <Palette className="h-4 w-4" aria-hidden="true" />
            <span
              className="absolute bottom-1 h-0.5 w-5 rounded-full"
              style={{ backgroundColor: selectedTextColor }}
            />
            <span className={tooltipClassName}>{t('articleEditor.textColorTooltip')}</span>
          </button>

          <div className="flex min-h-9 max-w-full min-w-0 flex-wrap items-center gap-1 rounded-xl bg-white px-2 py-2 shadow-sm dark:bg-neutral-900">
            {TEXT_COLOR_PRESETS.map((color) => (
              <button
                key={color.value}
                type="button"
                aria-label={t(color.labelKey)}
                onClick={() => handleTextColorApply(color.value)}
                className={`h-5 w-5 rounded-full border transition-transform hover:scale-110 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 ${
                  selectedTextColor === color.value
                    ? 'border-neutral-900 ring-2 ring-neutral-900/10 dark:border-white dark:ring-white/20'
                    : 'border-white dark:border-neutral-700'
                }`}
                style={{ backgroundColor: color.value }}
              />
            ))}
            <div className="ml-1 w-full max-w-[112px] min-w-[96px] flex-1 sm:w-[112px] sm:flex-none">
              <ColorPicker
                value={selectedTextColor}
                format="HEX"
                colorModes={['monochrome']}
                enableAlpha={false}
                recentColors={false}
                swatchColors={TEXT_COLOR_PRESETS.map((color) => color.value)}
                popupProps={{ placement: 'bottom-left' }}
                onChange={(value) => {
                  setSelectedTextColor(normalizeMarkdownColor(value, selectedTextColor));
                }}
              />
            </div>
          </div>

          {allowImageUpload && (
            <button
              type="button"
              aria-label={t('articleEditor.imageTooltip')}
              onClick={onImageUploadClick}
              className="group relative inline-flex h-9 items-center gap-2 rounded-xl px-3 text-xs font-semibold text-primary-600 transition-colors hover:bg-white focus-visible:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:text-primary-400 dark:hover:bg-neutral-800 dark:focus-visible:bg-neutral-800"
            >
              <ImagePlus className="h-4 w-4" aria-hidden="true" />
              {t('articleEditor.imageButton')}
              <span className={tooltipClassName}>{t('articleEditor.imageTooltip')}</span>
            </button>
          )}

          <button
            type="button"
            aria-label={t('articleEditor.aiAssistant')}
            onClick={() => setIsAIPanelOpen((prev) => !prev)}
            className={`group relative inline-flex h-9 items-center gap-2 rounded-xl px-3 text-xs font-semibold transition-colors ${
              isAIPanelOpen || aiSummaryPanel || aiWritingPanel
                ? 'bg-amber-50 text-amber-700 hover:bg-amber-100 dark:bg-amber-500/10 dark:text-amber-200 dark:hover:bg-amber-500/15'
                : 'text-neutral-500 hover:bg-white hover:text-neutral-900 focus-visible:bg-white focus-visible:text-neutral-900 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100 dark:focus-visible:bg-neutral-800 dark:focus-visible:text-neutral-100'
            }`}
          >
            <Sparkles className="h-4 w-4" aria-hidden="true" />
            AI
            <ChevronDown
              className={`h-3.5 w-3.5 transition-transform ${isAIPanelOpen ? 'rotate-180' : ''}`}
              aria-hidden="true"
            />
            <span className={tooltipClassName}>{t('articleEditor.aiAssistant')}</span>
          </button>

          <div className="ml-auto flex flex-wrap items-center gap-3 text-[11px] font-medium text-neutral-400 dark:text-neutral-500">
            <span>{contentStats.characters} {t('articleEditor.characters')}</span>
            <span>{contentStats.lines} {t('articleEditor.lines')}</span>
            <span>{contentStats.words} {t('articleEditor.words')}</span>
            <span>
              ~ {contentStats.readingMinutes} {t('articleEditor.readingMinutes')}
            </span>
          </div>
        </div>

        {isAIPanelOpen && (
          <section className="mb-4 rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-neutral-100 px-4 py-3 dark:border-neutral-800">
              <div>
                <div className="inline-flex items-center gap-2 text-sm font-semibold text-neutral-700 dark:text-neutral-200">
                  <Sparkles className="h-4 w-4 text-amber-500" aria-hidden="true" />
                  {t('articleEditor.aiAssistant')}
                </div>
                <div className="mt-1 text-[11px] text-neutral-400 dark:text-neutral-500">
                  {t('articleEditor.aiWritingResultHint')}
                </div>
              </div>
            </div>
            <div className="space-y-4 p-4">
              <div className="flex flex-wrap items-center gap-2">
                <button
                  type="button"
                  onClick={onGenerateSummary}
                  disabled={Boolean(aiSummaryPanel?.isGenerating)}
                  className="inline-flex items-center gap-1 rounded-full bg-white px-2.5 py-1.5 text-[11px] font-semibold text-amber-700 shadow-sm transition-colors hover:bg-amber-50 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-neutral-900 dark:text-amber-200 dark:hover:bg-neutral-800"
                >
                  {aiSummaryPanel?.isGenerating ? (
                    <>
                      <svg className="h-3 w-3 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                      </svg>
                      {t('articleEditor.summaryGenerating')}
                    </>
                  ) : (
                    <>
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                      </svg>
                      {t('articleEditor.summaryGenerate')}
                    </>
                  )}
                </button>
                {[
                  { action: 'polish' as const, labelKey: 'articleEditor.aiPolish' },
                  { action: 'expand' as const, labelKey: 'articleEditor.aiExpand' },
                  { action: 'shorten' as const, labelKey: 'articleEditor.aiShorten' },
                  { action: 'seo-title' as const, labelKey: 'articleEditor.aiSEOTitle' },
                ].map((item) => {
                  const isActive = aiWritingPanel?.isGenerating && aiWritingPanel.action === item.action;
                  return (
                    <button
                      key={item.action}
                      type="button"
                      onClick={() => onGenerateWritingAction(item.action)}
                      disabled={Boolean(aiWritingPanel?.isGenerating)}
                      className="inline-flex items-center gap-1 rounded-full bg-white px-2.5 py-1.5 text-[11px] font-semibold text-amber-700 shadow-sm transition-colors hover:bg-amber-50 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-neutral-900 dark:text-amber-200 dark:hover:bg-neutral-800"
                    >
                      {isActive && <span className="h-2 w-2 animate-pulse rounded-full bg-amber-500" />}
                      {t(item.labelKey)}
                    </button>
                  );
                })}
              </div>

              {(aiSummaryPanel || aiWritingPanel) && (
                <div className="rounded-xl bg-neutral-50 p-4 shadow-sm dark:bg-neutral-950">
                  {aiSummaryPanel ? (
                    aiSummaryPanel.isGenerating ? (
                      <div className="flex min-h-20 items-center gap-3 rounded-xl bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:bg-amber-500/10 dark:text-amber-200">
                        <span className="h-2.5 w-2.5 animate-pulse rounded-full bg-amber-500" />
                        {t('articleEditor.summaryGenerating')}
                      </div>
                    ) : (
                      <div className="space-y-3">
                        <div className="max-h-72 overflow-y-auto whitespace-pre-wrap rounded-xl bg-white px-3 py-2.5 text-sm leading-7 text-neutral-700 dark:bg-neutral-950 dark:text-neutral-200">
                          {aiSummaryPanel.result}
                        </div>
                        <div className="flex justify-end">
                          <button
                            type="button"
                            onClick={onApplySummary}
                            className="btn btn-primary"
                          >
                            {t('articleEditor.aiWritingApply')}
                          </button>
                        </div>
                      </div>
                    )
                  ) : aiWritingPanel?.isGenerating ? (
                    <div className="flex min-h-20 items-center gap-3 rounded-xl bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:bg-amber-500/10 dark:text-amber-200">
                      <span className="h-2.5 w-2.5 animate-pulse rounded-full bg-amber-500" />
                      {t('articleEditor.aiWritingGenerating')}
                    </div>
                  ) : aiWritingPanel?.action === 'seo-title' && aiWritingPanel.suggestions.length > 0 ? (
                    <div className="space-y-2">
                      {aiWritingPanel.suggestions.map((suggestion) => (
                        <button
                          key={suggestion}
                          type="button"
                          onClick={() => onApplyWritingResult(suggestion)}
                          className="block w-full rounded-xl border border-neutral-100 bg-white px-3 py-2.5 text-left text-sm font-semibold text-neutral-700 transition-colors hover:border-amber-200 hover:bg-amber-50 dark:border-neutral-800 dark:bg-neutral-950 dark:text-neutral-100 dark:hover:border-amber-500/30 dark:hover:bg-amber-500/10"
                        >
                          {suggestion}
                        </button>
                      ))}
                    </div>
                  ) : aiWritingPanel ? (
                    <div className="space-y-3">
                      <div className="max-h-72 overflow-y-auto whitespace-pre-wrap rounded-xl bg-white px-3 py-2.5 text-sm leading-7 text-neutral-700 dark:bg-neutral-950 dark:text-neutral-200">
                        {aiWritingPanel.result}
                      </div>
                      <div className="flex justify-end">
                        <button
                          type="button"
                          onClick={() => onApplyWritingResult(aiWritingPanel.result)}
                          className="btn btn-primary"
                        >
                          {t('articleEditor.aiWritingApply')}
                        </button>
                      </div>
                    </div>
                  ) : null}
                </div>
              )}
            </div>
          </section>
        )}

        <div className={panelGridClassName}>
          {editorMode !== 'preview' && (
            <section
              className={`flex ${panelSizeClass} flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900`}
            >
              <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-3 dark:border-neutral-800">
                <div>
                  <div className="text-sm font-semibold text-neutral-700 dark:text-neutral-200">
                    Markdown
                  </div>
                  <div className="text-[11px] text-neutral-400 dark:text-neutral-500">
                    {t('articleEditor.sourceEdit')}
                  </div>
                </div>
                {lastSavedTime && (
                  <div className="flex items-center gap-2 text-[10px] font-bold uppercase tracking-wider text-neutral-400">
                    <div
                      className={`h-1.5 w-1.5 rounded-full ${
                        isAutoSaving ? 'animate-pulse bg-amber-400' : 'bg-emerald-400'
                      }`}
                      />
                    {isAutoSaving ? t('articleEditor.autoSaving') : t('articleEditor.savedAt', { time: lastSavedTime })}
                  </div>
                )}
              </div>
              <textarea
                ref={textareaRef}
                className={`admin-markdown-editor ${textareaSizeClass} flex-1 resize-none overflow-y-auto border-0 bg-transparent px-5 py-4 text-sm leading-7 text-neutral-800 outline-none dark:text-neutral-100`}
                value={content}
                onChange={(event) => onContentChange(event.target.value)}
                onScroll={handleEditorScroll}
                onPaste={onPaste}
                onSelect={onSelect}
                onKeyUp={onKeyUp}
                onMouseUp={onMouseUp}
                placeholder={resolvedPlaceholder}
              />
            </section>
          )}

          {editorMode !== 'edit' && (
            <section
              className={`flex ${panelSizeClass} flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900`}
            >
              <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-3 dark:border-neutral-800">
                <div>
                  <div className="text-sm font-semibold text-neutral-700 dark:text-neutral-200">
                    {t('articleEditor.previewTitle')}
                  </div>
                  <div className="text-[11px] text-neutral-400 dark:text-neutral-500">
                    {t('articleEditor.previewSubtitle')}
                  </div>
                </div>
              </div>
              <div
                ref={previewScrollRef}
                className={previewBodyClassName}
                onScroll={handlePreviewScroll}
              >
                {content.trim() ? (
                  <Suspense
                    fallback={
                      <div className="h-full animate-pulse rounded-xl bg-neutral-100 dark:bg-neutral-800" />
                    }
                  >
                    <ArticlePreview content={content} />
                  </Suspense>
                ) : (
                  <div className="flex h-full min-h-[420px] items-center justify-center rounded-xl border border-dashed border-neutral-200 text-sm text-neutral-400 dark:border-neutral-800 dark:text-neutral-500">
                    {t('articleEditor.previewEmpty')}
                  </div>
                )}
              </div>
            </section>
          )}
        </div>
      </div>
    </div>
  );
};
