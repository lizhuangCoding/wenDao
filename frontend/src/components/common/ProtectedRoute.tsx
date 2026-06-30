import { Navigate } from 'react-router-dom';
import { useAuthStore } from '@/store';
import { Loading } from './Loading';

interface ProtectedRouteProps {
  children: React.ReactNode;
}

export const ProtectedRoute = ({ children }: ProtectedRouteProps) => {
  const { authChecked, isAuthenticated } = useAuthStore((state) => ({
    authChecked: state.authChecked,
    isAuthenticated: state.isAuthenticated,
  }));

  if (!authChecked) {
    return <Loading />;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};
