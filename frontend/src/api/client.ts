import axios, { AxiosError, AxiosInstance, AxiosRequestConfig } from 'axios';
import type { ApiResponse, ApiError } from '@/types';
import { createAuthRefreshQueue } from '@/api/authRefreshQueue';
import { shouldAttemptTokenRefresh } from '@/utils/pageBehavior';
import { normalizeApiError } from '@/utils/apiError';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api';

export interface RequestConfig extends AxiosRequestConfig {
  skipAuthRedirect?: boolean;
}

export const getApiUrl = (path: string) => {
  const normalizedBase = API_BASE_URL.endsWith('/') ? API_BASE_URL : `${API_BASE_URL}/`;
  const normalizedPath = path.startsWith('/') ? path.slice(1) : path;
  const baseURL = /^https?:\/\//.test(normalizedBase)
    ? normalizedBase
    : new URL(normalizedBase, window.location.origin).toString();

  return new URL(normalizedPath, baseURL).toString();
};

const readCookie = (name: string) => {
  if (typeof document === 'undefined') return '';
  const prefix = `${name}=`;
  return document.cookie
    .split(';')
    .map((part) => part.trim())
    .find((part) => part.startsWith(prefix))
    ?.slice(prefix.length) || '';
};

const shouldAttachCSRFToken = (method?: string) => {
  const normalized = (method || 'get').toLowerCase();
  return !['get', 'head', 'options'].includes(normalized);
};

// 创建 axios 实例
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true, // 允许发送 Cookie
});

// 请求拦截器
apiClient.interceptors.request.use(
  (config) => {
    config.headers = config.headers || {};
    // 从 localStorage 获取 Access Token
    const token = localStorage.getItem('access_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    if (shouldAttachCSRFToken(config.method)) {
      const csrfToken = readCookie('csrf_token');
      if (csrfToken) {
        config.headers['X-CSRF-Token'] = csrfToken;
      }
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 标记是否正在刷新 token
let isRefreshing = false;
// 存储等待刷新 token 的请求
const refreshQueue = createAuthRefreshQueue();

// 响应拦截器
apiClient.interceptors.response.use(
  (response) => {
    const { data } = response;

    // 统一处理响应
    if (data.code === 0) {
      return data.data;
    }

    // 处理业务错误
    return Promise.reject({
      code: data.code,
      message: data.message,
    } as ApiError);
  },
  async (error: AxiosError<ApiResponse<unknown>>) => {
    const originalRequest = error.config as RequestConfig & { _retry?: boolean };

    // 401 未授权，且不是刷新 token 请求，且不是重试请求
    if (
      originalRequest &&
      shouldAttemptTokenRefresh({
        status: error.response?.status,
        url: originalRequest.url,
        alreadyRetried: originalRequest._retry,
        skipAuthRedirect: originalRequest.skipAuthRedirect,
      })
    ) {
      if (isRefreshing) {
        // 正在刷新 token，等待并重试
        return new Promise((resolve, reject) => {
          refreshQueue.add(() => {
            resolve(apiClient(originalRequest));
          }, reject);
        });
      }

      // 开始刷新 token
      originalRequest._retry = true;
      isRefreshing = true;

      try {
        // 调用刷新 token 接口（Cookie 会自动发送）
        const response = await apiClient.post<
          { access_token: string; expires_in: number },
          { access_token: string; expires_in: number }
        >('/auth/refresh', {});

        const { access_token } = response;

        // 更新 localStorage 中的 Access Token
        localStorage.setItem('access_token', access_token);

        // 重试原请求
        if (originalRequest.headers) {
          originalRequest.headers.Authorization = `Bearer ${access_token}`;
        }
        refreshQueue.resolveAll();

        return apiClient(originalRequest);
      } catch (refreshError) {
        // 刷新失败，清除 token 并跳转登录
        localStorage.removeItem('access_token');
        refreshQueue.rejectAll(refreshError);
        if (!originalRequest.skipAuthRedirect) {
          window.location.href = '/login';
        }
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    // 处理 HTTP 错误
    const apiError: ApiError = normalizeApiError(error);

    // 401 未授权，清除 token 并跳转登录
    if (error.response?.status === 401) {
      localStorage.removeItem('access_token');
      // 只有非刷新请求才跳转登录
      if (!originalRequest?.url?.includes('/auth/refresh') && !originalRequest?.skipAuthRedirect) {
        window.location.href = '/login';
      }
    }

    return Promise.reject(apiError);
  }
);

// 封装请求方法
export const request = {
  get: <T = unknown>(url: string, config?: RequestConfig): Promise<T> => {
    return apiClient.get(url, config);
  },

  post: <T = unknown, D = unknown>(url: string, data?: D, config?: RequestConfig): Promise<T> => {
    return apiClient.post(url, data, config);
  },

  put: <T = unknown, D = unknown>(url: string, data?: D, config?: RequestConfig): Promise<T> => {
    return apiClient.put(url, data, config);
  },

  delete: <T = unknown>(url: string, config?: RequestConfig): Promise<T> => {
    return apiClient.delete(url, config);
  },

  patch: <T = unknown, D = unknown>(url: string, data?: D, config?: RequestConfig): Promise<T> => {
    return apiClient.patch(url, data, config);
  },
};

export default apiClient;
