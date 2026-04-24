import { useState, useEffect, useCallback, useRef } from 'react';
import { TocItem } from '@/utils/markdown';

interface Props {
  headings: TocItem[];
}

export const TableOfContents: React.FC<Props> = ({ headings }) => {
  const [activeId, setActiveId] = useState<string>('');
  const isClickingRef = useRef(false);
  const clickTimerRef = useRef<ReturnType<typeof setTimeout>>();

  const handleClick = useCallback((id: string) => {
    const element = document.getElementById(id);
    if (element) {
      isClickingRef.current = true;
      setActiveId(id);
      element.scrollIntoView({ behavior: 'smooth' });
      history.pushState(null, '', `#${id}`);

      if (clickTimerRef.current) clearTimeout(clickTimerRef.current);
      clickTimerRef.current = setTimeout(() => {
        isClickingRef.current = false;
      }, 1000);
    }
  }, []);

  useEffect(() => {
    if (headings.length === 0) return;

    const handleScroll = () => {
      if (isClickingRef.current) return;

      const headerOffset = 100;
      let currentId = headings[0].id;

      for (const heading of headings) {
        const element = document.getElementById(heading.id);
        if (element) {
          const rect = element.getBoundingClientRect();
          if (rect.top <= headerOffset) {
            currentId = heading.id;
          } else {
            break;
          }
        }
      }

      setActiveId(currentId);
    };

    window.addEventListener('scroll', handleScroll, { passive: true });
    handleScroll();

    return () => {
      window.removeEventListener('scroll', handleScroll);
      if (clickTimerRef.current) clearTimeout(clickTimerRef.current);
    };
  }, [headings]);

  if (headings.length === 0) return null;

  return (
    <div className="sticky top-24 max-h-[calc(100vh-140px)] overflow-y-auto scrollbar-hide py-2">
      <h4 className="text-xs font-bold text-neutral-400 mb-4 uppercase tracking-widest pl-4">
        目录导航
      </h4>
      <div className="relative border-l border-neutral-200 ml-4">
        <div
          className="absolute left-[-1px] w-[2px] bg-primary-500 transition-all duration-300 ease-in-out"
          style={{
            height: '24px',
            top: `${headings.findIndex((h) => h.id === activeId) * 32 + 4}px`,
            opacity: activeId ? 1 : 0,
          }}
        />

        <ul className="space-y-0">
          {headings.map((heading) => (
            <li key={heading.id} className="h-8 flex items-center">
              <button
                onClick={() => handleClick(heading.id)}
                className={`w-full text-left text-sm py-1 pl-4 transition-all duration-200 truncate ${
                  activeId === heading.id
                    ? 'text-primary-600 dark:text-primary-400 font-bold'
                    : 'text-neutral-500 dark:text-neutral-300 hover:text-neutral-800 dark:hover:text-neutral-100 hover:pl-5'
                }`}
                style={{
                  paddingLeft: `${16 + Math.max(0, heading.level - 2) * 12}px`,
                  opacity: heading.level > 3 ? 0.8 : 1,
                }}
              >
                {heading.text}
              </button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
};
