import { motion } from 'framer-motion';
import { Sparkles } from 'lucide-react';
import { resolveAgentMood, type AgentMoodInput, type AgentMoodKey } from '@/utils/agentMood';

interface AgentMoodIndicatorProps extends AgentMoodInput {
  showText?: boolean;
  size?: 'md' | 'sm';
}

interface AIProcessingHaloProps extends AgentMoodInput {
  elapsedLabel: string;
  stageLabel?: string | null;
}

type ToneKey = ReturnType<typeof resolveAgentMood>['tone'];

const toneClasses: Record<ToneKey, {
  accent: string;
  badge: string;
  gradient: string;
  ring: string;
}> = {
  amber: {
    accent: 'text-amber-600 dark:text-amber-300',
    badge: 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-200',
    gradient: 'from-amber-300 via-yellow-400 to-orange-500',
    ring: 'border-amber-200 dark:border-amber-800/60',
  },
  blue: {
    accent: 'text-blue-600 dark:text-blue-300',
    badge: 'bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-200',
    gradient: 'from-blue-300 via-sky-400 to-cyan-500',
    ring: 'border-blue-200 dark:border-blue-800/60',
  },
  cyan: {
    accent: 'text-cyan-600 dark:text-cyan-300',
    badge: 'bg-cyan-50 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-200',
    gradient: 'from-cyan-300 via-teal-400 to-emerald-500',
    ring: 'border-cyan-200 dark:border-cyan-800/60',
  },
  emerald: {
    accent: 'text-primary-600 dark:text-primary-300',
    badge: 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200',
    gradient: 'from-primary-300 via-emerald-400 to-teal-500',
    ring: 'border-primary-200 dark:border-primary-800/60',
  },
  fuchsia: {
    accent: 'text-fuchsia-600 dark:text-fuchsia-300',
    badge: 'bg-fuchsia-50 text-fuchsia-700 dark:bg-fuchsia-900/30 dark:text-fuchsia-200',
    gradient: 'from-fuchsia-300 via-pink-400 to-rose-500',
    ring: 'border-fuchsia-200 dark:border-fuchsia-800/60',
  },
  indigo: {
    accent: 'text-indigo-600 dark:text-indigo-300',
    badge: 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-200',
    gradient: 'from-indigo-300 via-violet-400 to-purple-500',
    ring: 'border-indigo-200 dark:border-indigo-800/60',
  },
  rose: {
    accent: 'text-rose-600 dark:text-rose-300',
    badge: 'bg-rose-50 text-rose-700 dark:bg-rose-900/30 dark:text-rose-200',
    gradient: 'from-rose-300 via-pink-400 to-red-500',
    ring: 'border-rose-200 dark:border-rose-800/60',
  },
  violet: {
    accent: 'text-violet-600 dark:text-violet-300',
    badge: 'bg-violet-50 text-violet-700 dark:bg-violet-900/30 dark:text-violet-200',
    gradient: 'from-violet-300 via-purple-400 to-fuchsia-500',
    ring: 'border-violet-200 dark:border-violet-800/60',
  },
};

const failedTone = {
  accent: 'text-red-600 dark:text-red-300',
  badge: 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-200',
  gradient: 'from-red-300 via-rose-400 to-red-600',
  ring: 'border-red-200 dark:border-red-800/60',
};

const sizeClasses = {
  md: {
    frame: 'h-14 w-14',
    svg: 'h-14 w-14',
  },
  sm: {
    frame: 'h-11 w-11',
    svg: 'h-11 w-11',
  },
};

const characterColors: Record<AgentMoodKey, { coat: string; trim: string; prop: string }> = {
  clarifier: { coat: '#f59e0b', trim: '#fef3c7', prop: '#2563eb' },
  executor: { coat: '#4f46e5', trim: '#c7d2fe', prop: '#94a3b8' },
  found: { coat: '#f59e0b', trim: '#fef3c7', prop: '#facc15' },
  journalist: { coat: '#0891b2', trim: '#cffafe', prop: '#0f172a' },
  librarian: { coat: '#059669', trim: '#d1fae5', prop: '#7c3aed' },
  planner: { coat: '#2563eb', trim: '#dbeafe', prop: '#22c55e' },
  replanner: { coat: '#c026d3', trim: '#fae8ff', prop: '#ec4899' },
  reviewer: { coat: '#e11d48', trim: '#ffe4e6', prop: '#0f172a' },
  synthesizer: { coat: '#7c3aed', trim: '#ede9fe', prop: '#06b6d4' },
  thinking: { coat: '#0f766e', trim: '#ccfbf1', prop: '#14b8a6' },
};

const renderMouth = (moodKey: AgentMoodKey, isFailed: boolean) => {
  if (isFailed) {
    return <path d="M27 30 Q32 27 37 30" fill="none" stroke="#7c2d12" strokeLinecap="round" strokeWidth="1.7" />;
  }

  if (moodKey === 'found') {
    return <path d="M27 29 Q32 34 37 29" fill="none" stroke="#7c2d12" strokeLinecap="round" strokeWidth="1.8" />;
  }

  return <path d="M28 29 Q32 31.5 36 29" fill="none" stroke="#7c2d12" strokeLinecap="round" strokeWidth="1.5" />;
};

const renderAccessory = (moodKey: AgentMoodKey, propColor: string) => {
  switch (moodKey) {
    case 'librarian':
      return (
        <>
          <g fill="none" stroke="#334155" strokeWidth="1.6">
            <circle cx="27.5" cy="22" r="3.6" />
            <circle cx="36.5" cy="22" r="3.6" />
            <path d="M31.1 22H32.9" />
          </g>
          <motion.g
            animate={{ rotate: [-3, 3, -3], y: [0, -0.8, 0] }}
            style={{ transformOrigin: '31px 45px' }}
            transition={{ duration: 1.8, repeat: Infinity, ease: 'easeInOut' }}
          >
            <rect x="14" y="39" width="17" height="15" rx="2.5" fill={propColor} />
            <rect x="31" y="39" width="17" height="15" rx="2.5" fill="#8b5cf6" />
            <path d="M31 39V54" stroke="#ede9fe" strokeWidth="1.2" />
            <motion.path
              d="M19 43H27M19 47H26M36 43H43M36 47H42"
              stroke="#ede9fe"
              strokeLinecap="round"
              strokeWidth="1"
              animate={{ opacity: [0.5, 1, 0.5] }}
              transition={{ duration: 1.5, repeat: Infinity, ease: 'easeInOut' }}
            />
          </motion.g>
        </>
      );
    case 'journalist':
      return (
        <motion.g
          animate={{ x: [0, 1.2, 0] }}
          transition={{ duration: 1.6, repeat: Infinity, ease: 'easeInOut' }}
        >
          <rect x="39" y="35" width="15" height="14" rx="2.2" fill="#f8fafc" stroke="#94a3b8" strokeWidth="1.1" />
          <path d="M42 39H51M42 42H49M42 45H50" stroke="#0891b2" strokeLinecap="round" strokeWidth="1.1" />
          <rect x="18" y="37" width="9" height="6" rx="2" fill="#f8fafc" stroke="#94a3b8" strokeWidth="1" />
        </motion.g>
      );
    case 'synthesizer':
      return (
        <g>
          <motion.path
            d="M18 40 C24 33, 40 33, 46 40"
            fill="none"
            stroke="#06b6d4"
            strokeLinecap="round"
            strokeWidth="1.8"
            animate={{ pathLength: [0.35, 1, 0.35] }}
            transition={{ duration: 2, repeat: Infinity, ease: 'easeInOut' }}
          />
          <circle cx="18" cy="40" r="3" fill="#67e8f9" />
          <circle cx="46" cy="40" r="3" fill="#c084fc" />
          <motion.circle
            cx="32"
            cy="34"
            r="3.4"
            fill={propColor}
            animate={{ scale: [0.85, 1.15, 0.85] }}
            transition={{ duration: 1.4, repeat: Infinity, ease: 'easeInOut' }}
          />
        </g>
      );
    case 'reviewer':
      return (
        <g>
          <motion.g
            animate={{ rotate: [-8, 5, -8] }}
            style={{ transformOrigin: '44px 39px' }}
            transition={{ duration: 1.7, repeat: Infinity, ease: 'easeInOut' }}
          >
            <circle cx="43" cy="38" r="5.8" fill="#ffffff" fillOpacity="0.75" stroke={propColor} strokeWidth="2" />
            <path d="M47.5 42.5L53 48" stroke={propColor} strokeLinecap="round" strokeWidth="2.6" />
          </motion.g>
          <path d="M17 42L21 46L28 38" fill="none" stroke="#22c55e" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.4" />
        </g>
      );
    case 'planner':
      return (
        <motion.g
          animate={{ y: [0, -1, 0] }}
          transition={{ duration: 1.7, repeat: Infinity, ease: 'easeInOut' }}
        >
          <rect x="13" y="38" width="38" height="14" rx="3" fill="#f8fafc" stroke="#93c5fd" strokeWidth="1.2" />
          <path d="M20 47 C25 38, 33 51, 43 42" fill="none" stroke={propColor} strokeLinecap="round" strokeWidth="2" />
          <circle cx="20" cy="47" r="2" fill="#2563eb" />
          <circle cx="43" cy="42" r="2" fill="#16a34a" />
        </motion.g>
      );
    case 'executor':
      return (
        <motion.g
          animate={{ rotate: [-8, 10, -8] }}
          style={{ transformOrigin: '45px 42px' }}
          transition={{ duration: 1.2, repeat: Infinity, ease: 'easeInOut' }}
        >
          <path d="M41 36L47 42M46 35L51 40L44 47L39 42Z" fill="#e2e8f0" stroke="#475569" strokeLinejoin="round" strokeWidth="1.4" />
          <circle cx="22" cy="43" r="4" fill="none" stroke={propColor} strokeWidth="2" />
          <path d="M22 35V39M22 47V51M14 43H18M26 43H30" stroke={propColor} strokeLinecap="round" strokeWidth="1.5" />
        </motion.g>
      );
    case 'replanner':
      return (
        <motion.g
          animate={{ rotate: 360 }}
          style={{ transformOrigin: '44px 41px' }}
          transition={{ duration: 2.2, repeat: Infinity, ease: 'linear' }}
        >
          <path d="M48 36A7 7 0 1 0 50 43" fill="none" stroke={propColor} strokeLinecap="round" strokeWidth="2" />
          <path d="M48 36H53V31" fill="none" stroke={propColor} strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" />
        </motion.g>
      );
    case 'clarifier':
      return (
        <motion.g
          animate={{ y: [0, -2, 0] }}
          transition={{ duration: 1.5, repeat: Infinity, ease: 'easeInOut' }}
        >
          <path d="M40 8H55C57 8 58 9 58 11V20C58 22 57 23 55 23H49L45 27V23H40C38 23 37 22 37 20V11C37 9 38 8 40 8Z" fill="#fff7ed" stroke="#f59e0b" strokeWidth="1.3" />
          <text x="47.5" y="19" textAnchor="middle" fontSize="10" fontWeight="800" fill="#d97706">?</text>
        </motion.g>
      );
    case 'found':
      return (
        <>
          <motion.g
            animate={{ y: [-1, -4, -1], scale: [1, 1.08, 1] }}
            style={{ transformOrigin: '32px 8px' }}
            transition={{ duration: 1.3, repeat: Infinity, ease: 'easeInOut' }}
          >
            <path d="M32 4C27.8 4 24.6 7.2 24.6 11.2C24.6 13.9 26 15.5 27.6 17.2C28.4 18 28.9 19 29 20.2H35C35.1 19 35.6 18 36.4 17.2C38 15.5 39.4 13.9 39.4 11.2C39.4 7.2 36.2 4 32 4Z" fill={propColor} stroke="#f59e0b" strokeWidth="1.2" />
            <path d="M29 22H35M30 25H34" stroke="#92400e" strokeLinecap="round" strokeWidth="1.4" />
          </motion.g>
          <motion.g
            animate={{ opacity: [0.2, 1, 0.2], scale: [0.8, 1.15, 0.8] }}
            transition={{ duration: 1.4, repeat: Infinity, ease: 'easeInOut' }}
          >
            <path d="M14 15L16 19L20 20L16 22L14 26L12 22L8 20L12 19Z" fill="#fbbf24" />
            <path d="M51 25L53 28L57 29L53 31L51 35L49 31L45 29L49 28Z" fill="#fb7185" />
          </motion.g>
        </>
      );
    default:
      return (
        <g>
          <motion.circle
            cx="47"
            cy="17"
            r="3"
            fill="#99f6e4"
            animate={{ y: [0, -3, 0], opacity: [0.4, 1, 0.4] }}
            transition={{ duration: 1.5, repeat: Infinity, ease: 'easeInOut' }}
          />
          <motion.circle
            cx="53"
            cy="11"
            r="2"
            fill={propColor}
            animate={{ y: [0, -2, 0], opacity: [0.3, 0.9, 0.3] }}
            transition={{ duration: 1.7, repeat: Infinity, ease: 'easeInOut' }}
          />
        </g>
      );
  }
};

const AgentCharacter = ({
  isFailed,
  moodKey,
  size,
  tone,
}: {
  isFailed: boolean;
  moodKey: AgentMoodKey;
  size: 'md' | 'sm';
  tone: typeof failedTone;
}) => {
  const dimensions = sizeClasses[size];
  const colors = characterColors[moodKey];

  return (
    <div className={`relative flex ${dimensions.frame} flex-shrink-0 items-center justify-center ${tone.accent}`}>
      <motion.span
        className={`absolute inset-0 rounded-full bg-gradient-to-br ${tone.gradient} opacity-20 blur-lg`}
        animate={{ opacity: [0.12, 0.28, 0.12], scale: [0.88, 1.08, 0.88] }}
        transition={{ duration: moodKey === 'found' ? 1.2 : 2, repeat: Infinity, ease: 'easeInOut' }}
      />
      <motion.span
        className={`absolute inset-1 rounded-full border ${tone.ring}`}
        animate={{ scale: [0.96, 1.04, 0.96] }}
        transition={{ duration: 2.4, repeat: Infinity, ease: 'easeInOut' }}
      />
      <motion.svg
        className={`relative ${dimensions.svg} drop-shadow-sm`}
        data-agent-character={moodKey}
        viewBox="0 0 64 64"
        aria-hidden="true"
      >
        <motion.g
          animate={{
            rotate: moodKey === 'found' ? [0, -3, 3, 0] : [-1.2, 1.2, -1.2],
            y: moodKey === 'found' ? [0, -2, 0] : [0, -1, 0],
          }}
          style={{ transformOrigin: '32px 34px' }}
          transition={{ duration: moodKey === 'found' ? 0.95 : 2.2, repeat: Infinity, ease: 'easeInOut' }}
        >
          <ellipse cx="32" cy="57" rx="18" ry="4" fill="#0f172a" opacity="0.14" />
          <path d="M19 54C20.5 41 24.8 35 32 35C39.2 35 43.5 41 45 54Z" fill={colors.coat} />
          <path d="M27 36L32 45L37 36" fill={colors.trim} opacity="0.95" />
          <motion.path
            d="M21 42C16 43 14 47 13 51M43 42C48 43 50 47 51 51"
            fill="none"
            stroke="#f5c7a4"
            strokeLinecap="round"
            strokeWidth="4"
            animate={{ pathLength: [0.82, 1, 0.82] }}
            transition={{ duration: 1.8, repeat: Infinity, ease: 'easeInOut' }}
          />
          <circle cx="32" cy="23" r="12" fill="#ffd9b5" />
          <path d="M21 21C23 11 30 8 38 11C43 13 45 18 44 23C39 18 31 16 21 21Z" fill="#3b2f2f" />
          <circle cx="27.8" cy="23" r="1.4" fill="#1f2937" />
          <circle cx="36.2" cy="23" r="1.4" fill="#1f2937" />
          {renderMouth(moodKey, isFailed)}
          {renderAccessory(moodKey, colors.prop)}
        </motion.g>
      </motion.svg>
    </div>
  );
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
  const tone = isFailed ? failedTone : toneClasses[mood.tone];

  return (
    <div className="inline-flex min-w-0 items-center gap-3" data-agent-mood={mood.key}>
      <AgentCharacter isFailed={isFailed} moodKey={mood.key} size={size} tone={tone} />

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
  const tone = status === 'failed' ? failedTone : toneClasses[mood.tone];

  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: 6 }}
      className={`relative mb-4 overflow-hidden rounded-2xl border ${tone.ring} bg-white/85 p-3 shadow-soft backdrop-blur dark:bg-neutral-900/85`}
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
