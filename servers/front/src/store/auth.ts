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
  setToken: (token: string) => void;
}

function loadFromStorage(): Pick<AuthState, 'token' | 'userUUID' | 'username' | 'role' | 'isAuthenticated'> {
  const token = localStorage.getItem('platform_token');
  const userUUID = localStorage.getItem('platform_user_uuid');
  const username = localStorage.getItem('platform_username') || '';
  const roleRaw = localStorage.getItem('platform_role');
  // role 只接受合法值，否则降级为 'user'
  const validRoles = ['admin', 'user', 'readonly'];
  const role = (validRoles.includes(roleRaw || '') ? roleRaw : 'user') as AuthState['role'];
  if (token && userUUID) {
    return { token, userUUID, username, role, isAuthenticated: true };
  }
  return { token: null, userUUID: null, username: null, role: null, isAuthenticated: false };
}

export const useAuthStore = create<AuthState>((set) => ({
  ...loadFromStorage(),

  setAuth: (auth: AuthToken) => {
    localStorage.setItem('platform_token', auth.token);
    localStorage.setItem('platform_user_uuid', auth.user_uuid);
    localStorage.setItem('platform_username', auth.username);
    localStorage.setItem('platform_role', auth.role);
    set({
      token: auth.token,
      userUUID: auth.user_uuid,
      username: auth.username,
      role: auth.role,
      isAuthenticated: true,
    });
  },

  clearAuth: () => {
    localStorage.removeItem('platform_token');
    localStorage.removeItem('platform_user_uuid');
    localStorage.removeItem('platform_username');
    localStorage.removeItem('platform_role');
    set({ token: null, userUUID: null, username: null, role: null, isAuthenticated: false });
  },

  setToken: (token: string) => {
    localStorage.setItem('platform_token', token);
    set({ token });
  },
}));
