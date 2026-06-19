import type { FormEvent, KeyboardEvent, RefObject } from 'react';
import { useTranslation } from 'react-i18next';
import { SendHorizontal } from 'lucide-react';

interface ChatComposerProps {
  disabled: boolean;
  input: string;
  pendingQuestion?: string | null;
  placeholder: string;
  poweredBy: string;
  requiresUserInput: boolean;
  textareaRef: RefObject<HTMLTextAreaElement>;
  onChange: (value: string) => void;
  onKeyDown: (event: KeyboardEvent<HTMLTextAreaElement>) => void;
  onSubmit: (event: FormEvent) => void;
}

export const ChatComposer = ({
  disabled,
  input,
  onChange,
  onKeyDown,
  onSubmit,
  pendingQuestion,
  placeholder,
  poweredBy,
  requiresUserInput,
  textareaRef,
}: ChatComposerProps) => {
  const { t } = useTranslation();

  return (
    <form onSubmit={onSubmit} className="relative group">
      <textarea
        ref={textareaRef}
        rows={1}
        value={input}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={onKeyDown}
        placeholder={requiresUserInput && pendingQuestion ? pendingQuestion : placeholder}
        className="max-h-40 min-h-[52px] w-full resize-none overflow-y-auto bg-neutral-50 dark:bg-neutral-700 border-2 border-neutral-200 dark:border-neutral-600 rounded-2xl py-3.5 pl-4 pr-14 sm:min-h-[56px] sm:py-4 sm:pl-6 sm:pr-16 text-sm font-bold leading-6 text-neutral-900 dark:text-neutral-100 placeholder-neutral-400 dark:placeholder-neutral-500 transition-all focus:outline-none focus:border-primary-500 focus:bg-white dark:focus:bg-neutral-600 focus:shadow-elevated"
        disabled={disabled}
        aria-label={t('chat.composerAriaLabel')}
      />
      <button
        type="submit"
        disabled={disabled || !input.trim()}
        className="absolute bottom-3 right-3 h-9 w-9 sm:h-10 sm:w-10 bg-neutral-900 dark:bg-primary-600 text-white rounded-xl flex items-center justify-center transition-all hover:bg-primary-600 dark:hover:bg-primary-500 disabled:opacity-20 active:scale-90"
        aria-label={t('chat.send')}
      >
        <SendHorizontal className="h-5 w-5" aria-hidden="true" />
      </button>
      <p className="text-[10px] text-center text-neutral-300 dark:text-neutral-600 font-bold uppercase tracking-widest mt-4">
        {poweredBy}
      </p>
    </form>
  );
};
