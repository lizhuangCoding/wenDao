import { Link } from 'react-router-dom';
import type { ArticleListItem } from '@/types';
import { formatDate } from '@/utils';

interface ArticleCardProps {
  article: ArticleListItem;
}

export const ArticleCard = ({ article }: ArticleCardProps) => {
  return (
    <Link to={`/article/${article.slug}`} className="group block">
      <article className="flex flex-col h-full">
        <div className="relative aspect-[16/10] mb-8 overflow-hidden rounded-2xl bg-neutral-100 shadow-soft transition-all duration-500 group-hover:shadow-elevated group-hover:-translate-y-1">
          {article.cover_image ? (
            <img
              src={article.cover_image}
              alt={article.title}
              className="w-full h-full object-cover transition-transform duration-700 ease-out group-hover:scale-110"
            />
          ) : (
            <div className="w-full h-full flex items-center justify-center bg-gradient-to-br from-neutral-50 to-neutral-100 text-neutral-200">
              <svg xmlns="http://www.w3.org/2000/svg" className="h-16 w-16" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
            </div>
          )}

          <div className="absolute top-4 left-4">
            <div className="flex flex-wrap gap-2">
              <span className="px-3 py-1 bg-white/90 backdrop-blur-md text-[10px] font-black tracking-[0.2em] text-neutral-900 rounded-full shadow-sm uppercase dark:bg-neutral-800 dark:text-neutral-100">
                {article.category.name}
              </span>
              {article.source_type === 'knowledge_document' && (
                <span className="px-3 py-1 bg-primary-500/90 backdrop-blur-md text-[10px] font-black tracking-[0.2em] text-white rounded-full shadow-sm">
                  知识文档
                </span>
              )}
            </div>
          </div>

          {article.is_top && (
            <div className="absolute top-4 right-4">
              <div className="inline-flex items-center gap-1.5 rounded-full border border-amber-200/70 bg-gradient-to-r from-amber-50/95 via-white/95 to-orange-50/95 px-3 py-1.5 shadow-sm backdrop-blur-md transition-transform duration-300 group-hover:scale-105 dark:border-amber-400/20 dark:from-amber-100/95 dark:via-neutral-100/95 dark:to-orange-100/95">
                <span className="relative flex h-5 w-5 items-center justify-center rounded-full bg-gradient-to-br from-amber-400 to-orange-500 text-white shadow-sm">
                  <span className="absolute inset-0 rounded-full bg-white/30 animate-ping opacity-40"></span>
                  <svg xmlns="http://www.w3.org/2000/svg" className="relative h-3 w-3" viewBox="0 0 20 20" fill="currentColor">
                    <path d="M9.25 2.75a.75.75 0 011.5 0v1.12c0 .52.21 1.02.586 1.384l.91.91c.363.364.864.586 1.379.586h1.125a.75.75 0 010 1.5h-.934a1.25 1.25 0 00-1.01.514l-.568.781a2.75 2.75 0 01-2.225 1.135H8.99a2.75 2.75 0 01-2.225-1.135l-.568-.78a1.25 1.25 0 00-1.01-.515H4.25a.75.75 0 010-1.5h1.125c.515 0 1.016-.222 1.379-.586l.91-.91A1.95 1.95 0 008.25 3.87V2.75z" />
                    <path d="M8 12.75a.75.75 0 01.75.75V17a1.25 1.25 0 002.5 0v-3.5a.75.75 0 011.5 0V17a2.75 2.75 0 01-5.5 0v-3.5a.75.75 0 01.75-.75z" />
                  </svg>
                </span>
                <span className="text-[10px] font-black tracking-[0.24em] text-amber-700 uppercase dark:text-amber-900">
                  置顶
                </span>
              </div>
            </div>
          )}
        </div>

        <div className="flex flex-col flex-1">
          <header className="mb-4">
            <h2 className="text-2xl font-serif font-black text-neutral-900 dark:text-neutral-100 leading-tight group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors duration-300">
              {article.title}
            </h2>
          </header>

          <div className="mb-6 flex-1">
            <p className="text-neutral-500 dark:text-neutral-400 text-sm leading-relaxed line-clamp-3 font-medium italic">
              {article.summary || 'Click to explore the depths of this curated piece.'}
            </p>
          </div>

          <footer className="flex items-center justify-between pt-6 border-t border-neutral-100">
            <div className="flex items-center gap-3">
              <div className="w-6 h-6 rounded-full overflow-hidden border border-neutral-200">
                <img
                  src={article.author.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${article.author.username}`}
                  alt={article.author.username}
                />
              </div>
              <span className="text-[10px] font-bold tracking-widest text-neutral-400 uppercase">{article.author.username}</span>
            </div>

            <div className="flex items-center gap-4 text-neutral-300">
              <div className="flex items-center gap-1.5">
                <span className="text-[10px] font-bold tabular-nums">{article.view_count}</span>
                <svg xmlns="http://www.w3.org/2000/svg" className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                </svg>
              </div>
              <div className="w-1 h-1 bg-neutral-200 rounded-full"></div>
              <div className="flex items-center gap-1.5">
                <span className="text-[10px] font-bold tabular-nums">{article.comment_count}</span>
                <svg xmlns="http://www.w3.org/2000/svg" className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.084-.98l-4.813 1.255a1 1 0 01-1.248-1.248l1.255-4.813A9.862 9.862 0 013 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                </svg>
              </div>
              <div className="w-1 h-1 bg-neutral-200 rounded-full"></div>
              <span className="text-[10px] font-bold tracking-wider uppercase">{formatDate(article.created_at)}</span>
            </div>
          </footer>
        </div>
      </article>
    </Link>
  );
};
