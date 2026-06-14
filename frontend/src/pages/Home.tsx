import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { articleApi, categoryApi, siteApi } from '@/api';
import { Layout, Loading, Pagination, EmptyState, ErrorState, CursorCometTrail } from '@/components/common';
import { ArticleCard } from '@/components/article';
import { ArticlePlanetHero } from '@/components/home';
import { motion, AnimatePresence } from 'framer-motion';
import type { ArticlePlanetTimeMode } from '@/components/home/articlePlanetTime';

export const Home = () => {
  const { t } = useTranslation();
  const [currentPage, setCurrentPage] = useState(1);
  const [selectedCategory, setSelectedCategory] = useState<number>();
  const [searchKeyword, setSearchKeyword] = useState('');
  const [inputValue, setInputValue] = useState('');
  const [planetTimeMode, setPlanetTimeMode] = useState<ArticlePlanetTimeMode>('all');

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
    placeholderData: (previousData) => previousData,
  });

  const {
    data: orbitData,
    isLoading: isOrbitLoading,
    isError: isOrbitError,
  } = useQuery({
    queryKey: ['article-orbit'],
    queryFn: articleApi.getArticleOrbit,
    staleTime: 5 * 60 * 1000,
  });

  const totalPages = Math.max(1, articlesData?.totalPages ?? 1);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setSearchKeyword(inputValue);
    setCurrentPage(1);
  };

  const handleCategoryChange = (categoryId?: number) => {
    setSelectedCategory(categoryId);
    setCurrentPage(1);
  };

  return (
    <Layout>
      <CursorCometTrail />
      <ArticlePlanetHero
        articles={orbitData?.data ?? []}
        categories={categories}
        inputValue={inputValue}
        isError={isOrbitError}
        isLoading={isOrbitLoading}
        timeMode={planetTimeMode}
        selectedCategory={selectedCategory}
        slogan={siteData?.slogan}
        onCategoryChange={handleCategoryChange}
        onSearch={handleSearch}
        onSearchInputChange={setInputValue}
        onTimeModeChange={setPlanetTimeMode}
      />

      <div className="relative z-10 max-w-display mx-auto px-5 sm:px-10 lg:px-12 py-16 sm:py-24">

        {/* Article Grid */}
        {isLoading ? (
          <div className="py-20 flex justify-center"><Loading /></div>
        ) : isError ? (
          <ErrorState message={(error as any)?.message || t('home.articleListLoadFailed')} onRetry={() => refetch()} />
        ) : (
          <>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-x-12 gap-y-16 sm:gap-y-24">
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
