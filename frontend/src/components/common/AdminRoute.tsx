import { Navigate } from 'react-router-dom';
import { useAuthStore } from '@/store';
import { Loading } from './Loading';

interface AdminRouteProps {
  children: React.ReactNode;
}

export const AdminRoute = ({ children }: AdminRouteProps) => {
  const { authChecked, isAuthenticated, isAdmin } = useAuthStore();

  if (!authChecked) {
    return <Loading />;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (!isAdmin) {
    return <Navigate to="/" replace />;
  }

  return <>{children}</>;
};
