import { useState, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Cpu, Check } from 'lucide-react';
import { chatApi } from '@/api';
import type { ModelInfo } from '@/types';

interface ModelSelectorProps {
  selectedModel: { provider: string; model_name: string } | null;
  onSelect: (model: { provider: string; model_name: string } | null) => void;
}

export const ModelSelector = ({ selectedModel, onSelect }: ModelSelectorProps) => {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [models, setModels] = useState<ModelInfo[]>([]);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    chatApi.getModels().then((res) => {
      if (res.models) setModels(res.models);
    }).catch(() => {
      // keep empty list
    });
  }, []);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const currentLabel = selectedModel
    ? models.find((m) => m.provider === selectedModel.provider && m.model_name === selectedModel.model_name)?.display_name || `${selectedModel.provider} / ${selectedModel.model_name}`
    : null;

  if (models.length === 0) return null;

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={`inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-xs font-bold transition-colors ${
          currentLabel
            ? 'border-primary-200 dark:border-primary-800 bg-primary-50 dark:bg-primary-900/20 text-primary-700 dark:text-primary-300'
            : 'border-neutral-200 dark:border-neutral-700 bg-neutral-50 dark:bg-neutral-800 text-neutral-500 dark:text-neutral-400 hover:text-primary-600 dark:hover:text-primary-400 hover:border-primary-200 dark:hover:border-primary-700'
        }`}
        aria-label={t('chat.selectModel')}
        title={currentLabel || t('common.defaultModel')}
      >
        <Cpu className="h-3.5 w-3.5" aria-hidden="true" />
        <span className="max-w-[min(60vw,260px)] truncate">{currentLabel || t('common.defaultModel')}</span>
      </button>

      {open && (
        <div className="absolute bottom-full right-0 mb-2 w-[min(92vw,28rem)] rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 shadow-elevated z-50 py-1 overflow-hidden animate-in fade-in slide-in-from-bottom-1 duration-150">
          <button
            type="button"
            onClick={() => { onSelect(null); setOpen(false); }}
            className="w-full flex items-center gap-2 px-3 py-2.5 text-sm text-neutral-700 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-700 transition-colors"
          >
            <span className="flex-1 text-left">{t('chat.modelSelectorDefault')}</span>
            {!selectedModel && <Check className="h-4 w-4 text-primary-600 dark:text-primary-400" />}
          </button>
          <div className="mx-3 h-px bg-neutral-100 dark:bg-neutral-700 my-1" />
          {models.map((m) => {
            const isSelected = selectedModel?.provider === m.provider && selectedModel?.model_name === m.model_name;
            return (
              <button
                key={`${m.provider}/${m.model_name}`}
                type="button"
                onClick={() => { onSelect({ provider: m.provider, model_name: m.model_name }); setOpen(false); }}
                className="w-full flex items-center gap-2 px-3 py-2.5 text-sm text-neutral-700 dark:text-neutral-300 hover:bg-neutral-50 dark:hover:bg-neutral-700 transition-colors"
                title={m.display_name}
              >
                <span className="min-w-0 flex-1 whitespace-normal break-words text-left leading-5">{m.display_name}</span>
                {isSelected && <Check className="h-4 w-4 text-primary-600 dark:text-primary-400" />}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
};
