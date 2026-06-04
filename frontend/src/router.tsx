import { lazy } from 'react';
import { createBrowserRouter, Navigate } from 'react-router-dom';
import {
  AdminRoute,
  NotFoundPage,
  ProtectedRoute,
  RouteErrorFallback,
  RouteSuspenseBoundary,
} from './components/common';

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
  <RouteSuspenseBoundary>{element}</RouteSuspenseBoundary>
);

export const router = createBrowserRouter([
  {
    path: '/',
    element: withSuspense(<Home />),
    errorElement: <RouteErrorFallback />,
  },
  {
    path: '/article/:slug',
    element: withSuspense(<ArticleDetail />),
    errorElement: <RouteErrorFallback />,
  },
  {
    path: '/ai-chat',
    element: withSuspense(
      <ProtectedRoute>
        <AIChat />
      </ProtectedRoute>
    ),
    errorElement: <RouteErrorFallback />,
  },
  {
    path: '/profile',
    element: withSuspense(
      <ProtectedRoute>
        <Profile />
      </ProtectedRoute>
    ),
    errorElement: <RouteErrorFallback />,
  },
  {
    path: '/login',
    element: withSuspense(<Login />),
    errorElement: <RouteErrorFallback />,
  },
  {
    path: '/register',
    element: withSuspense(<Register />),
    errorElement: <RouteErrorFallback />,
  },
  {
    path: '/admin',
    element: withSuspense(
      <AdminRoute>
        <AdminLayout />
      </AdminRoute>
    ),
    errorElement: <RouteErrorFallback />,
    children: [
      { index: true, element: <Navigate to="/admin/stats" replace /> },
      { path: 'stats', element: withSuspense(<Dashboard />) },
      { path: 'articles', element: withSuspense(<ArticleList />) },
      { path: 'articles/new', element: withSuspense(<ArticleEditor />) },
      { path: 'articles/edit/:id', element: withSuspense(<ArticleEditor />) },
      { path: 'categories', element: withSuspense(<CategoryList />) },
      { path: 'comments', element: withSuspense(<CommentList />) },
      { path: 'knowledge-documents', element: withSuspense(<KnowledgeDocumentList />) },
      { path: 'knowledge-documents/:id', element: withSuspense(<KnowledgeDocumentDetail />) },
    ],
  },
  {
    path: '*',
    element: withSuspense(<NotFoundPage />),
  },
]);
