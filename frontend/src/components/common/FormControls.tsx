import {
  Children,
  isValidElement,
  forwardRef,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type InputHTMLAttributes,
  type KeyboardEvent,
  type ReactNode,
  type ReactElement,
  type SelectHTMLAttributes,
  type TextareaHTMLAttributes,
} from 'react';
import { Check, ChevronDown } from 'lucide-react';
import { cn } from '@/utils';

const controlClassName =
  'w-full rounded-xl border border-neutral-200 bg-white px-3 py-2.5 text-sm text-neutral-800 outline-none transition-colors placeholder:text-neutral-400 focus:border-primary-400 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:placeholder:text-neutral-500';

interface TextInputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'className'> {
  className?: string;
  leading?: ReactNode;
}

export const TextInput = forwardRef<HTMLInputElement, TextInputProps>(
  ({ className, leading, ...props }, ref) => (
    <div className={cn('relative', className)}>
      {leading ? (
        <div className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-neutral-400 dark:text-neutral-500">
          {leading}
        </div>
      ) : null}
      <input
        ref={ref}
        className={cn(controlClassName, leading ? 'pl-9' : '')}
        {...props}
      />
    </div>
  )
);

TextInput.displayName = 'TextInput';

type SelectOptionElement = ReactElement<{
  value?: string | number;
  children?: ReactNode;
  disabled?: boolean;
}>;

interface SelectInputProps extends Omit<SelectHTMLAttributes<HTMLSelectElement>, 'className' | 'children'> {
  className?: string;
  children: ReactNode;
}

const getOptionLabel = (children: ReactNode): string => {
  if (typeof children === 'string' || typeof children === 'number') return String(children);
  if (Array.isArray(children)) return children.map(getOptionLabel).join('');
  return '';
};

export const SelectInput = forwardRef<HTMLButtonElement, SelectInputProps>(
  ({ className, children, value, defaultValue, onChange, disabled, name, id, 'aria-label': ariaLabel }, ref) => {
    const generatedId = useId();
    const listboxId = `${id || generatedId}-listbox`;
    const rootRef = useRef<HTMLDivElement>(null);
    const [isOpen, setIsOpen] = useState(false);
    const [internalValue, setInternalValue] = useState(() => String(value ?? defaultValue ?? ''));
    const selectedValue = String(value ?? internalValue);
    const options = useMemo(() => {
      return Children.toArray(children)
        .filter(isValidElement)
        .map((child) => {
          const option = child as SelectOptionElement;
          const label = getOptionLabel(option.props.children);
          const optionValue = String(option.props.value ?? label);
          return {
            value: optionValue,
            label,
            disabled: Boolean(option.props.disabled),
          };
        });
    }, [children]);
    const selectedOption = options.find((option) => option.value === selectedValue) || options[0];

    useEffect(() => {
      if (value !== undefined) {
        setInternalValue(String(value));
      }
    }, [value]);

    useEffect(() => {
      if (!isOpen) return;
      const handlePointerDown = (event: PointerEvent) => {
        if (!rootRef.current?.contains(event.target as Node)) {
          setIsOpen(false);
        }
      };
      document.addEventListener('pointerdown', handlePointerDown);
      return () => document.removeEventListener('pointerdown', handlePointerDown);
    }, [isOpen]);

    const commitValue = (nextValue: string) => {
      setInternalValue(nextValue);
      setIsOpen(false);
      onChange?.({
        target: { value: nextValue, name },
        currentTarget: { value: nextValue, name },
      } as ChangeEvent<HTMLSelectElement>);
    };

    const handleKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
      if (event.key === 'Escape') {
        setIsOpen(false);
      }
      if (event.key === 'ArrowDown' || event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        setIsOpen(true);
      }
    };

    return (
      <div ref={rootRef} className={cn('relative min-w-36', className)}>
        <button
          ref={ref}
          id={id}
          type="button"
          role="combobox"
          aria-controls={listboxId}
          aria-expanded={isOpen}
          aria-label={ariaLabel}
          disabled={disabled}
          onClick={() => setIsOpen((open) => !open)}
          onKeyDown={handleKeyDown}
          className={cn(
            controlClassName,
            'flex min-h-10 items-center justify-between gap-3 pr-2.5 text-left shadow-sm hover:border-neutral-300 hover:bg-neutral-50 disabled:cursor-not-allowed disabled:opacity-60 dark:hover:border-neutral-600 dark:hover:bg-neutral-800'
          )}
        >
          <span className="truncate">{selectedOption?.label}</span>
          <ChevronDown className={cn('h-4 w-4 shrink-0 text-neutral-400 transition-transform', isOpen ? 'rotate-180' : '')} />
        </button>

        {isOpen && (
          <div
            id={listboxId}
            role="listbox"
            className="absolute z-50 mt-2 max-h-64 w-full overflow-y-auto rounded-xl border border-neutral-200 bg-white p-1.5 shadow-xl shadow-neutral-900/10 dark:border-neutral-700 dark:bg-neutral-900 dark:shadow-black/30"
          >
            {options.map((option) => {
              const isSelected = option.value === selectedValue;
              return (
                <button
                  key={option.value}
                  type="button"
                  role="option"
                  aria-selected={isSelected}
                  disabled={option.disabled}
                  onClick={() => commitValue(option.value)}
                  className={cn(
                    'flex w-full items-center justify-between gap-3 rounded-lg px-3 py-2 text-left text-sm font-medium transition-colors',
                    isSelected
                      ? 'bg-primary-50 text-primary-700 dark:bg-primary-500/15 dark:text-primary-200'
                      : 'text-neutral-700 hover:bg-neutral-100 dark:text-neutral-200 dark:hover:bg-neutral-800',
                    option.disabled ? 'cursor-not-allowed opacity-50' : ''
                  )}
                >
                  <span className="truncate">{option.label}</span>
                  {isSelected && <Check className="h-4 w-4 shrink-0" />}
                </button>
              );
            })}
          </div>
        )}
      </div>
    );
  }
);

SelectInput.displayName = 'SelectInput';

interface TextAreaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  className?: string;
}

export const TextArea = forwardRef<HTMLTextAreaElement, TextAreaProps>(
  ({ className, ...props }, ref) => (
    <textarea
      ref={ref}
      className={cn(controlClassName, 'min-h-28 resize-y', className)}
      {...props}
    />
  )
);

TextArea.displayName = 'TextArea';
