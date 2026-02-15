import { create } from 'zustand';

interface AuthState {
  isAuthenticated: boolean;
  authRequired: boolean;
  isLoading: boolean;
  error: string | null;
  checkAuth: () => Promise<void>;
  login: (username: string, password: string, turnstileToken: string) => Promise<void>;
  logout: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: false,
  authRequired: true,
  isLoading: true,
  error: null,

  checkAuth: async () => {
    try {
      const res = await fetch('/api/v1/auth/status', { credentials: 'include' });
      const json = await res.json();
      set({
        isAuthenticated: json.data.authenticated,
        authRequired: json.data.authRequired,
        isLoading: false,
      });
    } catch {
      set({ isAuthenticated: false, isLoading: false });
    }
  },

  login: async (username, password, turnstileToken) => {
    set({ error: null });
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ username, password, turnstileToken }),
    });
    const json = await res.json();
    if (!res.ok) {
      set({ error: json.error?.message || 'Login failed' });
      throw new Error(json.error?.message || 'Login failed');
    }
    set({ isAuthenticated: true, error: null });
  },

  logout: async () => {
    await fetch('/api/v1/auth/logout', {
      method: 'POST',
      credentials: 'include',
    });
    set({ isAuthenticated: false });
  },
}));
