import { useTranslation } from 'react-i18next';
import { AgentMoodIndicator } from './AgentMoodIndicator';
import type { ChatStage, ChatStep } from '@/types';

interface ChatStageBannerProps {
  currentStage: ChatStage | null;
  featuredAgentStep: ChatStep | null;
  isAssistantProcessing: boolean;
  label?: string | null;
  pendingQuestion?: string | null;
  processingDurationLabel: string;
  requiresUserInput: boolean;
}

export const ChatStageBanner = ({
  currentStage,
  featuredAgentStep,
  isAssistantProcessing,
  label,
  pendingQuestion,
  processingDurationLabel,
  requiresUserInput,
}: ChatStageBannerProps) => {
  const { t } = useTranslation();
  if (!label) return null;

  return (
    <div className="mb-4 rounded-xl border border-primary-200 bg-primary-50 px-4 py-3 text-sm text-primary-700 dark:border-primary-800 dark:bg-primary-900/20 dark:text-primary-300">
      <div className="flex flex-wrap items-center gap-2 justify-between">
        <span className="inline-flex items-center gap-2">
          {isAssistantProcessing && (
            <AgentMoodIndicator
              agentName={featuredAgentStep?.agent_name}
              detail={featuredAgentStep?.detail}
              showText={false}
              size="sm"
              stage={currentStage}
              status={featuredAgentStep?.status}
              summary={featuredAgentStep?.summary}
            />
          )}
          {label}
          {requiresUserInput && pendingQuestion ? `：${pendingQuestion}` : ''}
        </span>
        {isAssistantProcessing && (
          <span className="inline-flex items-center rounded-full bg-white/80 dark:bg-primary-950/40 px-2.5 py-1 text-[11px] font-bold text-primary-700 dark:text-primary-200">
            {t('chat.elapsed', { duration: processingDurationLabel })}
          </span>
        )}
      </div>
    </div>
  );
};
