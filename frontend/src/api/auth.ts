import { request, getApiUrl } from './client';
import type {
  LoginRequest,
  RegisterRequest,
  AuthResponse,
  CurrentUserResponse,
} from '@/types';

// 用户认证 API
export const authApi = {
  // 登录
  login: (data: LoginRequest) => {
    return request.post<AuthResponse>('/auth/login', data);
  },

  // 注册
  register: (data: RegisterRequest) => {
    return request.post<AuthResponse>('/auth/register', data);
  },

  // 登出
  logout: () => {
    return request.post('/auth/logout');
  },

  // 获取当前用户信息
  getCurrentUser: (options?: { skipAuthRedirect?: boolean }) => {
    return request.get<CurrentUserResponse>('/auth/me', {
      skipAuthRedirect: options?.skipAuthRedirect,
    });
  },

  // GitHub OAuth 登录入口
  getGitHubLoginUrl: () => {
    return getApiUrl('/auth/github');
  },

  // 跳转到 GitHub OAuth 登录
  startGitHubLogin: () => {
    window.location.href = getApiUrl('/auth/github');
  },

  // 修改用户名
  updateUsername: (username: string) => {
    return request.put('/users/me/username', { username });
  },
};
