import i18n from '@/i18n';

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

export const shouldFetchCurrentUser = (
  token: string | null | undefined,
  hasCheckedCookieAuth = false
) => Boolean(token) || !hasCheckedCookieAuth;

export const shouldApplyCurrentUserResult = (
  requestToken: string | null | undefined,
  currentToken: string | null | undefined
) => {
  if (!requestToken) {
    return !currentToken;
  }
  return currentToken === requestToken;
};

export const shouldClearAuthAfterCurrentUserFailure = shouldApplyCurrentUserResult;

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
  const locale = i18n.resolvedLanguage || i18n.language || 'zh';
  const isEnglish = locale.startsWith('en');

  if (isEdit) return isEnglish ? 'Update Article' : '更新文章';
  return status === 'published'
    ? isEnglish ? 'Publish Article' : '发布文章'
    : isEnglish ? 'Save Draft' : '保存草稿';
};
