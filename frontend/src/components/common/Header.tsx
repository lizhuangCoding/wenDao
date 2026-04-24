import { Link, useNavigate } from 'react-router-dom';
import { useScrollDirection, useAuth } from '@/hooks';
import { useThemeStore } from '@/store';
import { useTranslation } from 'react-i18next';
import { cn } from '@/utils';

export const Header = () => {
  const scrollDirection = useScrollDirection();
  const { user, isAuthenticated, logout } = useAuth();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { theme, toggleTheme, language, setLanguage } = useThemeStore();

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  const getThemeIcon = () => {
    if (theme === 'auto') {
      return (
        <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
        </svg>
      );
    } else if (theme === 'dark') {
      return (
        <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
        </svg>
      );
    } else {
      return (
        <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
        </svg>
      );
    }
  };

  return (
    <header
      className={cn(
        'fixed top-0 left-0 right-0 z-50 transition-all duration-500 ease-in-out',
        'bg-white/70 dark:bg-neutral-900/70 backdrop-blur-xl border-b border-neutral-200/50 dark:border-neutral-800/50 shadow-sm',
        scrollDirection === 'down' ? '-translate-y-full' : 'translate-y-0'
      )}
    >
      <div className="max-w-display mx-auto px-6 sm:px-10 lg:px-12">
        <div className="flex items-center justify-between h-20">
          {/* Logo */}
          <Link
            to="/"
            className="group flex items-center gap-2"
          >
            <span className="text-3xl font-serif font-black tracking-tighter text-neutral-900 dark:text-neutral-100 group-hover:text-primary-600 transition-colors">
              问道<span className="text-primary-500 text-4xl">.</span>
            </span>
          </Link>

          {/* Navigation */}
          <nav className="hidden md:flex items-center gap-6">
            <Link
              to="/"
              className="text-sm font-medium tracking-wide text-neutral-500 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-neutral-100 transition-colors relative after:absolute after:bottom-[-4px] after:left-0 after:w-0 after:h-[2px] after:bg-primary-500 hover:after:w-full after:transition-all"
            >
              {t('nav.explore')}
            </Link>
            <Link
              to="/ai-chat"
              className="text-sm font-medium tracking-wide text-neutral-500 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-neutral-100 transition-colors relative after:absolute after:bottom-[-4px] after:left-0 after:w-0 after:h-[2px] after:bg-primary-500 hover:after:w-full after:transition-all"
            >
              {t('nav.aiAssistant')}
            </Link>

            {/* Theme & Language Controls */}
            <div className="flex items-center gap-2 pl-4 border-l border-neutral-200/60 dark:border-neutral-700/60">
              {/* Language Switch */}
              <button
                onClick={() => setLanguage(language === 'zh' ? 'en' : 'zh')}
                className="flex items-center gap-1 px-2 py-1.5 text-sm font-medium text-neutral-500 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-neutral-100 transition-colors"
                title={t('common.language')}
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 5h12M9 3v2m1.048 9.5A18.022 18.022 0 016.412 9m6.088 9h7M11 21l5-10 5 10M12.751 5C11.783 10.77 8.07 15.61 3 18.129" />
                </svg>
                <span className="uppercase">{language}</span>
              </button>

              {/* Theme Switch */}
              <button
                onClick={toggleTheme}
                className="p-2 text-neutral-500 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-neutral-100 transition-colors"
                title={t('common.theme')}
              >
                {getThemeIcon()}
              </button>
            </div>

            {/* User Menu */}
            {isAuthenticated ? (
              <div className="flex items-center gap-4 pl-4 border-l border-neutral-200/60 dark:border-neutral-700/60">
                <Link to="/profile" className="group flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full overflow-hidden border-2 border-primary-100 dark:border-primary-800 shadow-soft transition-transform group-hover:scale-105">
                    <img
                      src={user?.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${user?.username}`}
                      alt={user?.username}
                      className="w-full h-full object-cover"
                    />
                  </div>
                  <div className="flex flex-col">
                    <span className="text-sm font-bold text-neutral-800 dark:text-neutral-200 leading-tight transition-colors group-hover:text-primary-600 dark:group-hover:text-primary-400">{user?.username}</span>
                    <span className="text-[10px] text-neutral-400 font-bold uppercase tracking-widest">{user?.role}</span>
                  </div>
                </Link>

                {user?.role === 'admin' && (
                  <Link
                    to="/admin"
                    className="p-2 text-neutral-400 dark:text-neutral-500 hover:text-primary-600 dark:hover:text-primary-400 transition-colors"
                    title={t('nav.admin')}
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                    </svg>
                  </Link>
                )}

                <button
                  onClick={handleLogout}
                  className="text-xs font-bold text-neutral-400 dark:text-neutral-500 hover:text-red-500 dark:hover:text-red-400 transition-colors border border-neutral-200 dark:border-neutral-700 px-3 py-1.5 rounded-full"
                >
                  {t('nav.logout')}
                </button>
              </div>
            ) : (
              <div className="flex items-center gap-4 pl-4 border-l border-neutral-200/60 dark:border-neutral-700/60">
                <Link to="/login" className="text-sm font-bold text-neutral-500 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-neutral-100 transition-colors">
                  {t('nav.login')}
                </Link>
                <Link to="/register" className="bg-neutral-900 dark:bg-neutral-100 text-white dark:text-neutral-900 text-xs font-bold tracking-widest px-6 py-2.5 rounded-full hover:bg-primary-600 dark:hover:bg-primary-500 transition-all shadow-soft">
                  {t('nav.signup')}
                </Link>
              </div>
            )}
          </nav>
        </div>
      </div>
    </header>
  );
};