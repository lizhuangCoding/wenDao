import { beforeEach, describe, expect, it, vi } from 'vitest';

const authApiMock = vi.hoisted(() => ({
  login: vi.fn(),
  register: vi.fn(),
  logout: vi.fn(),
  getCurrentUser: vi.fn(),
}));

vi.mock('@/api', () => ({
  authApi: authApiMock,
}));

describe('auth store behavior', () => {
  beforeEach(async () => {
    vi.resetModules();
    vi.clearAllMocks();
    localStorage.clear();
  });

  it('logs in without persisting the access token to localStorage and marks auth as checked', async () => {
    authApiMock.login.mockResolvedValue({
      user: {
        id: 7,
        username: 'lizhuang',
        email: 'lizhuang@example.com',
        role: 'admin',
        status: 'active',
        created_at: '2026-06-30T00:00:00Z',
        updated_at: '2026-06-30T00:00:00Z',
      },
      access_token: 'fresh-access-token',
      refresh_token: 'fresh-refresh-token',
      expires_in: 3600,
    });

    const { useAuthStore } = await import('./authStore');

    await useAuthStore.getState().login('lizhuang@example.com', 'password');

    const state = useAuthStore.getState() as typeof useAuthStore.getState extends () => infer T ? T & { authChecked?: boolean } : never;
    expect(state.isAuthenticated).toBe(true);
    expect(state.isAdmin).toBe(true);
    expect(state.user?.username).toBe('lizhuang');
    expect(state.authChecked).toBe(true);
    expect(localStorage.getItem('access_token')).toBeNull();
  });
});
