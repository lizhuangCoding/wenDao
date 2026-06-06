const trimTrailingSlashes = (value: string) => value.trim().replace(/\/+$/, '');

export const getPublicSiteUrl = () => {
  const configuredSiteUrl = trimTrailingSlashes(import.meta.env.VITE_SITE_URL || '');
  if (configuredSiteUrl) return configuredSiteUrl;

  if (typeof window !== 'undefined' && window.location?.origin) {
    return trimTrailingSlashes(window.location.origin);
  }

  return '';
};

export const toAbsoluteSeoUrl = (value: string) => {
  const url = value.trim();
  if (!url) return '';
  if (/^https?:\/\//i.test(url)) return url;

  const siteUrl = getPublicSiteUrl();
  if (!siteUrl) return url;

  try {
    return new URL(url, `${siteUrl}/`).toString();
  } catch {
    return url;
  }
};
