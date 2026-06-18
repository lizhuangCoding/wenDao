import { Link, useLocation, Outlet } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Layout } from '../common';

export const AdminLayout = () => {
  const location = useLocation();
  const { t } = useTranslation();

  const menuItems = [
    { name: t('admin.stats'), path: '/admin/stats', icon: 'M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z' },
    { name: 'AI 观测', path: '/admin/ai-observability', icon: 'M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.25 8.25L18 9.25l-.25-1a2 2 0 00-1.5-1.5l-1-.25 1-.25a2 2 0 001.5-1.5l.25-1 .25 1a2 2 0 001.5 1.5l1 .25-1 .25a2 2 0 00-1.5 1.5zM16.5 20.25l-.5 1.5-.5-1.5a2.25 2.25 0 00-1.5-1.5l-1.5-.5 1.5-.5a2.25 2.25 0 001.5-1.5l.5-1.5.5 1.5a2.25 2.25 0 001.5 1.5l1.5.5-1.5.5a2.25 2.25 0 00-1.5 1.5z' },
    { name: t('users.title'), path: '/admin/users', icon: 'M15 19.128a9.38 9.38 0 002.625.372 9.337 9.337 0 004.121-.952 4.125 4.125 0 00-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 018.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0111.964-3.07M12 6.375a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zm8.25 2.25a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z' },
    { name: t('admin.articles'), path: '/admin/articles', icon: 'M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z' },
    { name: t('admin.categories'), path: '/admin/categories', icon: 'M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25H12' },
    { name: '合集', path: '/admin/collections', icon: 'M6.75 3.75h10.5A2.25 2.25 0 0119.5 6v12a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 18V6a2.25 2.25 0 012.25-2.25zM8.25 7.5h7.5M8.25 12h7.5M8.25 16.5h4.5' },
    { name: t('admin.comments'), path: '/admin/comments', icon: 'M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.123C7.69 12.94 7.05 12 5 12c0 0 4.556 3.75 9 8.25' },
    { name: t('knowledgeDocument.reviewTitle'), path: '/admin/knowledge-documents', icon: 'M4.5 4.5h15v15h-15z' },
    { name: t('broadcast.title'), path: '/admin/broadcast', icon: 'M11 5.882V19.24a1.76 1.76 0 01-3.417.592l-2.147-6.15M18 13a3 3 0 100-6M5.436 13.683A4.001 4.001 0 017 6h1.832c4.1 0 7.625-1.234 9.168-3v14c-1.543-1.766-5.067-3-9.168-3H7a3.988 3.988 0 01-1.564-.317z' },
    { name: t('settings.title'), path: '/admin/settings', icon: 'M11.42 15.17l7.25-7.25m0 0l-7.25 7.28m7.25-7.28L15.59 4.81m3.08 3.11L22 11.42M6.75 3.5A3.25 3.25 0 003.5 6.75v10.5A3.25 3.25 0 006.75 20.5h10.5a3.25 3.25 0 003.25-3.25V13' },
  ];

  return (
    <Layout>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 sm:py-10 lg:py-12 flex flex-col md:flex-row gap-5 sm:gap-8 min-h-[calc(100dvh-4rem)]">
        <aside className="w-full shrink-0 md:w-64">
          <nav className="flex gap-2 overflow-x-auto md:overflow-visible md:block md:space-y-1 scrollbar-hide bg-white dark:bg-neutral-800 rounded-2xl shadow-sm border border-neutral-100 dark:border-neutral-700 p-2">
            {menuItems.map((item) => {
              const isActive = location.pathname.startsWith(item.path);
              return (
                <Link
                  key={item.path}
                  to={item.path}
                  className={`flex shrink-0 whitespace-nowrap items-center gap-3 px-4 py-3 rounded-xl font-medium transition-all md:w-full ${
                    isActive
                      ? 'bg-primary-50 dark:bg-primary-900/30 text-primary-600 dark:text-primary-400'
                      : 'text-neutral-600 dark:text-neutral-400 hover:bg-neutral-50 dark:hover:bg-neutral-800 hover:text-neutral-900 dark:hover:text-neutral-200'
                  }`}
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={item.icon} />
                  </svg>
                  {item.name}
                </Link>
              );
            })}
          </nav>
        </aside>

        <main className="flex-1 min-w-0">
          <Outlet />
        </main>
      </div>
    </Layout>
  );
};
