import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from '@/types';
import { authApi } from '@/api';

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isAdmin: boolean;

  // Actions
  setAuth: (user: User, token: string) => void;
  setUser: (user: User) => void;
  clearAuth: () => void;
  login: (email: string, password: string) => Promise<void>;
  register: (username: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  fetchCurrentUser: (options?: { silent?: boolean }) => Promise<boolean>;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      isAuthenticated: false,
      isAdmin: false,

      setAuth: (user, token) => {
        localStorage.setItem('access_token', token);
        set({
          user,
          token,
          isAuthenticated: true,
          isAdmin: user.role === 'admin',
        });
      },

      setUser: (user) => {
        set({
          user,
          isAuthenticated: true,
          isAdmin: user.role === 'admin',
        });
      },

      clearAuth: () => {
        localStorage.removeItem('access_token');
        set({
          user: null,
          token: null,
          isAuthenticated: false,
          isAdmin: false,
        });
      },

      login: async (email, password) => {
        const response = await authApi.login({ email, password });
        get().setAuth(response.user, response.access_token);
      },

      register: async (username, email, password) => {
        const response = await authApi.register({ username, email, password });
        get().setAuth(response.user, response.access_token);
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
        try {
          const response = await authApi.getCurrentUser({
            skipAuthRedirect: options?.silent,
          });
          get().setUser(response.user);
          return true;
        } catch (error) {
          get().clearAuth();
          return false;
        }
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        token: state.token,
        user: state.user,
        isAuthenticated: state.isAuthenticated,
        isAdmin: state.isAdmin,
      }),
    }
  )
);
