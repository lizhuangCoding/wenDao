import { request } from './client';
import type { ArticleSearchResult, PaginatedResponse, PaginationParams } from '@/types';
import { toPaginationQuery } from './pagination';

type ArticleSearchParams = PaginationParams & {
  q?: string;
  category_id?: number;
  tag_id?: number;
};

export const searchApi = {
  searchArticles: (params: ArticleSearchParams) => {
    return request.get<PaginatedResponse<ArticleSearchResult>>('/search/articles', {
      params: toPaginationQuery(params),
    });
  },
};
