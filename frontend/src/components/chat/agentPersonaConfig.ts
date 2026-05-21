import { resolveAgentMood, type AgentMoodKey } from '@/utils/agentMood';

export type ToneKey = ReturnType<typeof resolveAgentMood>['tone'];

export interface AgentTone {
  accent: string;
  badge: string;
  gradient: string;
  ring: string;
}

export const toneClasses: Record<ToneKey, AgentTone> = {
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

export const failedTone: AgentTone = {
  accent: 'text-red-600 dark:text-red-300',
  badge: 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-200',
  gradient: 'from-red-300 via-rose-400 to-red-600',
  ring: 'border-red-200 dark:border-red-800/60',
};

export const personaSizeClasses = {
  md: {
    frame: 'h-14 w-14',
    svg: 'h-14 w-14',
  },
  sm: {
    frame: 'h-11 w-11',
    svg: 'h-11 w-11',
  },
};

export const characterColors: Record<AgentMoodKey, { coat: string; trim: string; prop: string }> = {
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
