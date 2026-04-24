import { Suspense, lazy } from 'react';
import { createBrowserRouter, Navigate } from 'react-router-dom';
import { AdminRoute, Loading, ProtectedRoute } from './components/common';

const Home = lazy(() => import('./pages/Home').then((module) => ({ default: module.Home })));
const ArticleDetail = lazy(() =>
  import('./pages/ArticleDetail').then((module) => ({ default: module.ArticleDetail }))
);
const AIChat = lazy(() => import('./pages/AIChat').then((module) => ({ default: module.AIChat })));
const Profile = lazy(() => import('./pages/Profile').then((module) => ({ default: module.Profile })));
const Login = lazy(() => import('./pages/Login').then((module) => ({ default: module.Login })));
const Register = lazy(() => import('./pages/Register').then((module) => ({ default: module.Register })));
const AdminLayout = lazy(() =>
  import('./components/admin/AdminLayout').then((module) => ({ default: module.AdminLayout }))
);
const ArticleList = lazy(() =>
  import('./views/admin/articles/ArticleList').then((module) => ({ default: module.ArticleList }))
);
const ArticleEditor = lazy(() =>
  import('./views/admin/articles/ArticleEditor').then((module) => ({ default: module.ArticleEditor }))
);
const CategoryList = lazy(() =>
  import('./views/admin/categories/CategoryList').then((module) => ({ default: module.CategoryList }))
);
const CommentList = lazy(() =>
  import('./views/admin/comments/CommentList').then((module) => ({ default: module.CommentList }))
);
const Dashboard = lazy(() =>
  import('./views/admin/Dashboard').then((module) => ({ default: module.Dashboard }))
);
const KnowledgeDocumentList = lazy(() =>
  import('./views/admin/knowledge-documents/KnowledgeDocumentList').then((module) => ({
    default: module.KnowledgeDocumentList,
  }))
);
const KnowledgeDocumentDetail = lazy(() =>
  import('./views/admin/knowledge-documents/KnowledgeDocumentDetail').then((module) => ({
    default: module.KnowledgeDocumentDetail,
  }))
);

const withSuspense = (element: React.ReactNode) => (
  <Suspense fallback={<Loading />}>{element}</Suspense>
);

export const router = createBrowserRouter([
  {
    path: '/',
    element: withSuspense(<Home />),
  },
  {
    path: '/article/:slug',
    element: withSuspense(<ArticleDetail />),
  },
  {
    path: '/ai-chat',
    element: withSuspense(
      <ProtectedRoute>
        <AIChat />
      </ProtectedRoute>
    ),
  },
  {
    path: '/profile',
    element: withSuspense(
      <ProtectedRoute>
        <Profile />
      </ProtectedRoute>
    ),
  },
  {
    path: '/login',
    element: withSuspense(<Login />),
  },
  {
    path: '/register',
    element: withSuspense(<Register />),
  },
  {
    path: '/admin',
    element: withSuspense(
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
    element: withSuspense(
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
