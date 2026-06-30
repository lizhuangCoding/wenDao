import { create } from 'zustand';
import type { User } from '@/types';
import { authApi } from '@/api';
import {
  shouldApplyCurrentUserResult,
  shouldClearAuthAfterCurrentUserFailure,
} from '@/utils/pageBehavior';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isAdmin: boolean;
  authChecked: boolean;
  authVersion: number;

  // Actions
  setAuth: (user: User) => void;
  setUser: (user: User) => void;
  clearAuth: () => void;
  login: (email: string, password: string) => Promise<void>;
  register: (
    username: string,
    email: string,
    password: string,
    verificationCode: string
  ) => Promise<void>;
  logout: () => Promise<void>;
  fetchCurrentUser: (options?: { silent?: boolean }) => Promise<boolean>;
}

const buildAuthenticatedState = (user: User) => ({
  user,
  isAuthenticated: true,
  isAdmin: user.role === 'admin',
  authChecked: true,
});

export const useAuthStore = create<AuthState>()((set, get) => ({
  user: null,
  isAuthenticated: false,
  isAdmin: false,
  authChecked: false,
  authVersion: 0,

  setAuth: (user) => {
    set((state) => ({
      ...buildAuthenticatedState(user),
      authVersion: state.authVersion + 1,
    }));
  },

  setUser: (user) => {
    set((state) => ({
      ...buildAuthenticatedState(user),
      authVersion: state.authVersion + 1,
    }));
  },

  clearAuth: () => {
    set((state) => ({
      user: null,
      isAuthenticated: false,
      isAdmin: false,
      authChecked: true,
      authVersion: state.authVersion + 1,
    }));
  },

  login: async (email, password) => {
    const response = await authApi.login({ email, password });
    get().setAuth(response.user);
  },

  register: async (username, email, password, verificationCode) => {
    const response = await authApi.register({
      username,
      email,
      password,
      verification_code: verificationCode,
    });
    get().setAuth(response.user);
  },

  logout: async () => {
    try {
      await authApi.logout();
    } catch (error) {
      // 忽略错误，继续清除本地状态
    } finally {
      get().clearAuth();
    }
  },

  fetchCurrentUser: async (options) => {
    const requestVersion = get().authVersion;
    try {
      const response = await authApi.getCurrentUser({
        skipAuthRedirect: options?.silent,
      });
      if (!shouldApplyCurrentUserResult(requestVersion, get().authVersion)) {
        return false;
      }
      set((state) => ({
        ...buildAuthenticatedState(response.user),
        authVersion: state.authVersion + 1,
      }));
      return true;
    } catch (error) {
      if (shouldClearAuthAfterCurrentUserFailure(requestVersion, get().authVersion)) {
        set((state) => ({
          user: null,
          isAuthenticated: false,
          isAdmin: false,
          authChecked: true,
          authVersion: state.authVersion + 1,
        }));
      }
      return false;
    }
  },
}));
