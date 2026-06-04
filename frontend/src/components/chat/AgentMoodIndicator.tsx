import { motion, useReducedMotion } from 'framer-motion';
import { Sparkles } from 'lucide-react';
import { AgentCharacter } from './AgentCharacter';
import { failedTone, toneClasses } from './agentPersonaConfig';
import { resolveAgentMood, type AgentMoodInput } from '@/utils/agentMood';

interface AgentMoodIndicatorProps extends AgentMoodInput {
  showText?: boolean;
  size?: 'md' | 'sm';
}

interface AIProcessingHaloProps extends AgentMoodInput {
  elapsedLabel: string;
  stageLabel?: string | null;
}

const getTone = (status: AgentMoodInput['status'], moodTone: ReturnType<typeof resolveAgentMood>['tone']) => {
  return status === 'failed' ? failedTone : toneClasses[moodTone];
};

export const AgentMoodIndicator = ({
  agentName,
  detail,
  showText = true,
  size = 'md',
  stage,
  status,
  summary,
}: AgentMoodIndicatorProps) => {
  const mood = resolveAgentMood({ agentName, detail, stage, status, summary });
  const isFailed = status === 'failed';
  const tone = getTone(status, mood.tone);
  const prefersReducedMotion = useReducedMotion();

  return (
    <div className="inline-flex min-w-0 items-center gap-3" data-agent-mood={mood.key}>
      <motion.div
        animate={{ scale: prefersReducedMotion ? 1 : [1, 1.04, 1] }}
        transition={{ duration: 2.4, repeat: Infinity, ease: 'easeInOut' }}
      >
        <AgentCharacter isFailed={isFailed} moodKey={mood.key} size={size} tone={tone} />
      </motion.div>

      {showText && (
        <span className="min-w-0">
          <span className="block truncate text-xs font-black text-neutral-800 dark:text-neutral-100">
            {isFailed ? '执行遇到阻碍' : mood.label}
          </span>
          <span className="mt-0.5 block truncate text-[11px] font-medium text-neutral-500 dark:text-neutral-400">
            {isFailed ? '需要检查当前步骤' : mood.caption}
          </span>
        </span>
      )}
    </div>
  );
};

export const AIProcessingHalo = ({
  agentName,
  detail,
  elapsedLabel,
  stage,
  stageLabel,
  status,
  summary,
}: AIProcessingHaloProps) => {
  const mood = resolveAgentMood({ agentName, detail, stage, status, summary });
  const tone = getTone(status, mood.tone);

  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: 6 }}
      className={`relative mb-4 overflow-hidden rounded-2xl border ${tone.ring} bg-white/85 p-3 shadow-soft backdrop-blur dark:bg-[#07111a]/85`}
    >
      <motion.div
        className={`absolute left-0 top-0 h-px w-1/2 bg-gradient-to-r from-transparent via-current to-transparent ${tone.accent}`}
        animate={{ x: ['-100%', '220%'] }}
        transition={{ duration: 1.8, repeat: Infinity, ease: 'easeInOut' }}
      />
      <motion.div
        className={`absolute -inset-x-10 -bottom-8 h-16 bg-gradient-to-r ${tone.gradient} opacity-10 blur-2xl`}
        animate={{ opacity: [0.08, mood.key === 'found' ? 0.28 : 0.16, 0.08], scale: [0.9, 1.06, 0.9] }}
        transition={{ duration: mood.key === 'found' ? 1.3 : 2.2, repeat: Infinity, ease: 'easeInOut' }}
      />
      <div className="relative flex items-center justify-between gap-3">
        <AgentMoodIndicator
          agentName={agentName}
          detail={detail}
          stage={stage}
          status={status}
          summary={summary}
          size="sm"
        />
        <div className={`flex flex-shrink-0 items-center gap-2 rounded-full px-3 py-1 text-[11px] font-black ${tone.badge}`}>
          <Sparkles className="h-3.5 w-3.5" aria-hidden="true" />
          {mood.key === 'found' ? '已命中 · ' : ''}
          {stageLabel || mood.label} · {elapsedLabel}
        </div>
      </div>
    </motion.div>
  );
};
