import { act, render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it } from 'vitest';
import { AdminRoute } from './AdminRoute';
import { ProtectedRoute } from './ProtectedRoute';
import { useAuthStore } from '@/store';

const renderProtectedRoutes = () =>
  render(
    <MemoryRouter initialEntries={['/private']}>
      <Routes>
        <Route path="/login" element={<div>login screen</div>} />
        <Route
          path="/private"
          element={
            <ProtectedRoute>
              <div>private screen</div>
            </ProtectedRoute>
          }
        />
      </Routes>
    </MemoryRouter>
  );

const renderAdminRoutes = () =>
  render(
    <MemoryRouter initialEntries={['/admin-only']}>
      <Routes>
        <Route path="/" element={<div>home screen</div>} />
        <Route path="/login" element={<div>login screen</div>} />
        <Route
          path="/admin-only"
          element={
            <AdminRoute>
              <div>admin screen</div>
            </AdminRoute>
          }
        />
      </Routes>
    </MemoryRouter>
  );

describe('auth route guards', () => {
  beforeEach(() => {
    useAuthStore.setState({
      user: null,
      isAuthenticated: false,
      isAdmin: false,
      authChecked: false,
    } as never);
  });

  it('waits for auth restoration before redirecting protected routes', () => {
    renderProtectedRoutes();

    expect(screen.queryByText('login screen')).not.toBeInTheDocument();
    expect(screen.queryByText('private screen')).not.toBeInTheDocument();

    act(() => {
      useAuthStore.setState({ authChecked: true } as never);
    });

    expect(screen.getByText('login screen')).toBeInTheDocument();
  });

  it('waits for auth restoration before redirecting admin routes', () => {
    renderAdminRoutes();

    expect(screen.queryByText('login screen')).not.toBeInTheDocument();
    expect(screen.queryByText('home screen')).not.toBeInTheDocument();
    expect(screen.queryByText('admin screen')).not.toBeInTheDocument();

    act(() => {
      useAuthStore.setState({ authChecked: true } as never);
    });

    expect(screen.getByText('login screen')).toBeInTheDocument();
  });
});
