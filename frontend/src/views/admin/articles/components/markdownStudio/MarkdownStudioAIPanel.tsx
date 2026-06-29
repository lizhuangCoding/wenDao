import { Sparkles } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { AIWritingAction } from '@/api/chat';
import type { SummaryPanelState, WritingPanelState } from './types';

interface MarkdownStudioAIPanelProps {
  aiSummaryPanel: SummaryPanelState | null;
  aiWritingPanel: WritingPanelState | null;
  onGenerateSummary: () => void;
  onApplySummary: () => void;
  onGenerateWritingAction: (action: AIWritingAction) => void;
  onApplyWritingResult: (result: string) => void;
}

const writingActions: Array<{ action: AIWritingAction; labelKey: string }> = [
  { action: 'polish', labelKey: 'articleEditor.aiPolish' },
  { action: 'expand', labelKey: 'articleEditor.aiExpand' },
  { action: 'shorten', labelKey: 'articleEditor.aiShorten' },
  { action: 'seo-title', labelKey: 'articleEditor.aiSEOTitle' },
];

export const MarkdownStudioAIPanel = ({
  aiSummaryPanel,
  aiWritingPanel,
  onGenerateSummary,
  onApplySummary,
  onGenerateWritingAction,
  onApplyWritingResult,
}: MarkdownStudioAIPanelProps) => {
  const { t } = useTranslation();

  return (
    <section className="mb-4 rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-neutral-200 px-4 py-3 dark:border-neutral-700">
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
                <svg
                  className="h-3 w-3 animate-spin"
                  xmlns="http://www.w3.org/2000/svg"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle
                    className="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    strokeWidth="4"
                  />
                  <path
                    className="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  />
                </svg>
                {t('articleEditor.summaryGenerating')}
              </>
            ) : (
              <>
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  className="h-3.5 w-3.5"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M13 10V3L4 14h7v7l9-11h-7z"
                  />
                </svg>
                {t('articleEditor.summaryGenerate')}
              </>
            )}
          </button>
          {writingActions.map((item) => {
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
                    <button type="button" onClick={onApplySummary} className="btn btn-primary">
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
                    className="block w-full rounded-xl border border-neutral-200 bg-white px-3 py-2.5 text-left text-sm font-semibold text-neutral-700 transition-colors hover:border-amber-200 hover:bg-amber-50 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100 dark:hover:border-amber-500/30 dark:hover:bg-amber-500/10"
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
  );
};
