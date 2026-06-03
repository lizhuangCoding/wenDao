export interface RouteChunkStorage {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
}

const ROUTE_CHUNK_RELOAD_PREFIX = 'wendao:route-chunk-reload';

const ROUTE_MODULE_ERROR_PATTERNS = [
  'Importing a module script failed',
  'Failed to fetch dynamically imported module',
  'error loading dynamically imported module',
  'Unable to preload CSS',
];

const getErrorMessage = (error: unknown) => {
  if (error instanceof Error) return error.message;
  if (typeof error === 'string') return error;
  if (error && typeof error === 'object' && 'message' in error) {
    return String((error as { message?: unknown }).message ?? '');
  }
  return '';
};

const normalizePathname = (pathname: string) => {
  const cleanedPathname = pathname.split('?')[0].split('#')[0];
  return cleanedPathname || '/';
};

export const getRouteChunkReloadKey = (pathname: string) => {
  return `${ROUTE_CHUNK_RELOAD_PREFIX}:${normalizePathname(pathname)}`;
};

export const isRouteModuleImportError = (error: unknown) => {
  const message = getErrorMessage(error);
  return ROUTE_MODULE_ERROR_PATTERNS.some((pattern) => message.includes(pattern));
};

export const shouldAttemptRouteChunkReload = (
  error: unknown,
  pathname: string,
  storage: RouteChunkStorage
) => {
  if (!isRouteModuleImportError(error)) return false;

  const key = getRouteChunkReloadKey(pathname);

  try {
    if (storage.getItem(key) === '1') return false;
    storage.setItem(key, '1');
    return true;
  } catch {
    return false;
  }
};

export const clearRouteChunkReloadAttempt = (pathname: string, storage: RouteChunkStorage) => {
  try {
    storage.removeItem(getRouteChunkReloadKey(pathname));
  } catch {
    // Storage can be unavailable in some mobile privacy modes.
  }
};
