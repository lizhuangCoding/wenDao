import { useAuthStore } from '@/store';

export const useAuth = () => {
  const { user, isAuthenticated, isAdmin, login, register, logout, fetchCurrentUser, setUser } =
    useAuthStore();

  return {
    user,
    isAuthenticated,
    isAdmin,
    login,
    register,
    logout,
    fetchCurrentUser,
    setUser,
  };
};
