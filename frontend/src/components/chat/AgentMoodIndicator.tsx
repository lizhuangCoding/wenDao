import { motion } from 'framer-motion';
import {
  Blend,
  BookOpen,
  CircleHelp,
  Cog,
  Glasses,
  GitMerge,
  Lightbulb,
  Newspaper,
  Orbit,
  RefreshCcw,
  Route,
  SearchCheck,
  Smile,
  Sparkles,
  type LucideIcon,
} from 'lucide-react';
import { resolveAgentMood, type AgentMoodInput, type AgentMoodKey } from '@/utils/agentMood';

interface AgentMoodIndicatorProps extends AgentMoodInput {
  showText?: boolean;
  size?: 'md' | 'sm';
}

interface AIProcessingHaloProps extends AgentMoodInput {
  elapsedLabel: string;
  stageLabel?: string | null;
}

const iconByMood: Record<AgentMoodKey, LucideIcon> = {
  clarifier: CircleHelp,
  executor: Cog,
  found: Lightbulb,
  journalist: Newspaper,
  librarian: BookOpen,
  planner: Route,
  replanner: RefreshCcw,
  reviewer: SearchCheck,
  synthesizer: GitMerge,
  thinking: Orbit,
};

const toneClasses = {
  amber: {
    accent: 'text-amber-600 dark:text-amber-300',
    gradient: 'from-amber-400 to-orange-500',
  },
  blue: {
    accent: 'text-blue-600 dark:text-blue-300',
    gradient: 'from-blue-400 to-sky-500',
  },
  cyan: {
    accent: 'text-cyan-600 dark:text-cyan-300',
    gradient: 'from-cyan-400 to-teal-500',
  },
  emerald: {
    accent: 'text-primary-600 dark:text-primary-300',
    gradient: 'from-primary-400 to-emerald-600',
  },
  fuchsia: {
    accent: 'text-fuchsia-600 dark:text-fuchsia-300',
    gradient: 'from-fuchsia-400 to-pink-500',
  },
  indigo: {
    accent: 'text-indigo-600 dark:text-indigo-300',
    gradient: 'from-indigo-400 to-violet-500',
  },
  rose: {
    accent: 'text-rose-600 dark:text-rose-300',
    gradient: 'from-rose-400 to-red-500',
  },
  violet: {
    accent: 'text-violet-600 dark:text-violet-300',
    gradient: 'from-violet-400 to-purple-500',
  },
};

const failedTone = {
  accent: 'text-red-600 dark:text-red-300',
  gradient: 'from-red-400 to-rose-600',
};

const sizeClasses = {
  md: {
    icon: 'h-5 w-5',
    orbit: 'h-11 w-11',
    shell: 'h-9 w-9',
  },
  sm: {
    icon: 'h-4 w-4',
    orbit: 'h-9 w-9',
    shell: 'h-7 w-7',
  },
};

const renderAccessory = (moodKey: AgentMoodKey) => {
  if (moodKey === 'librarian') {
    return <Glasses className="absolute -top-1 h-4 w-4 text-current" aria-hidden="true" />;
  }

  if (moodKey === 'found') {
    return <Smile className="absolute -right-1 -top-1 h-4 w-4 text-current" aria-hidden="true" />;
  }

  if (moodKey === 'synthesizer') {
    return <Blend className="absolute -bottom-1 -right-1 h-3.5 w-3.5 text-current" aria-hidden="true" />;
  }

  return null;
};

export const AgentMoodIndicator = ({
  agentName,
  showText = true,
  size = 'md',
  stage,
  status,
  summary,
}: AgentMoodIndicatorProps) => {
  const mood = resolveAgentMood({ agentName, stage, status, summary });
  const Icon = iconByMood[mood.key];
  const tone = status === 'failed' ? failedTone : toneClasses[mood.tone];
  const dimensions = sizeClasses[size];

  return (
    <div className="inline-flex min-w-0 items-center gap-3" data-agent-mood={mood.key}>
      <div className={`relative flex ${dimensions.orbit} flex-shrink-0 items-center justify-center ${tone.accent}`}>
        <motion.span
          className="absolute inset-0 rounded-full bg-current opacity-15 blur-md"
          animate={{ opacity: [0.12, 0.28, 0.12], scale: [0.9, 1.18, 0.9] }}
          transition={{ duration: 1.8, repeat: Infinity, ease: 'easeInOut' }}
        />
        <svg className="absolute inset-0 h-full w-full" viewBox="0 0 44 44" aria-hidden="true">
          <motion.ellipse
            cx="22"
            cy="22"
            fill="none"
            rx="17"
            ry="6.5"
            stroke="currentColor"
            strokeLinecap="round"
            strokeWidth="1.4"
            animate={{ pathLength: [0.25, 0.92, 0.25], rotate: 360 }}
            style={{ originX: '22px', originY: '22px' }}
            transition={{ duration: 2.4, repeat: Infinity, ease: 'linear' }}
          />
          <motion.ellipse
            cx="22"
            cy="22"
            fill="none"
            rx="17"
            ry="6.5"
            stroke="currentColor"
            strokeLinecap="round"
            strokeWidth="1.1"
            transform="rotate(62 22 22)"
            animate={{ pathLength: [0.72, 0.28, 0.72] }}
            transition={{ duration: 2, repeat: Infinity, ease: 'easeInOut' }}
          />
        </svg>
        <span className={`relative flex ${dimensions.shell} items-center justify-center rounded-full bg-gradient-to-br ${tone.gradient} text-white shadow-soft`}>
          <Icon className={dimensions.icon} aria-hidden="true" />
          {renderAccessory(mood.key)}
        </span>
      </div>

      {showText && (
        <span className="min-w-0">
          <span className="block truncate text-xs font-black text-neutral-800 dark:text-neutral-100">
            {status === 'failed' ? '执行遇到阻碍' : mood.label}
          </span>
          <span className="mt-0.5 block truncate text-[11px] font-medium text-neutral-500 dark:text-neutral-400">
            {status === 'failed' ? '需要检查当前步骤' : mood.caption}
          </span>
        </span>
      )}
    </div>
  );
};

export const AIProcessingHalo = ({
  agentName,
  elapsedLabel,
  stage,
  stageLabel,
  status,
  summary,
}: AIProcessingHaloProps) => {
  const mood = resolveAgentMood({ agentName, stage, status, summary });

  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: 6 }}
      className="relative mb-4 overflow-hidden rounded-2xl border border-primary-100 bg-white/85 p-3 shadow-soft backdrop-blur dark:border-primary-900/40 dark:bg-neutral-900/85"
    >
      <motion.div
        className="absolute left-0 top-0 h-px w-1/2 bg-gradient-to-r from-transparent via-primary-400 to-transparent"
        animate={{ x: ['-100%', '220%'] }}
        transition={{ duration: 1.8, repeat: Infinity, ease: 'easeInOut' }}
      />
      <motion.div
        className="absolute -inset-x-10 -bottom-8 h-16 bg-primary-400/10 blur-2xl"
        animate={{ opacity: [0.2, 0.55, 0.2], scale: [0.9, 1.05, 0.9] }}
        transition={{ duration: 2.2, repeat: Infinity, ease: 'easeInOut' }}
      />
      <div className="relative flex items-center justify-between gap-3">
        <AgentMoodIndicator
          agentName={agentName}
          stage={stage}
          status={status}
          summary={summary}
          size="sm"
        />
        <div className="flex flex-shrink-0 items-center gap-2 rounded-full bg-primary-50 px-3 py-1 text-[11px] font-black text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">
          <Sparkles className="h-3.5 w-3.5" aria-hidden="true" />
          {stageLabel || mood.label} · {elapsedLabel}
        </div>
      </div>
    </motion.div>
  );
};
