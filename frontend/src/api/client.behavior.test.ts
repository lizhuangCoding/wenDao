import type { AxiosAdapter, InternalAxiosRequestConfig, AxiosResponse } from 'axios';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const createSuccessResponse = <T,>(
  config: InternalAxiosRequestConfig,
  data: T
): AxiosResponse<{ code: number; message: string; data: T }> => ({
  config,
  data: {
    code: 0,
    message: 'ok',
    data,
  },
  headers: {},
  status: 200,
  statusText: 'OK',
});

const createUnauthorizedError = (config: InternalAxiosRequestConfig) => ({
  config,
  response: {
    config,
    data: {
      code: 401,
      message: 'unauthorized',
      data: null,
    },
    headers: {},
    status: 401,
    statusText: 'Unauthorized',
  },
});

describe('api client auth refresh behavior', () => {
  beforeEach(() => {
    vi.resetModules();
    localStorage.clear();
    document.cookie = 'csrf_token=test-csrf';
  });

  it('replays queued requests after a single token refresh', async () => {
    localStorage.setItem('access_token', 'stale-token');

    let protectedAttempts = 0;
    let refreshAttempts = 0;
    let releaseRefresh!: () => void;
    const refreshReleased = new Promise<void>((resolve) => {
      releaseRefresh = resolve;
    });

    const { default: apiClient, request } = await import('@/api/client');

    const adapter: AxiosAdapter = async (config) => {
      if (config.url === '/auth/refresh') {
        refreshAttempts += 1;
        await refreshReleased;
        return createSuccessResponse(config, {
          access_token: 'fresh-token',
          expires_in: 3600,
        });
      }

      if (config.url?.startsWith('/protected')) {
        protectedAttempts += 1;
        const authHeader = String(config.headers?.Authorization || '');
        if (authHeader === 'Bearer fresh-token') {
          return createSuccessResponse(config, {
            ok: true,
            authorization: authHeader,
            url: config.url,
          });
        }
        throw createUnauthorizedError(config);
      }

      throw new Error(`unexpected request: ${config.url}`);
    };

    apiClient.defaults.adapter = adapter;

    const firstRequest = request.get('/protected/one');
    const secondRequest = request.get('/protected/two');

    await Promise.resolve();
    await Promise.resolve();
    releaseRefresh();

    const [first, second] = await Promise.all([firstRequest, secondRequest]);

    expect(refreshAttempts).toBe(1);
    expect(protectedAttempts).toBe(4);
    expect(first).toMatchObject({
      ok: true,
      authorization: 'Bearer fresh-token',
      url: '/protected/one',
    });
    expect(second).toMatchObject({
      ok: true,
      authorization: 'Bearer fresh-token',
      url: '/protected/two',
    });
    expect(localStorage.getItem('access_token')).toBe('fresh-token');
  });
});
