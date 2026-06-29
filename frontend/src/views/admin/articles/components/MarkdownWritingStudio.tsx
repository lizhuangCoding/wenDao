import {
  type ClipboardEvent,
  type RefObject,
  Suspense,
  lazy,
  useRef,
  useState,
} from 'react';
import { useTranslation } from 'react-i18next';
import {
  DEFAULT_TEXT_COLOR,
  applyMarkdownAction,
  applyMarkdownColor,
  normalizeMarkdownColor,
  type ApplyMarkdownActionResult,
  type MarkdownAction,
} from '@/utils/markdownEditor';
import { MarkdownStudioAIPanel } from './markdownStudio/MarkdownStudioAIPanel';
import { MarkdownStudioToolbar } from './markdownStudio/MarkdownStudioToolbar';
import { useMarkdownStudioScrollSync } from './markdownStudio/useMarkdownStudioScrollSync';
import type {
  ContentStats,
  EditorMode,
  SummaryPanelState,
  WritingPanelState,
} from './markdownStudio/types';

const ArticlePreview = lazy(() =>
  import('../ArticlePreview').then((module) => ({ default: module.ArticlePreview }))
);

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
  aiSummaryPanel?: SummaryPanelState | null;
  aiWritingPanel?: WritingPanelState | null;
  onGenerateSummary?: () => void;
  onApplySummary?: () => void;
  onGenerateWritingAction?: (action: WritingPanelState['action']) => void;
  onApplyWritingResult?: (result: string) => void;
}

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
  const { editorScrollMirrorRef, handleEditorScroll, handlePreviewScroll } =
    useMarkdownStudioScrollSync({
      content,
      editorMode,
      textareaRef,
      previewScrollRef,
    });

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
    ? 'mb-3 flex shrink-0 flex-col gap-3 rounded-2xl border border-neutral-200 bg-white px-4 py-3 shadow-sm dark:border-neutral-700 dark:bg-neutral-900 lg:flex-row lg:items-center lg:justify-between'
    : 'flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between';
  const studioShellClassName = isImmersive
    ? 'flex min-h-0 flex-1 flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-neutral-50/90 p-3 dark:border-neutral-700 dark:bg-neutral-950/90'
    : 'rounded-2xl border border-neutral-200 bg-neutral-50/70 p-3 dark:border-neutral-700 dark:bg-neutral-950/40';
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
      </div>

      <div className={studioShellClassName}>
        <div className={toolbarClassName}>
          <MarkdownStudioToolbar
            editorMode={editorMode}
            onModeChange={setEditorMode}
            isImmersive={isImmersive}
            onImmersiveChange={onImmersiveChange}
            selectedTextColor={selectedTextColor}
            onSelectedTextColorChange={setSelectedTextColor}
            onTextColorApply={handleTextColorApply}
            onMarkdownAction={handleMarkdownAction}
            onImageUploadClick={onImageUploadClick}
            allowImageUpload={allowImageUpload}
            isAIPanelOpen={isAIPanelOpen}
            hasAIActivity={Boolean(aiSummaryPanel || aiWritingPanel)}
            onToggleAIPanel={() => setIsAIPanelOpen((prev) => !prev)}
            contentStats={contentStats}
          />
        </div>

        {isAIPanelOpen && (
          <MarkdownStudioAIPanel
            aiSummaryPanel={aiSummaryPanel}
            aiWritingPanel={aiWritingPanel}
            onGenerateSummary={onGenerateSummary}
            onApplySummary={onApplySummary}
            onGenerateWritingAction={onGenerateWritingAction}
            onApplyWritingResult={onApplyWritingResult}
          />
        )}

        <div className={panelGridClassName}>
          {editorMode !== 'preview' && (
            <section
              className={`flex ${panelSizeClass} flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-700 dark:bg-neutral-900`}
            >
              <div className="flex items-center justify-between border-b border-neutral-200 px-4 py-3 dark:border-neutral-700">
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
              className={`flex ${panelSizeClass} flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-700 dark:bg-neutral-900`}
            >
              <div className="flex items-center justify-between border-b border-neutral-200 px-4 py-3 dark:border-neutral-700">
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
                  <div className="flex h-full min-h-[420px] items-center justify-center rounded-xl border border-dashed border-neutral-200 text-sm text-neutral-400 dark:border-neutral-700 dark:text-neutral-500">
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
