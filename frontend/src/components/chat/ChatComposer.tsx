import type { FormEvent, KeyboardEvent, RefObject } from 'react';
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
}: ChatComposerProps) => (
  <form onSubmit={onSubmit} className="relative group">
    <textarea
      ref={textareaRef}
      rows={1}
      value={input}
      onChange={(event) => onChange(event.target.value)}
      onKeyDown={onKeyDown}
      placeholder={requiresUserInput && pendingQuestion ? pendingQuestion : placeholder}
      className="max-h-40 min-h-[56px] w-full resize-none overflow-y-auto bg-neutral-50 dark:bg-neutral-700 border-2 border-neutral-100 dark:border-neutral-600 rounded-2xl py-4 px-6 pr-16 text-sm font-bold leading-6 text-neutral-900 dark:text-neutral-100 placeholder-neutral-400 dark:placeholder-neutral-500 transition-all focus:outline-none focus:border-primary-500 focus:bg-white dark:focus:bg-neutral-600 focus:shadow-elevated"
      disabled={disabled}
      aria-label="给 AI 助手发送消息，Shift+Enter 换行"
    />
    <button
      type="submit"
      disabled={disabled || !input.trim()}
      className="absolute bottom-3 right-3 w-10 h-10 bg-neutral-900 dark:bg-primary-600 text-white rounded-xl flex items-center justify-center transition-all hover:bg-primary-600 dark:hover:bg-primary-500 disabled:opacity-20 active:scale-90"
      aria-label="发送消息"
    >
      <SendHorizontal className="h-5 w-5" aria-hidden="true" />
    </button>
    <p className="text-[10px] text-center text-neutral-300 dark:text-neutral-600 font-bold uppercase tracking-widest mt-4">
      {poweredBy}
    </p>
  </form>
);
