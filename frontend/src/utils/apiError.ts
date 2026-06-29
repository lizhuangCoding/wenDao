import axios, { type AxiosError } from 'axios';
import type { ApiError, ApiResponse } from '@/types';

const DEFAULT_ERROR_CODE = -1;

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return typeof value === 'object' && value !== null;
};

const pickMessage = (value: unknown) => {
  return typeof value === 'string' && value.trim() ? value : undefined;
};

const buildAxiosErrorMessage = (
  error: AxiosError<ApiResponse<unknown>>
) => {
  const apiMessage = pickMessage(error.response?.data?.message);
  if (apiMessage) return apiMessage;

  if (error.code === 'ECONNABORTED') {
    return '请求超时：服务器响应时间过长，请稍后重试';
  }

  if (!error.response) {
    return '网络连接失败：请检查网络连接或稍后重试';
  }

  return `请求失败：服务器返回 HTTP ${error.response.status}`;
};

export const normalizeApiError = (
  error: unknown,
  fallbackMessage = '请求失败，请稍后重试'
): ApiError => {
  if (axios.isAxiosError<ApiResponse<unknown>>(error)) {
    return {
      code: error.response?.data?.code ?? error.response?.status ?? DEFAULT_ERROR_CODE,
      message: buildAxiosErrorMessage(error),
    };
  }

  if (isRecord(error)) {
    const message = pickMessage(error.message);
    const code = typeof error.code === 'number' ? error.code : DEFAULT_ERROR_CODE;
    if (message) {
      return { code, message };
    }
  }

  if (error instanceof Error) {
    return {
      code: DEFAULT_ERROR_CODE,
      message: pickMessage(error.message) ?? fallbackMessage,
    };
  }

  return {
    code: DEFAULT_ERROR_CODE,
    message: fallbackMessage,
  };
};

export const getApiErrorMessage = (error: unknown, fallbackMessage: string) => {
  return normalizeApiError(error, fallbackMessage).message;
};
