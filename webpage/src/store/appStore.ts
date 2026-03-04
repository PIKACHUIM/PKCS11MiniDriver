import { create } from 'zustand';
import type { SlotInfo } from '../types';
import { getSlots, getHealth } from '../api';

interface AppState {
  // 服务连接状态
  connected: boolean;
  serverVersion: string;
  // Slot 列表
  slots: SlotInfo[];
  slotsLoading: boolean;
  // 主题
  darkMode: boolean;
  // 操作
  checkConnection: () => Promise<void>;
  loadSlots: () => Promise<void>;
  toggleDarkMode: () => void;
}

export const useAppStore = create<AppState>((set) => ({
  connected: false,
  serverVersion: '',
  slots: [],
  slotsLoading: false,
  darkMode: localStorage.getItem('darkMode') === 'true',

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

  toggleDarkMode: () =>
    set((state) => {
      const next = !state.darkMode;
      localStorage.setItem('darkMode', String(next));
      return { darkMode: next };
    }),
}));
