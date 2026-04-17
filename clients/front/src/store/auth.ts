import { create } from 'zustand';
import type { AuthToken } from '../types';

interface AuthState {
  token: string | null;
  userUUID: string | null;
  username: string | null;
  role: 'admin' | 'user' | 'readonly' | null;
  isAuthenticated: boolean;

  setAuth: (auth: AuthToken) => void;
  clearAuth: () => void;
  /** 仅更新 token（刷新时使用） */
  setToken: (token: string) => void;
}

function loadFromStorage(): Pick<AuthState, 'token' | 'userUUID' | 'username' | 'role' | 'isAuthenticated'> {
  const token = localStorage.getItem('auth_token');
  const userUUID = localStorage.getItem('auth_user_uuid');
  const username = localStorage.getItem('auth_username');
  const role = localStorage.getItem('auth_role') as AuthState['role'];
  if (token && userUUID && username && role) {
    return { token, userUUID, username, role, isAuthenticated: true };
  }
  return { token: null, userUUID: null, username: null, role: null, isAuthenticated: false };
}

export const useAuthStore = create<AuthState>((set) => ({
  ...loadFromStorage(),

  setAuth: (auth: AuthToken) => {
    localStorage.setItem('auth_token', auth.token);
    localStorage.setItem('auth_user_uuid', auth.user_uuid);
    localStorage.setItem('auth_username', auth.username);
    localStorage.setItem('auth_role', auth.role);
    set({
      token: auth.token,
      userUUID: auth.user_uuid,
      username: auth.username,
      role: auth.role,
      isAuthenticated: true,
    });
  },

  clearAuth: () => {
    localStorage.removeItem('auth_token');
    localStorage.removeItem('auth_user_uuid');
    localStorage.removeItem('auth_username');
    localStorage.removeItem('auth_role');
    set({ token: null, userUUID: null, username: null, role: null, isAuthenticated: false });
  },

  setToken: (token: string) => {
    localStorage.setItem('auth_token', token);
    set({ token });
  },
}));
