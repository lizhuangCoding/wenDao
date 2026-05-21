import { BookOpen, Boxes, Network, Search, Sparkles, type LucideIcon } from 'lucide-react';

interface StarterPrompt {
  title: string;
  description: string;
  prompt: string;
  icon: LucideIcon;
}

interface StarterPromptsProps {
  disabled?: boolean;
  heading: string;
  subheading: string;
  onSelect: (prompt: string) => void;
}

const STARTER_PROMPTS: StarterPrompt[] = [
  {
    title: '调研 K8s',
    description: '从架构、组件、场景和学习路径切入',
    prompt: '帮我调研一下 K8s，请从核心架构、关键组件、典型使用场景和学习路径四个角度总结。',
    icon: Boxes,
  },
  {
    title: '总结分布式系统',
    description: '提炼核心概念、常见问题和工程取舍',
    prompt: '帮我总结一下分布式系统的核心概念，包括一致性、可用性、分区容错、共识、事务和可观测性。',
    icon: Network,
  },
  {
    title: '做技术选型',
    description: '比较方案优劣，给出可执行建议',
    prompt: '我想做一个技术选型，请帮我按场景、成本、复杂度、风险和长期维护性做一份对比分析。',
    icon: Search,
  },
  {
    title: '生成文章大纲',
    description: '把一个主题拆成清晰的写作结构',
    prompt: '请帮我为一篇技术文章生成大纲，要求结构清晰、观点明确，并给出每一节应该写什么。',
    icon: BookOpen,
  },
];

export const StarterPrompts = ({ disabled = false, heading, onSelect, subheading }: StarterPromptsProps) => (
  <div className="flex min-h-full flex-col items-center justify-center text-center">
    <div className="mx-auto max-w-3xl">
      <div className="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-2xl bg-primary-50 text-primary-500 shadow-soft dark:bg-primary-900/30 dark:text-primary-300">
        <Sparkles className="h-8 w-8" aria-hidden="true" />
      </div>
      <h3 className="mb-2 text-2xl font-serif font-black text-neutral-900 dark:text-neutral-100">
        {heading}
      </h3>
      <p className="mx-auto max-w-md text-sm font-medium leading-6 text-neutral-400 dark:text-neutral-500">
        {subheading}
      </p>

      <div className="mt-8 grid gap-3 sm:grid-cols-2">
        {STARTER_PROMPTS.map((starterPrompt) => {
          const Icon = starterPrompt.icon;

          return (
            <button
              key={starterPrompt.title}
              type="button"
              disabled={disabled}
              onClick={() => onSelect(starterPrompt.prompt)}
              className="group flex min-h-28 items-start gap-4 rounded-2xl border border-neutral-100 bg-white p-4 text-left shadow-sm transition-all hover:-translate-y-0.5 hover:border-primary-200 hover:shadow-soft disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-700 dark:bg-neutral-800 dark:hover:border-primary-800"
            >
              <span className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-neutral-500 transition-colors group-hover:bg-primary-50 group-hover:text-primary-600 dark:bg-neutral-700 dark:text-neutral-300 dark:group-hover:bg-primary-900/30 dark:group-hover:text-primary-300">
                <Icon className="h-5 w-5" aria-hidden="true" />
              </span>
              <span className="min-w-0">
                <span className="block text-sm font-black text-neutral-800 dark:text-neutral-100">
                  {starterPrompt.title}
                </span>
                <span className="mt-1 block text-xs font-medium leading-5 text-neutral-500 dark:text-neutral-400">
                  {starterPrompt.description}
                </span>
              </span>
            </button>
          );
        })}
      </div>
    </div>
  </div>
);
