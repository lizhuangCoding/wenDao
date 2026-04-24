import { create } from 'zustand';

interface UIState {
  // Header 可见性
  isHeaderVisible: boolean;
  setHeaderVisible: (visible: boolean) => void;

  // 全局 loading
  isLoading: boolean;
  setLoading: (loading: boolean) => void;

  // Toast 通知
  toast: {
    show: boolean;
    message: string;
    type: 'success' | 'error' | 'info';
  };
  showToast: (message: string, type: 'success' | 'error' | 'info') => void;
  hideToast: () => void;
}

export const useUIStore = create<UIState>((set) => ({
  isHeaderVisible: true,
  setHeaderVisible: (visible) => set({ isHeaderVisible: visible }),

  isLoading: false,
  setLoading: (loading) => set({ isLoading: loading }),

  toast: {
    show: false,
    message: '',
    type: 'info',
  },
  showToast: (message, type) =>
    set({
      toast: { show: true, message, type },
    }),
  hideToast: () =>
    set({
      toast: { show: false, message: '', type: 'info' },
    }),
}));
