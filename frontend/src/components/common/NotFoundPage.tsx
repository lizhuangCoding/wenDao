import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { ArrowLeft, Home, Shuffle, Sparkles } from 'lucide-react';
import { articleApi } from '@/api';
import { Layout } from './Layout';
import { Button } from './Button';

export const NotFoundPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { data } = useQuery({
    queryKey: ['not-found-recommended-articles'],
    queryFn: () => articleApi.getArticles({ page: 1, pageSize: 3 }),
    staleTime: 5 * 60 * 1000,
  });
  const recommendedArticles = useMemo(() => data?.data ?? [], [data?.data]);
  const randomArticle = useMemo(() => {
    if (recommendedArticles.length === 0) return undefined;
    return recommendedArticles[Math.floor(Math.random() * recommendedArticles.length)];
  }, [recommendedArticles]);

  const handleRandomArticle = () => {
    if (randomArticle) {
      navigate(`/article/${randomArticle.slug}`);
      return;
    }
    navigate('/');
  };

  return (
    <Layout>
      <main className="mx-auto grid min-h-[70vh] max-w-5xl gap-10 px-6 py-20 lg:grid-cols-[minmax(0,1fr)_360px] lg:items-center">
        <section>
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-primary-100 bg-primary-50 px-3 py-1 text-xs font-black uppercase tracking-[0.24em] text-primary-700 dark:border-primary-900/50 dark:bg-primary-900/20 dark:text-primary-300">
            <Sparkles className="h-3.5 w-3.5" />
            404
          </div>
          <h1 className="max-w-2xl font-serif text-4xl font-black leading-tight text-neutral-950 dark:text-neutral-100 sm:text-5xl">
            {t('notFound.title')}
          </h1>
          <p className="mt-5 max-w-xl text-base font-medium leading-8 text-neutral-500 dark:text-neutral-400">
            {t('notFound.description')}
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <Link to="/" className="btn btn-primary inline-flex items-center gap-2">
              <Home className="h-4 w-4" />
              {t('common.backToHome')}
            </Link>
            <Button variant="secondary" onClick={() => navigate(-1)}>
              <ArrowLeft className="h-4 w-4" />
              {t('notFound.backPrevious')}
            </Button>
            <Button variant="secondary" onClick={handleRandomArticle}>
              <Shuffle className="h-4 w-4" />
              {t('notFound.randomArticle')}
            </Button>
          </div>
        </section>

        <aside className="rounded-2xl border border-neutral-200 bg-white p-5 shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
          <div className="mb-4 flex items-center justify-between gap-3">
            <div>
              <p className="text-xs font-black uppercase tracking-[0.24em] text-primary-600 dark:text-primary-400">Recommended</p>
              <h2 className="mt-1 text-lg font-black text-neutral-900 dark:text-neutral-100">{t('notFound.recommendedTitle')}</h2>
            </div>
            <Sparkles className="h-5 w-5 text-primary-500" />
          </div>
          <div className="space-y-3">
            {recommendedArticles.map((article) => (
              <Link
                key={article.id}
                to={`/article/${article.slug}`}
                className="block rounded-xl border border-neutral-100 bg-neutral-50 px-4 py-3 transition-colors hover:border-primary-200 hover:bg-primary-50 dark:border-neutral-800 dark:bg-neutral-950 dark:hover:border-primary-500/40 dark:hover:bg-primary-500/10"
              >
                <h3 className="line-clamp-2 text-sm font-bold leading-6 text-neutral-900 dark:text-neutral-100">{article.title}</h3>
                {article.summary && (
                  <p className="mt-1 line-clamp-2 text-xs leading-5 text-neutral-500 dark:text-neutral-400">{article.summary}</p>
                )}
              </Link>
            ))}
            {recommendedArticles.length === 0 && (
              <p className="rounded-xl bg-neutral-50 px-4 py-3 text-sm font-medium text-neutral-500 dark:bg-neutral-950 dark:text-neutral-400">
                {t('notFound.noRecommendations')}
              </p>
            )}
          </div>
        </aside>
      </main>
    </Layout>
  );
};
