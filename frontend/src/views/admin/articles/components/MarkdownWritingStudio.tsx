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
  Code,
  Eye,
  Heading2,
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
  type LucideIcon,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
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
  onImageUploadClick: () => void;
  allowImageUpload?: boolean;
  helperText?: string;
  placeholder?: string;
  contentStats: ContentStats;
  lastSavedTime: string | null;
  isAutoSaving: boolean;
  isImmersive: boolean;
  onImmersiveChange: (isImmersive: boolean) => void;
}

const markdownToolbarActions: Array<{
  action: MarkdownAction;
  labelKey: string;
  icon: LucideIcon;
}> = [
  { action: 'heading', labelKey: 'articleEditor.toolbarHeading', icon: Heading2 },
  { action: 'bold', labelKey: 'articleEditor.toolbarBold', icon: Bold },
  { action: 'quote', labelKey: 'articleEditor.toolbarQuote', icon: Quote },
  { action: 'unordered-list', labelKey: 'articleEditor.toolbarUnorderedList', icon: List },
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

export const MarkdownWritingStudio = ({
  content,
  onContentChange,
  textareaRef,
  onPaste,
  onImageUploadClick,
  allowImageUpload = true,
  helperText,
  placeholder,
  contentStats,
  lastSavedTime,
  isAutoSaving,
  isImmersive,
  onImmersiveChange,
}: MarkdownWritingStudioProps) => {
  const { t } = useTranslation();
  const [editorMode, setEditorMode] = useState<EditorMode>('split');
  const [selectedTextColor, setSelectedTextColor] = useState(DEFAULT_TEXT_COLOR);
  const previewScrollRef = useRef<HTMLDivElement>(null);
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
        const nextScrollTop = getSynchronizedScrollTop({
          sourceScrollTop: source.scrollTop,
          sourceScrollHeight: source.scrollHeight,
          sourceClientHeight: source.clientHeight,
          targetScrollHeight: target.scrollHeight,
          targetClientHeight: target.clientHeight,
        });

        if (Math.abs(target.scrollTop - nextScrollTop) < 1) return;

        isSyncingScrollRef.current = true;
        target.scrollTop = nextScrollTop;

        requestAnimationFrame(() => {
          isSyncingScrollRef.current = false;
        });
      });
    },
    [editorMode]
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

          <div className="ml-auto flex flex-wrap items-center gap-3 text-[11px] font-medium text-neutral-400 dark:text-neutral-500">
            <span>{contentStats.characters} {t('articleEditor.characters')}</span>
            <span>{contentStats.lines} {t('articleEditor.lines')}</span>
            <span>{contentStats.words} {t('articleEditor.words')}</span>
            <span>
              ~ {contentStats.readingMinutes} {t('articleEditor.readingMinutes')}
            </span>
          </div>
        </div>

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
