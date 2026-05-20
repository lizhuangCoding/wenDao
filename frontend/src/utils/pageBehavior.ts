type ArticleStatus = 'draft' | 'published';

interface TokenRefreshDecision {
  status?: number;
  url?: string;
  alreadyRetried?: boolean;
  skipAuthRedirect?: boolean;
}

interface ArticlePrimaryActionOptions {
  isEdit: boolean;
  status: ArticleStatus;
}

export const shouldFetchCurrentUser = (token: string | null | undefined) => Boolean(token);

export const shouldAttemptTokenRefresh = ({
  status,
  url = '',
  alreadyRetried = false,
  skipAuthRedirect = false,
}: TokenRefreshDecision) =>
  status === 401 && !skipAuthRedirect && !alreadyRetried && !url.includes('/auth/refresh');

export const getArticlePrimaryActionLabel = ({
  isEdit,
  status,
}: ArticlePrimaryActionOptions) => {
  if (isEdit) return '更新文章';
  return status === 'published' ? '发布文章' : '保存草稿';
};
