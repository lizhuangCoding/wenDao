import axios, { AxiosError, AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import type { ApiResponse, ApiError } from '@/types';

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
    // 从 localStorage 获取 Access Token
    const token = localStorage.getItem('access_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
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
let requests: (() => void)[] = [];

// 响应拦截器
apiClient.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
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
  async (error: AxiosError<ApiResponse>) => {
    const originalRequest = error.config as RequestConfig & { _retry?: boolean };

    // 401 未授权，且不是刷新 token 请求，且不是重试请求
    if (
      error.response?.status === 401 &&
      originalRequest &&
      !originalRequest.url?.includes('/auth/refresh') &&
      !originalRequest._retry
    ) {
      if (isRefreshing) {
        // 正在刷新 token，等待并重试
        return new Promise((resolve) => {
          requests.push(() => {
            resolve(apiClient(originalRequest));
          });
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
        requests.forEach((cb) => cb());
        requests = [];

        return apiClient(originalRequest);
      } catch (refreshError) {
        // 刷新失败，清除 token 并跳转登录
        localStorage.removeItem('access_token');
        if (!originalRequest.skipAuthRedirect) {
          window.location.href = '/login';
        }
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    // 处理 HTTP 错误
    const apiError: ApiError = {
      code: error.response?.status || -1,
      message: error.response?.data?.message || error.message || '网络请求失败',
    };

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
  get: <T = any>(url: string, config?: RequestConfig): Promise<T> => {
    return apiClient.get(url, config);
  },

  post: <T = any>(url: string, data?: any, config?: RequestConfig): Promise<T> => {
    return apiClient.post(url, data, config);
  },

  put: <T = any>(url: string, data?: any, config?: RequestConfig): Promise<T> => {
    return apiClient.put(url, data, config);
  },

  delete: <T = any>(url: string, config?: RequestConfig): Promise<T> => {
    return apiClient.delete(url, config);
  },

  patch: <T = any>(url: string, data?: any, config?: RequestConfig): Promise<T> => {
    return apiClient.patch(url, data, config);
  },
};

export default apiClient;
