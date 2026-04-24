import { createBrowserRouter, Navigate } from 'react-router-dom';
import { Home } from './pages/Home';
import { ArticleDetail } from './pages/ArticleDetail';
import { AIChat } from './pages/AIChat';
import { Profile } from './pages/Profile';
import { Login } from './pages/Login';
import { Register } from './pages/Register';
import { useAuthStore } from './store';
import { AdminRoute } from './components/common';
import { AdminLayout } from './components/admin';
import { ArticleList } from './views/admin/articles/ArticleList';
import { ArticleEditor } from './views/admin/articles/ArticleEditor';
import { CategoryList } from './views/admin/categories/CategoryList';
import { CommentList } from './views/admin/comments/CommentList';
import { Dashboard } from './views/admin/Dashboard';
import { KnowledgeDocumentList } from './views/admin/knowledge-documents/KnowledgeDocumentList';
import { KnowledgeDocumentDetail } from './views/admin/knowledge-documents/KnowledgeDocumentDetail';

const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};

export const router = createBrowserRouter([
  {
    path: '/',
    element: <Home />,
  },
  {
    path: '/article/:slug',
    element: <ArticleDetail />,
  },
  {
    path: '/ai-chat',
    element: (
      <ProtectedRoute>
        <AIChat />
      </ProtectedRoute>
    ),
  },
  {
    path: '/profile',
    element: (
      <ProtectedRoute>
        <Profile />
      </ProtectedRoute>
    ),
  },
  {
    path: '/login',
    element: <Login />,
  },
  {
    path: '/register',
    element: <Register />,
  },
  {
    path: '/admin',
    element: (
      <AdminRoute>
        <AdminLayout />
      </AdminRoute>
    ),
    children: [
      { index: true, element: <Navigate to="/admin/stats" replace /> },
      { path: 'stats', element: <Dashboard /> },
      { path: 'articles', element: <ArticleList /> },
      { path: 'articles/new', element: <ArticleEditor /> },
      { path: 'articles/edit/:id', element: <ArticleEditor /> },
      { path: 'categories', element: <CategoryList /> },
      { path: 'comments', element: <CommentList /> },
      { path: 'knowledge-documents', element: <KnowledgeDocumentList /> },
      { path: 'knowledge-documents/:id', element: <KnowledgeDocumentDetail /> },
    ],
  },
  {
    path: '*',
    element: (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-4xl font-bold text-neutral-700 mb-4">404</h1>
          <p className="text-neutral-600 mb-6">页面不存在</p>
          <a href="/" className="btn btn-primary">
            返回首页
          </a>
        </div>
      </div>
    ),
  },
]);
