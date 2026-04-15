import { create } from 'zustand';
import type { SlotInfo } from '../types';
import { getSlots, getHealth } from '../api';

export type ThemeMode = 'light' | 'dark' | 'system';

/** 根据 themeMode 和系统偏好计算实际是否暗黑 */
function resolveIsDark(mode: ThemeMode): boolean {
  if (mode === 'dark') return true;
  if (mode === 'light') return false;
  // system：跟随系统
  return window.matchMedia('(prefers-color-scheme: dark)').matches;
}

interface AppState {
  // 服务连接状态
  connected: boolean;
  serverVersion: string;
  // Slot 列表
  slots: SlotInfo[];
  slotsLoading: boolean;
  // 主题
  themeMode: ThemeMode;
  /** 计算属性：实际是否暗黑（供 ConfigProvider 使用） */
  darkMode: boolean;
  // 操作
  checkConnection: () => Promise<void>;
  loadSlots: () => Promise<void>;
  /** @deprecated 使用 setThemeMode 代替 */
  toggleDarkMode: () => void;
  setThemeMode: (mode: ThemeMode) => void;
}

function loadThemeMode(): ThemeMode {
  const saved = localStorage.getItem('themeMode') as ThemeMode | null;
  if (saved === 'light' || saved === 'dark' || saved === 'system') return saved;
  // 兼容旧版 darkMode 存储
  if (localStorage.getItem('darkMode') === 'true') return 'dark';
  return 'light';
}

const initialThemeMode = loadThemeMode();

export const useAppStore = create<AppState>((set) => ({
  connected: false,
  serverVersion: '',
  slots: [],
  slotsLoading: false,
  themeMode: initialThemeMode,
  darkMode: resolveIsDark(initialThemeMode),

  checkConnection: async () => {
    try {
      const health = await getHealth();
      set({ connected: true, serverVersion: health.version });
    } catch {
      set({ connected: false, serverVersion: '' });
    }
  },

  loadSlots: async () => {
    set({ slotsLoading: true });
    try {
      const slots = await getSlots();
      set({ slots: Array.isArray(slots) ? slots : [] });
    } catch {
      set({ slots: [] });
    } finally {
      set({ slotsLoading: false });
    }
  },

  setThemeMode: (mode: ThemeMode) => {
    localStorage.setItem('themeMode', mode);
    set({ themeMode: mode, darkMode: resolveIsDark(mode) });
  },

  toggleDarkMode: () =>
    set((state) => {
      const next = !state.darkMode;
      const mode: ThemeMode = next ? 'dark' : 'light';
      localStorage.setItem('themeMode', mode);
      return { themeMode: mode, darkMode: next };
    }),
}));
