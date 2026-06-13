import { useParams, Link, useNavigate } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { ArrowLeft, ArrowRight, Bookmark, Heart } from 'lucide-react';
import { articleApi } from '@/api';
import { Layout, ErrorState } from '@/components/common';
import { ArticleContent, ArticleDetailSkeleton, TableOfContents } from '@/components/article';
import { estimateReadingTime, extractHeadings } from '@/utils/markdown';
import { CommentList } from '@/components/comment';
import { formatDate } from '@/utils';
import { toAbsoluteSeoUrl } from '@/utils/seo';
import { useAuth } from '@/hooks';
import { useUIStore } from '@/store';
import { useEffect, useMemo } from 'react';
import { motion } from 'framer-motion';
import { Helmet } from 'react-helmet-async';
import type { Article, ArticleInteractionState } from '@/types';

type ArticleInteractionAction = 'like' | 'unlike' | 'favorite' | 'unfavorite';

export const ArticleDetail = () => {
  const { t } = useTranslation();
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const { isAdmin, isAuthenticated } = useAuth();

  const { data: article, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['article', slug],
    queryFn: () => articleApi.getArticleBySlug(slug!),
    enabled: !!slug,
  });

  useEffect(() => {
    window.scrollTo({ top: 0, left: 0, behavior: 'auto' });
  }, [slug]);

  const interactionQuery = useQuery({
    queryKey: ['article-interaction', article?.id],
    queryFn: () => articleApi.getArticleInteraction(article!.id),
    enabled: isAuthenticated && !!article?.id,
    staleTime: 60_000,
  });

  const updateArticleLikeCount = (delta: number) => {
    queryClient.setQueryData<Article | undefined>(['article', slug], (current) => {
      if (!current) return current;
      return {
        ...current,
        like_count: Math.max(0, current.like_count + delta),
      };
    });
  };

  const interactionMutation = useMutation({
    mutationFn: async (action: ArticleInteractionAction) => {
      if (!article) throw new Error('article missing');
      switch (action) {
      case 'like':
        return articleApi.likeArticle(article.id);
      case 'unlike':
        return articleApi.unlikeArticle(article.id);
      case 'favorite':
        return articleApi.favoriteArticle(article.id);
      case 'unfavorite':
        return articleApi.unfavoriteArticle(article.id);
      }
    },
    onSuccess: (state: ArticleInteractionState, action) => {
      if (!article) return;
      queryClient.setQueryData(['article-interaction', article.id], state);
      if (action === 'like') updateArticleLikeCount(1);
      if (action === 'unlike') updateArticleLikeCount(-1);
      queryClient.invalidateQueries({ queryKey: ['article', slug] });
    },
    onError: (mutationError: any) => {
      showToast(mutationError?.message || t('article.operationFailed'), 'error');
    },
  });

  const headings = useMemo(() => {
    if (!article?.content) return [];
    return extractHeadings(article.content);
  }, [article?.content]);

  if (isLoading) {
    return (
      <Layout>
        <Helmet>
          <title>{`${t('common.loading')} - 问道`}</title>
        </Helmet>
        <div className="max-w-display mx-auto px-6 sm:px-10 lg:px-12 py-20">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
          >
            <ArticleDetailSkeleton />
          </motion.div>
        </div>
      </Layout>
    );
  }

  if (isError) {
    return (
      <Layout>
        <div className="max-w-reading mx-auto px-6 py-16">
          <ErrorState
            message={(error as any)?.message || t('article.articleLoadFailed')}
            onRetry={() => refetch()}
          />
        </div>
      </Layout>
    );
  }

  if (!article) {
    return (
      <Layout>
        <div className="max-w-reading mx-auto px-6 py-32 text-center">
          <h1 className="text-4xl font-serif font-black text-neutral-900 dark:text-neutral-100 mb-4">{t('article.pieceNotFound')}</h1>
          <button type="button" onClick={() => navigate('/')} className="text-primary-600 dark:text-primary-400 font-bold tracking-widest uppercase text-xs">{t('article.returnGallery')}</button>
        </div>
      </Layout>
    );
  }

  const canonicalUrl = slug ? toAbsoluteSeoUrl(`/article/${slug}`) : '';
  const pageTitle = `${article.title} - 问道`;
  const pageDescription = article.summary || article.title;
  const ogImage = toAbsoluteSeoUrl(article.cover_image || '/favicon.svg');
  const publishDate = article.published_at || article.created_at;
  const modifiedDate = article.updated_at || publishDate;
  const readingTime = estimateReadingTime(article.content);
  const interactionState = interactionQuery.data || { liked: false, favorited: false };
  const isInteractionPending = interactionMutation.isPending || interactionQuery.isLoading;

  const requireLogin = () => {
    showToast(t('article.loginToInteract'), 'info');
    navigate('/login');
  };

  const handleLikeClick = () => {
    if (!isAuthenticated) {
      requireLogin();
      return;
    }
    interactionMutation.mutate(interactionState.liked ? 'unlike' : 'like');
  };

  const handleFavoriteClick = () => {
    if (!isAuthenticated) {
      requireLogin();
      return;
    }
    interactionMutation.mutate(interactionState.favorited ? 'unfavorite' : 'favorite');
  };

  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'Article',
    headline: article.title,
    description: pageDescription,
    image: ogImage,
    url: canonicalUrl,
    mainEntityOfPage: canonicalUrl,
    datePublished: publishDate,
    dateModified: modifiedDate,
    ...(article.author ? {
      author: {
        '@type': 'Person',
        name: article.author.username,
      },
    } : {}),
    publisher: {
      '@type': 'Organization',
      name: '问道',
    },
  };

  return (
    <Layout>
      <Helmet>
        <title>{pageTitle}</title>
        <meta name="description" content={pageDescription} />
        <meta property="og:title" content={pageTitle} />
        <meta property="og:description" content={pageDescription} />
        <meta property="og:type" content="article" />
        <meta property="og:image" content={ogImage} />
        <meta property="og:url" content={canonicalUrl} />
        {publishDate && <meta property="article:published_time" content={publishDate} />}
        {modifiedDate && <meta property="article:modified_time" content={modifiedDate} />}
        {article.author?.username && <meta property="article:author" content={article.author.username} />}
        <meta name="twitter:card" content={article.cover_image ? 'summary_large_image' : 'summary'} />
        <meta name="twitter:title" content={pageTitle} />
        <meta name="twitter:description" content={pageDescription} />
        <meta name="twitter:image" content={ogImage} />
        <link rel="canonical" href={canonicalUrl} />
        <script type="application/ld+json">
          {JSON.stringify(jsonLd)}
        </script>
      </Helmet>
      <div className="max-w-display mx-auto px-6 sm:px-10 lg:px-12 py-20">
        <div className="flex flex-col lg:flex-row justify-center gap-16">
          <aside className="hidden lg:fixed lg:left-[max(1.5rem,calc((100vw-1400px)/2+3rem))] lg:top-32 lg:z-20 lg:block lg:w-64 lg:max-h-[calc(100vh-8rem)] lg:overflow-y-auto lg:scrollbar-hide">
            <TableOfContents headings={headings} />
          </aside>

          <div className="hidden lg:block w-64 shrink-0" aria-hidden="true" />

          <motion.article
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.8 }}
            className="flex-1 min-w-0 max-w-reading"
          >
            <header className="mb-16">
              <div className="flex items-center gap-4 mb-8">
                <span className="text-[10px] font-black tracking-[0.3em] text-primary-600 dark:text-primary-400 uppercase bg-primary-50 dark:bg-primary-900/30 px-3 py-1 rounded-full">
                  {article.category.name}
                </span>
                <div className="w-8 h-px bg-neutral-200 dark:bg-neutral-700"></div>
                <span className="text-[10px] font-black tracking-[0.2em] text-neutral-400 dark:text-neutral-500 uppercase">
                  {formatDate(article.created_at)}
                </span>
                <div className="w-1 h-1 rounded-full bg-neutral-300 dark:bg-neutral-600" />
                <span className="text-[10px] font-black tracking-[0.2em] text-neutral-400 dark:text-neutral-500 uppercase">
                  {t('article.readingTime', { count: readingTime })}
                </span>
              </div>

              <h1 className="text-5xl md:text-7xl font-serif font-black text-neutral-900 dark:text-neutral-100 leading-[1.1] tracking-tight mb-10">
                {article.title}
              </h1>

              {article.summary && (
                <div className="pl-6 border-l-4 border-primary-500 mb-12">
                  <p className="text-xl text-neutral-500 dark:text-neutral-400 font-medium italic leading-relaxed">
                    {article.summary}
                  </p>
                </div>
              )}

              {article.cover_image && (
                <div className="w-full mb-16 rounded-[32px] overflow-hidden shadow-elevated">
                  <img
                    src={article.cover_image}
                    alt={article.title}
                    loading="eager"
                    decoding="async"
                    className="w-full h-auto object-cover max-h-[500px]"
                  />
                </div>
              )}

              {isAdmin && (
                <div className="flex gap-4 mb-8">
                  <Link to={`/admin/articles/edit/${article.id}`} className="bg-neutral-900 dark:bg-neutral-100 text-white dark:text-neutral-900 text-[10px] font-black tracking-widest px-6 py-3 rounded-full hover:bg-primary-600 dark:hover:bg-primary-500 transition-all uppercase">
                    {t('article.editPiece')}
                  </Link>
                </div>
              )}
            </header>

            <div className="article-reading-body">
              <ArticleContent content={article.content} />
            </div>

            <div className="article-interaction-actions mt-16 flex flex-wrap items-center justify-center gap-3 border-y border-neutral-100 py-8 dark:border-neutral-800">
              <button
                type="button"
                onClick={handleLikeClick}
                disabled={isInteractionPending}
                aria-pressed={interactionState.liked}
                className={`inline-flex items-center gap-2 rounded-full border px-4 py-2 text-sm font-bold transition-all disabled:cursor-not-allowed disabled:opacity-60 ${
                  interactionState.liked
                    ? 'border-rose-200 bg-rose-50 text-rose-600 shadow-sm dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-300'
                    : 'border-neutral-200 bg-white text-neutral-600 hover:border-rose-200 hover:text-rose-600 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:border-rose-500/40 dark:hover:text-rose-300'
                }`}
              >
                <Heart size={18} fill={interactionState.liked ? 'currentColor' : 'none'} />
                <span>{interactionState.liked ? t('article.liked') : t('article.like')}</span>
                <span className="tabular-nums text-neutral-400 dark:text-neutral-500">{article.like_count}</span>
              </button>
              <button
                type="button"
                onClick={handleFavoriteClick}
                disabled={isInteractionPending}
                aria-pressed={interactionState.favorited}
                className={`inline-flex items-center gap-2 rounded-full border px-4 py-2 text-sm font-bold transition-all disabled:cursor-not-allowed disabled:opacity-60 ${
                  interactionState.favorited
                    ? 'border-amber-200 bg-amber-50 text-amber-700 shadow-sm dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300'
                    : 'border-neutral-200 bg-white text-neutral-600 hover:border-amber-200 hover:text-amber-700 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:border-amber-500/40 dark:hover:text-amber-300'
                }`}
              >
                <Bookmark size={18} fill={interactionState.favorited ? 'currentColor' : 'none'} />
                <span>{interactionState.favorited ? t('article.favorited') : t('article.favorite')}</span>
              </button>
            </div>

            {article.collection_navigation && (
              <nav className="mt-12 rounded-xl border border-neutral-100 bg-white p-5 shadow-sm dark:border-neutral-800 dark:bg-neutral-900" aria-label="合集导航">
                <div className="mb-4 flex flex-wrap items-center justify-between gap-3 border-b border-neutral-100 pb-4 dark:border-neutral-800">
                  <div>
                    <p className="text-[10px] font-black uppercase tracking-[0.24em] text-primary-600 dark:text-primary-400">Collection</p>
                    <p className="mt-1 text-sm font-bold text-neutral-900 dark:text-neutral-100">
                      {article.collection_navigation.collection_name}
                    </p>
                  </div>
                  <span className="rounded-full bg-neutral-100 px-3 py-1 text-xs font-bold text-neutral-500 dark:bg-neutral-800 dark:text-neutral-400">
                    第 {article.collection_navigation.position} 篇 / 共 {article.collection_navigation.total} 篇
                  </span>
                </div>
                <div className="grid gap-3 sm:grid-cols-2">
                  {article.collection_navigation.previous ? (
                    <Link
                      to={`/article/${article.collection_navigation.previous.slug}`}
                      className="group flex min-h-24 flex-col justify-between rounded-lg border border-neutral-100 p-4 transition-all hover:border-primary-200 hover:bg-primary-50/60 dark:border-neutral-800 dark:hover:border-primary-500/30 dark:hover:bg-primary-500/10"
                    >
                      <span className="inline-flex items-center gap-2 text-xs font-black uppercase tracking-[0.18em] text-neutral-400 group-hover:text-primary-600 dark:text-neutral-500 dark:group-hover:text-primary-300">
                        <ArrowLeft className="h-4 w-4" />
                        上一篇
                      </span>
                      <span className="mt-3 line-clamp-2 text-sm font-bold leading-6 text-neutral-800 dark:text-neutral-200">
                        {article.collection_navigation.previous.title}
                      </span>
                    </Link>
                  ) : (
                    <div className="flex min-h-24 items-center rounded-lg border border-dashed border-neutral-100 p-4 text-sm font-medium text-neutral-400 dark:border-neutral-800 dark:text-neutral-600">
                      已经是合集第一篇
                    </div>
                  )}
                  {article.collection_navigation.next ? (
                    <Link
                      to={`/article/${article.collection_navigation.next.slug}`}
                      className="group flex min-h-24 flex-col justify-between rounded-lg border border-neutral-100 p-4 text-right transition-all hover:border-primary-200 hover:bg-primary-50/60 dark:border-neutral-800 dark:hover:border-primary-500/30 dark:hover:bg-primary-500/10"
                    >
                      <span className="inline-flex items-center justify-end gap-2 text-xs font-black uppercase tracking-[0.18em] text-neutral-400 group-hover:text-primary-600 dark:text-neutral-500 dark:group-hover:text-primary-300">
                        下一篇
                        <ArrowRight className="h-4 w-4" />
                      </span>
                      <span className="mt-3 line-clamp-2 text-sm font-bold leading-6 text-neutral-800 dark:text-neutral-200">
                        {article.collection_navigation.next.title}
                      </span>
                    </Link>
                  ) : (
                    <div className="flex min-h-24 items-center justify-end rounded-lg border border-dashed border-neutral-100 p-4 text-right text-sm font-medium text-neutral-400 dark:border-neutral-800 dark:text-neutral-600">
                      已经是合集最后一篇
                    </div>
                  )}
                </div>
              </nav>
            )}

            <div className="mt-24 pt-16 border-t border-neutral-100 dark:border-neutral-800">
              <CommentList articleId={article.id} totalCommentCount={article.comment_count} />
            </div>
          </motion.article>

          <div className="hidden xl:block w-64 shrink-0"></div>
        </div>
      </div>
    </Layout>
  );
};
