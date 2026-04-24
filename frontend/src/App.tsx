import { useEffect } from 'react';
import { RouterProvider } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { router } from './router';
import { useAuthStore, useUIStore } from './store';
import './styles/index.css';
import './styles/markdown.css';

// 创建 Query Client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,
      gcTime: 10 * 60 * 1000,
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

const Toast = () => {
  const { toast, hideToast } = useUIStore();

  useEffect(() => {
    if (toast.show) {
      const timer = setTimeout(() => {
        hideToast();
      }, 3000);
      return () => clearTimeout(timer);
    }
  }, [toast.show, hideToast]);

  if (!toast.show) return null;

  const bgColor = {
    success: 'bg-primary-500',
    error: 'bg-red-500',
    info: 'bg-blue-500',
  }[toast.type];

  return (
    <div className="fixed top-20 right-4 z-[100] animate-slide-up">
      <div className={`${bgColor} text-white px-6 py-3 rounded-lg shadow-lg`}>
        {toast.message}
      </div>
    </div>
  );
};

function App() {
  const { token, fetchCurrentUser } = useAuthStore();

  useEffect(() => {
    fetchCurrentUser({ silent: !token });
  }, [token, fetchCurrentUser]);

  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
      <Toast />
    </QueryClientProvider>
  );
}

export default App;
