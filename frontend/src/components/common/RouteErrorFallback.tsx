import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useLocation, useRouteError } from 'react-router-dom';
import { isRouteModuleImportError, shouldAttemptRouteChunkReload } from '@/utils/routeChunkRecovery';

const getErrorMessage = (error: unknown, fallbackMessage: string) => {
  if (error instanceof Error) return error.message;
  if (typeof error === 'string') return error;
  if (error && typeof error === 'object' && 'message' in error) {
    return String((error as { message?: unknown }).message ?? fallbackMessage);
  }
  return fallbackMessage;
};

export const RouteErrorFallback = () => {
  const { t } = useTranslation();
  const error = useRouteError();
  const location = useLocation();
  const isRouteImportError = isRouteModuleImportError(error);

  useEffect(() => {
    if (!isRouteImportError || typeof window === 'undefined') return;

    if (shouldAttemptRouteChunkReload(error, location.pathname, window.sessionStorage)) {
      window.location.reload();
    }
  }, [error, isRouteImportError, location.pathname]);

  return (
    <div className="min-h-screen bg-neutral-50 px-6 py-24 text-neutral-900 dark:bg-neutral-950 dark:text-neutral-100">
      <div className="mx-auto flex min-h-[60vh] max-w-lg flex-col justify-center">
        <p className="text-xs font-black uppercase tracking-[0.28em] text-primary-600 dark:text-primary-400">
          {t('common.loadingFailed')}
        </p>
        <h1 className="mt-4 text-3xl font-serif font-black">{t('common.loadingFailed')}</h1>
        <p className="mt-4 text-sm leading-7 text-neutral-600 dark:text-neutral-400">
          {isRouteImportError
            ? t('routeError.chunkReload')
            : getErrorMessage(error, t('routeError.pageLoadFailed'))}
        </p>
        <div className="mt-8 flex flex-wrap gap-3">
          <button
            type="button"
            onClick={() => window.location.reload()}
            className="rounded-full bg-neutral-900 px-5 py-2.5 text-xs font-black tracking-widest text-white transition-colors hover:bg-primary-600 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-primary-400 dark:hover:text-white"
          >
            {t('routeError.reload')}
          </button>
          <Link
            to="/"
            className="rounded-full border border-neutral-200 px-5 py-2.5 text-xs font-black tracking-widest text-neutral-700 transition-colors hover:bg-white dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-900"
          >
            {t('common.backToHome')}
          </Link>
        </div>
      </div>
    </div>
  );
};
