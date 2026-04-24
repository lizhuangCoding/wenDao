import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import i18n from '@/i18n';

type ThemeMode = 'light' | 'dark' | 'auto';

interface ThemeState {
  theme: ThemeMode;
  language: 'zh' | 'en';
  toggleTheme: () => void;
  setTheme: (theme: ThemeMode) => void;
  setLanguage: (lang: 'zh' | 'en') => void;
  initTheme: () => void;
}

// 获取当前时间应该是什么模式
const getAutoTheme = (): 'light' | 'dark' => {
  const hour = new Date().getHours();
  // 早上 6 点到晚上 6 点为白天
  return hour >= 6 && hour < 18 ? 'light' : 'dark';
};

export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      theme: 'auto',
      language: 'zh',
      toggleTheme: () => {
        const current = get().theme;
        let next: ThemeMode;
        if (current === 'light') next = 'dark';
        else if (current === 'dark') next = 'auto';
        else next = 'light';
        set({ theme: next });
        get().initTheme();
      },
      setTheme: (theme) => {
        set({ theme });
        get().initTheme();
      },
      setLanguage: (lang) => {
        set({ language: lang });
        i18n.changeLanguage(lang);
      },
      initTheme: () => {
        const { theme } = get();
        let actualTheme: 'light' | 'dark';

        if (theme === 'auto') {
          actualTheme = getAutoTheme();
        } else {
          actualTheme = theme;
        }

        if (actualTheme === 'dark') {
          document.documentElement.classList.add('dark');
        } else {
          document.documentElement.classList.remove('dark');
        }
      },
    }),
    {
      name: 'wendao-theme-settings',
      onRehydrateStorage: () => (state) => {
        // Apply theme on load
        if (state) {
          // 先设置语言
          if (state.language) {
            i18n.changeLanguage(state.language);
          }
          // 然后应用主题
          setTimeout(() => {
            let actualTheme: 'light' | 'dark';
            if (state.theme === 'auto') {
              actualTheme = getAutoTheme();
            } else {
              actualTheme = state.theme;
            }
            if (actualTheme === 'dark') {
              document.documentElement.classList.add('dark');
            } else {
              document.documentElement.classList.remove('dark');
            }
          }, 0);
        }
      },
    }
  )
);