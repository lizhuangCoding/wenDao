import { useTranslation } from 'react-i18next';

interface ErrorStateProps {
  title?: string;
  message?: string;
  onRetry?: () => void;
  className?: string;
}

export const ErrorState = ({
  title,
  message,
  onRetry,
  className = '',
}: ErrorStateProps) => {
  const { t } = useTranslation();
  return (
    <div className={`rounded-2xl border border-red-100 bg-red-50 px-6 py-8 dark:border-red-900/50 dark:bg-red-950/20 ${className}`}>
      <div className="text-sm font-bold text-red-700 dark:text-red-300">{title || t('common.loadingFailed')}</div>
      <div className="mt-2 text-sm text-red-500 dark:text-red-400">{message || t('common.loadingFailed')}</div>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="mt-5 rounded-xl bg-red-600 px-4 py-2 text-xs font-bold text-white transition-colors hover:bg-red-700"
        >
          {t('common.retry')}
        </button>
      )}
    </div>
  );
};
