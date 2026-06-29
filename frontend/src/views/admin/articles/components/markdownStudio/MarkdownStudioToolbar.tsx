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
  Sparkles,
  SplitSquareHorizontal,
  type LucideIcon,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  normalizeMarkdownColor,
  type MarkdownAction,
} from '@/utils/markdownEditor';
import type { ContentStats, EditorMode } from './types';

interface MarkdownStudioToolbarProps {
  editorMode: EditorMode;
  onModeChange: (mode: EditorMode) => void;
  isImmersive: boolean;
  onImmersiveChange: (isImmersive: boolean) => void;
  selectedTextColor: string;
  onSelectedTextColorChange: (color: string) => void;
  onTextColorApply: (color?: string) => void;
  onMarkdownAction: (action: MarkdownAction) => void;
  onImageUploadClick: () => void;
  allowImageUpload: boolean;
  isAIPanelOpen: boolean;
  hasAIActivity: boolean;
  onToggleAIPanel: () => void;
  contentStats: ContentStats;
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

const tooltipClassName =
  'pointer-events-none absolute left-1/2 top-full z-30 mt-2 -translate-x-1/2 whitespace-nowrap rounded-md border border-neutral-200 bg-white px-2 py-1 text-[11px] font-medium text-neutral-600 opacity-0 shadow-sm transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200';

const modeClassName = (isActive: boolean) =>
  `inline-flex h-9 items-center gap-2 rounded-xl px-3 text-xs font-semibold transition-colors ${
    isActive
      ? 'bg-neutral-900 text-white dark:bg-neutral-100 dark:text-neutral-900'
      : 'bg-neutral-100 text-neutral-500 hover:text-neutral-900 dark:bg-neutral-800 dark:text-neutral-400 dark:hover:text-neutral-100'
  }`;

export const MarkdownStudioToolbar = ({
  editorMode,
  onModeChange,
  isImmersive,
  onImmersiveChange,
  selectedTextColor,
  onSelectedTextColorChange,
  onTextColorApply,
  onMarkdownAction,
  onImageUploadClick,
  allowImageUpload,
  isAIPanelOpen,
  hasAIActivity,
  onToggleAIPanel,
  contentStats,
}: MarkdownStudioToolbarProps) => {
  const { t } = useTranslation();

  return (
    <>
      <div className="flex flex-wrap items-center gap-2">
        {(['edit', 'split', 'preview'] as const).map((mode) => {
          const Icon =
            mode === 'edit' ? PanelLeft : mode === 'split' ? SplitSquareHorizontal : Eye;
          return (
            <button
              key={mode}
              type="button"
              onClick={() => onModeChange(mode)}
              className={modeClassName(editorMode === mode)}
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
      </div>

      <div className="flex flex-wrap items-center gap-2">
        {markdownToolbarActions.map((item) => (
          <button
            key={item.action}
            type="button"
            aria-label={t(item.labelKey)}
            onClick={() => onMarkdownAction(item.action)}
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
          onClick={() => onTextColorApply()}
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
              onClick={() => onTextColorApply(color.value)}
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
                onSelectedTextColorChange(normalizeMarkdownColor(value, selectedTextColor));
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
          onClick={onToggleAIPanel}
          className={`group relative inline-flex h-9 items-center gap-2 rounded-xl px-3 text-xs font-semibold transition-colors ${
            isAIPanelOpen || hasAIActivity
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
          {t(isImmersive ? 'articleEditor.focusExit' : 'articleEditor.focusEnter')}
        </button>

        <div className="flex w-full flex-wrap items-center gap-3 text-[11px] font-medium text-neutral-400 dark:text-neutral-500 sm:ml-auto sm:w-auto">
          <span>
            {contentStats.characters} {t('articleEditor.characters')}
          </span>
          <span>
            {contentStats.lines} {t('articleEditor.lines')}
          </span>
          <span>
            {contentStats.words} {t('articleEditor.words')}
          </span>
          <span>
            ~ {contentStats.readingMinutes} {t('articleEditor.readingMinutes')}
          </span>
        </div>
      </div>
    </>
  );
};
