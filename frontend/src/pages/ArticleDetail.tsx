import { useParams, Link, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { articleApi } from '@/api';
import { Layout, Loading } from '@/components/common';
import { ArticleContent, TableOfContents } from '@/components/article';
import { extractHeadings } from '@/utils/markdown';
import { CommentList } from '@/components/comment';
import { formatDate } from '@/utils';
import { useAuth } from '@/hooks';
import { useMemo } from 'react';
import { motion } from 'framer-motion';

export const ArticleDetail = () => {
  const { t } = useTranslation();
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const { isAdmin } = useAuth();

  const { data: article, isLoading } = useQuery({
    queryKey: ['article', slug],
    queryFn: () => articleApi.getArticleBySlug(slug!),
    enabled: !!slug,
  });

  const headings = useMemo(() => {
    if (!article?.content) return [];
    return extractHeadings(article.content);
  }, [article?.content]);

  if (isLoading) return <Layout><Loading /></Layout>;

  if (!article) {
    return (
      <Layout>
        <div className="max-w-reading mx-auto px-6 py-32 text-center">
          <h1 className="text-4xl font-serif font-black text-neutral-900 dark:text-neutral-100 mb-4">{t('article.pieceNotFound')}</h1>
          <button onClick={() => navigate('/')} className="text-primary-600 dark:text-primary-400 font-bold tracking-widest uppercase text-xs">{t('article.returnGallery')}</button>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="max-w-display mx-auto px-6 sm:px-10 lg:px-12 py-20">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8 }}
          className="flex flex-col lg:flex-row justify-center gap-16"
        >
          <aside className="hidden lg:block w-64 shrink-0">
            <div className="sticky top-32">
              <TableOfContents headings={headings} />

              <div className="mt-12 pt-12 border-t border-neutral-100 dark:border-neutral-800">
                <h4 className="text-[10px] font-black tracking-[0.2em] text-neutral-400 dark:text-neutral-500 uppercase mb-6">{t('article.sharedBy')}</h4>
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full overflow-hidden border border-neutral-200 dark:border-neutral-700 shadow-sm">
                    <img src={article.author.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${article.author.username}`} alt={article.author.username} />
                  </div>
                  <div>
                    <p className="text-sm font-bold text-neutral-900 dark:text-neutral-100">{article.author.username}</p>
                    <p className="text-[10px] text-neutral-400 dark:text-neutral-500 font-bold uppercase tracking-tighter">{t('article.contributor')}</p>
                  </div>
                </div>
              </div>
            </div>
          </aside>

          <article className="flex-1 min-w-0 max-w-reading">
            <header className="mb-16">
              <div className="flex items-center gap-4 mb-8">
                <span className="text-[10px] font-black tracking-[0.3em] text-primary-600 dark:text-primary-400 uppercase bg-primary-50 dark:bg-primary-900/30 px-3 py-1 rounded-full">
                  {article.category.name}
                </span>
                <div className="w-8 h-px bg-neutral-200 dark:bg-neutral-700"></div>
                <span className="text-[10px] font-black tracking-[0.2em] text-neutral-400 dark:text-neutral-500 uppercase">
                  {formatDate(article.created_at)}
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

            <div className="prose-refined">
              <ArticleContent content={article.content} />
            </div>

            <div className="mt-24 pt-16 border-t border-neutral-100 dark:border-neutral-800">
              <CommentList articleId={article.id} totalCommentCount={article.comment_count} />
            </div>
          </article>

          <div className="hidden xl:block w-64 shrink-0"></div>
        </motion.div>
      </div>
    </Layout>
  );
};
