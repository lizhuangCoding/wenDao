import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { articleApi, categoryApi, siteApi } from '@/api';
import { Layout, Loading, Pagination, EmptyState, ErrorState } from '@/components/common';
import { ArticleCard } from '@/components/article';
import { motion, AnimatePresence } from 'framer-motion';

export const Home = () => {
  const { t } = useTranslation();
  const [currentPage, setCurrentPage] = useState(1);
  const [selectedCategory, setSelectedCategory] = useState<number>();
  const [searchKeyword, setSearchKeyword] = useState('');
  const [inputValue, setInputValue] = useState('');

  // 获取网站标语
  const { data: siteData } = useQuery({
    queryKey: ['slogan'],
    queryFn: siteApi.getSlogan,
    staleTime: 5 * 60 * 1000,
  });

  // 获取分类列表
  const { data: categories } = useQuery({
    queryKey: ['categories'],
    queryFn: categoryApi.getCategories,
  });

  // 获取文章列表
  const { data: articlesData, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['articles', currentPage, selectedCategory, searchKeyword],
    queryFn: () =>
      articleApi.getArticles({
        page: currentPage,
        pageSize: 9,
        category_id: selectedCategory,
        keyword: searchKeyword,
      }),
  });
  const totalPages = Math.max(1, articlesData?.totalPages ?? 1);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setSearchKeyword(inputValue);
    setCurrentPage(1);
  };

  return (
    <Layout>
      <div className="max-w-display mx-auto px-6 sm:px-10 lg:px-12 py-24">
        {/* Hero Section */}
        <section className="mb-32 relative">
          <motion.div
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 1, ease: [0.22, 1, 0.36, 1] }}
            className="max-w-5xl"
          >
            <h1 className="text-6xl md:text-8xl font-serif font-black text-neutral-900 dark:text-neutral-100 leading-[1.05] tracking-tight mb-10">
              {siteData?.slogan || '我不在执着于得到，而是享受走到'}
            </h1>
            <div className="flex flex-col md:flex-row md:items-center gap-8">
              <div className="flex items-center gap-4 text-primary-600 dark:text-primary-400">
                <div className="w-12 h-[2px] bg-primary-500"></div>
                <span className="text-sm font-bold tracking-[0.2em] uppercase">{t('home.heroSub')}</span>
              </div>

              {/* Search Integrated into Hero */}
              <form onSubmit={handleSearch} className="relative flex-1 max-w-md">
                <input
                  type="text"
                  placeholder={t('home.searchPlaceholder')}
                  className="w-full bg-transparent dark:bg-transparent border-b-2 border-neutral-200 dark:border-neutral-700 py-2 pl-0 pr-10 text-sm font-bold tracking-widest focus:outline-none focus:border-primary-500 text-neutral-900 dark:text-neutral-100 placeholder-neutral-400"
                  value={inputValue}
                  onChange={(e) => setInputValue(e.target.value)}
                />
                <button type="submit" className="absolute right-0 top-1/2 -translate-y-1/2 text-neutral-400 dark:text-neutral-500 hover:text-primary-600 dark:hover:text-primary-400 transition-colors">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                  </svg>
                </button>
              </form>
            </div>
          </motion.div>

          {/* Decorative element */}
          <div className="absolute -top-40 -right-20 w-96 h-96 bg-primary-50 dark:bg-primary-900/20 rounded-full blur-[120px] -z-10 opacity-40"></div>
        </section>

        {/* Categories Bar */}
        <section className="mb-20 border-b border-neutral-100 dark:border-neutral-800 pb-10 overflow-x-auto scrollbar-hide">
          <div className="flex items-center gap-10 whitespace-nowrap">
            <button
              onClick={() => { setSelectedCategory(undefined); setCurrentPage(1); }}
              className={`text-xs font-bold tracking-[0.3em] transition-all relative py-2 ${
                selectedCategory === undefined
                  ? 'text-primary-600 dark:text-primary-400 after:absolute after:bottom-0 after:left-0 after:w-full after:h-1 after:bg-primary-500'
                  : 'text-neutral-400 dark:text-neutral-500 hover:text-neutral-900 dark:hover:text-neutral-100'
              }`}
            >
              {t('home.allArticles')}
            </button>
            {categories?.map((cat) => (
              <button
                key={cat.id}
                onClick={() => { setSelectedCategory(cat.id); setCurrentPage(1); }}
                className={`text-xs font-bold tracking-[0.3em] transition-all relative py-2 ${
                  selectedCategory === cat.id
                    ? 'text-primary-600 dark:text-primary-400 after:absolute after:bottom-0 after:left-0 after:w-full after:h-1 after:bg-primary-500'
                    : 'text-neutral-400 dark:text-neutral-500 hover:text-neutral-900 dark:hover:text-neutral-100'
                }`}
              >
                {cat.name.toUpperCase()}
              </button>
            ))}
          </div>
        </section>

        {/* Article Grid */}
        {isLoading ? (
          <div className="py-20 flex justify-center"><Loading /></div>
        ) : isError ? (
          <ErrorState message={(error as any)?.message || '文章列表加载失败'} onRetry={() => refetch()} />
        ) : (
          <>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-x-12 gap-y-24">
              <AnimatePresence mode="popLayout">
                {articlesData?.data?.map((article, index) => (
                  <motion.div
                    key={article.id}
                    initial={{ opacity: 0, y: 40 }}
                    whileInView={{ opacity: 1, y: 0 }}
                    viewport={{ once: true }}
                    transition={{ duration: 0.7, delay: (index % 3) * 0.1 }}
                  >
                    <ArticleCard article={article} />
                  </motion.div>
                ))}
              </AnimatePresence>
            </div>

            {articlesData?.data?.length === 0 && (
              <EmptyState title={t('home.noResults')} className="py-32" />
            )}

            {articlesData && (
              <Pagination
                page={currentPage}
                totalPages={totalPages}
                onChange={setCurrentPage}
                previousLabel={t('home.newer')}
                nextLabel={t('home.older')}
                className="mt-40 border-t border-neutral-100 pt-16 dark:border-neutral-800"
              />
            )}
          </>
        )}
      </div>
    </Layout>
  );
};
