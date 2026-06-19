import { BookOpen, Boxes, Network, Search, Sparkles, type LucideIcon } from 'lucide-react';
import { motion, useMotionValue, useReducedMotion } from 'framer-motion';
import { useTranslation } from 'react-i18next';

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

interface StarterPromptButtonProps {
  disabled: boolean;
  starterPrompt: StarterPrompt;
  onSelect: (prompt: string) => void;
}

const StarterPromptButton = ({ disabled, onSelect, starterPrompt }: StarterPromptButtonProps) => {
  const Icon = starterPrompt.icon;
  const x = useMotionValue(0);
  const y = useMotionValue(0);
  const prefersReducedMotion = useReducedMotion();

  const handleMagneticPointerMove = (event: React.MouseEvent<HTMLButtonElement>) => {
    if (disabled || prefersReducedMotion) return;

    const rect = event.currentTarget.getBoundingClientRect();
    x.set((event.clientX - rect.left - rect.width / 2) * 0.08);
    y.set((event.clientY - rect.top - rect.height / 2) * 0.08);
  };

  const handleMagneticPointerLeave = () => {
    x.set(0);
    y.set(0);
  };

  return (
    <motion.button
      key={starterPrompt.title}
      type="button"
      disabled={disabled}
      onClick={() => onSelect(starterPrompt.prompt)}
      onMouseMove={handleMagneticPointerMove}
      onMouseLeave={handleMagneticPointerLeave}
      style={{ x, y }}
      whileHover={{ scale: 1.01 }}
      whileTap={{ scale: 0.99 }}
      className="group flex min-h-28 items-start gap-4 rounded-2xl border border-neutral-200 bg-white p-4 text-left shadow-sm transition-colors hover:border-primary-200 hover:shadow-soft disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-700 dark:bg-neutral-800 dark:hover:border-primary-800"
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
    </motion.button>
  );
};

export const StarterPrompts = ({ disabled = false, heading, onSelect, subheading }: StarterPromptsProps) => (
  <StarterPromptShell disabled={disabled} heading={heading} onSelect={onSelect} subheading={subheading} />
);

const StarterPromptShell = ({ disabled, heading, onSelect, subheading }: StarterPromptsProps) => {
  const { t } = useTranslation();

  const starterPrompts: StarterPrompt[] = [
    {
      title: t('chat.starterK8sTitle'),
      description: t('chat.starterK8sDescription'),
      prompt: t('chat.starterK8sPrompt'),
      icon: Boxes,
    },
    {
      title: t('chat.starterDistributedTitle'),
      description: t('chat.starterDistributedDescription'),
      prompt: t('chat.starterDistributedPrompt'),
      icon: Network,
    },
    {
      title: t('chat.starterTradeoffTitle'),
      description: t('chat.starterTradeoffDescription'),
      prompt: t('chat.starterTradeoffPrompt'),
      icon: Search,
    },
    {
      title: t('chat.starterOutlineTitle'),
      description: t('chat.starterOutlineDescription'),
      prompt: t('chat.starterOutlinePrompt'),
      icon: BookOpen,
    },
  ];

  return (
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
        {starterPrompts.map((starterPrompt) => (
          <StarterPromptButton
            key={starterPrompt.title}
            disabled={disabled ?? false}
            starterPrompt={starterPrompt}
            onSelect={onSelect}
          />
        ))}
      </div>
    </div>
  </div>
  );
};
