import { type ClipboardEvent, type RefObject, Suspense, lazy, useState } from 'react';
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
import {
  DEFAULT_TEXT_COLOR,
  applyMarkdownAction,
  applyMarkdownColor,
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
  contentStats: ContentStats;
  lastSavedTime: string | null;
  isAutoSaving: boolean;
  isImmersive: boolean;
  onImmersiveChange: (isImmersive: boolean) => void;
}

const markdownToolbarActions: Array<{
  action: MarkdownAction;
  label: string;
  icon: LucideIcon;
}> = [
  { action: 'heading', label: '二级标题', icon: Heading2 },
  { action: 'bold', label: '加粗', icon: Bold },
  { action: 'quote', label: '引用', icon: Quote },
  { action: 'unordered-list', label: '无序列表', icon: List },
  { action: 'ordered-list', label: '有序列表', icon: ListOrdered },
  { action: 'inline-code', label: '行内代码', icon: Code },
  { action: 'code-block', label: '代码块', icon: Pilcrow },
  { action: 'link', label: '链接', icon: LinkIcon },
  { action: 'divider', label: '分割线', icon: Minus },
];

const TEXT_COLOR_PRESETS = [
  { label: '红色', value: '#ef4444' },
  { label: '橙色', value: '#f97316' },
  { label: '琥珀', value: '#f59e0b' },
  { label: '绿色', value: '#10b981' },
  { label: '天蓝', value: '#0ea5e9' },
  { label: '靛蓝', value: '#6366f1' },
  { label: '粉色', value: '#ec4899' },
  { label: '灰色', value: '#525252' },
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
  contentStats,
  lastSavedTime,
  isAutoSaving,
  isImmersive,
  onImmersiveChange,
}: MarkdownWritingStudioProps) => {
  const [editorMode, setEditorMode] = useState<EditorMode>('split');
  const [selectedTextColor, setSelectedTextColor] = useState(DEFAULT_TEXT_COLOR);

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

  const panelMinHeightClass = isImmersive ? 'min-h-[calc(100vh-220px)]' : 'min-h-[640px]';
  const textareaMinHeightClass = isImmersive ? 'min-h-[calc(100vh-292px)]' : 'min-h-[580px]';

  return (
    <div className="space-y-3">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <label className="block text-sm font-semibold text-neutral-700 dark:text-neutral-200">
            内容 (Markdown)
          </label>
          <p className="mt-1 text-xs text-neutral-400 dark:text-neutral-500">
            支持工具栏插入常用 Markdown，粘贴图片会自动上传。
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
                {mode === 'edit' ? '编辑' : mode === 'split' ? '分屏' : '预览'}
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
            {isImmersive ? '退出专注' : '专注写作'}
          </button>
        </div>
      </div>

      <div className="rounded-2xl border border-neutral-200 bg-neutral-50/70 p-3 dark:border-neutral-800 dark:bg-neutral-950/40">
        <div className="mb-3 flex flex-wrap items-center gap-2">
          {markdownToolbarActions.map((item) => (
            <button
              key={item.action}
              type="button"
              aria-label={item.label}
              onClick={() => handleMarkdownAction(item.action)}
              className="group relative inline-flex h-9 w-9 items-center justify-center rounded-xl text-neutral-500 transition-colors hover:bg-white hover:text-neutral-900 focus-visible:bg-white focus-visible:text-neutral-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100 dark:focus-visible:bg-neutral-800 dark:focus-visible:text-neutral-100"
            >
              <item.icon className="h-4 w-4" aria-hidden="true" />
              <span className={tooltipClassName}>{item.label}</span>
            </button>
          ))}

          <div className="mx-1 h-6 w-px bg-neutral-200 dark:bg-neutral-800" />

          <button
            type="button"
            aria-label="应用当前字体颜色"
            onClick={() => handleTextColorApply()}
            className="group relative inline-flex h-9 w-9 items-center justify-center rounded-xl text-neutral-500 transition-colors hover:bg-white hover:text-neutral-900 focus-visible:bg-white focus-visible:text-neutral-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100 dark:focus-visible:bg-neutral-800 dark:focus-visible:text-neutral-100"
          >
            <Palette className="h-4 w-4" aria-hidden="true" />
            <span
              className="absolute bottom-1 h-0.5 w-5 rounded-full"
              style={{ backgroundColor: selectedTextColor }}
            />
            <span className={tooltipClassName}>字体颜色</span>
          </button>

          <div className="flex h-9 items-center gap-1 rounded-xl bg-white px-2 shadow-sm dark:bg-neutral-900">
            {TEXT_COLOR_PRESETS.map((color) => (
              <button
                key={color.value}
                type="button"
                aria-label={color.label}
                onClick={() => handleTextColorApply(color.value)}
                className={`h-5 w-5 rounded-full border transition-transform hover:scale-110 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 ${
                  selectedTextColor === color.value
                    ? 'border-neutral-900 ring-2 ring-neutral-900/10 dark:border-white dark:ring-white/20'
                    : 'border-white dark:border-neutral-700'
                }`}
                style={{ backgroundColor: color.value }}
              />
            ))}
            <div className="ml-1 w-[112px]">
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

          <button
            type="button"
            aria-label="插入图片"
            onClick={onImageUploadClick}
            className="group relative inline-flex h-9 items-center gap-2 rounded-xl px-3 text-xs font-semibold text-primary-600 transition-colors hover:bg-white focus-visible:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:text-primary-400 dark:hover:bg-neutral-800 dark:focus-visible:bg-neutral-800"
          >
            <ImagePlus className="h-4 w-4" aria-hidden="true" />
            图片
            <span className={tooltipClassName}>插入图片</span>
          </button>

          <div className="ml-auto flex flex-wrap items-center gap-3 text-[11px] font-medium text-neutral-400 dark:text-neutral-500">
            <span>{contentStats.characters} 字符</span>
            <span>{contentStats.lines} 行</span>
            <span>{contentStats.words} 词</span>
            <span>约 {contentStats.readingMinutes} 分钟</span>
          </div>
        </div>

        <div className={`grid gap-4 ${editorMode === 'split' ? 'lg:grid-cols-2' : 'grid-cols-1'}`}>
          {editorMode !== 'preview' && (
            <section
              className={`flex ${panelMinHeightClass} flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900`}
            >
              <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-3 dark:border-neutral-800">
                <div>
                  <div className="text-sm font-semibold text-neutral-700 dark:text-neutral-200">
                    Markdown
                  </div>
                  <div className="text-[11px] text-neutral-400 dark:text-neutral-500">
                    原文编辑
                  </div>
                </div>
                {lastSavedTime && (
                  <div className="flex items-center gap-2 text-[10px] font-bold uppercase tracking-wider text-neutral-400">
                    <div
                      className={`h-1.5 w-1.5 rounded-full ${
                        isAutoSaving ? 'animate-pulse bg-amber-400' : 'bg-emerald-400'
                      }`}
                    />
                    {isAutoSaving ? '正在自动保存' : `已保存 ${lastSavedTime}`}
                  </div>
                )}
              </div>
              <textarea
                ref={textareaRef}
                className={`admin-markdown-editor ${textareaMinHeightClass} flex-1 resize-none border-0 bg-transparent px-5 py-4 text-sm leading-7 text-neutral-800 outline-none dark:text-neutral-100`}
                value={content}
                onChange={(event) => onContentChange(event.target.value)}
                onPaste={onPaste}
                placeholder="使用 Markdown 编写内容..."
              />
            </section>
          )}

          {editorMode !== 'edit' && (
            <section
              className={`flex ${panelMinHeightClass} flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900`}
            >
              <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-3 dark:border-neutral-800">
                <div>
                  <div className="text-sm font-semibold text-neutral-700 dark:text-neutral-200">
                    Preview
                  </div>
                  <div className="text-[11px] text-neutral-400 dark:text-neutral-500">
                    编辑时预览
                  </div>
                </div>
              </div>
              <div className="article-reading-body admin-markdown-preview flex-1 overflow-y-auto px-6 py-5">
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
                    预览会在这里显示
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
