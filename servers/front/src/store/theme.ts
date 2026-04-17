import { create } from 'zustand';

export type ThemeMode = 'light' | 'dark' | 'system';

function resolveIsDark(mode: ThemeMode): boolean {
  if (mode === 'dark') return true;
  if (mode === 'light') return false;
  return window.matchMedia('(prefers-color-scheme: dark)').matches;
}

function loadThemeMode(): ThemeMode {
  const saved = localStorage.getItem('platform_themeMode') as ThemeMode | null;
  if (saved === 'light' || saved === 'dark' || saved === 'system') return saved;
  return 'light';
}

interface ThemeState {
  themeMode: ThemeMode;
  darkMode: boolean;
  setThemeMode: (mode: ThemeMode) => void;
  toggleDarkMode: () => void;
}

const initialMode = loadThemeMode();

export const useThemeStore = create<ThemeState>((set) => ({
  themeMode: initialMode,
  darkMode: resolveIsDark(initialMode),

  setThemeMode: (mode: ThemeMode) => {
    localStorage.setItem('platform_themeMode', mode);
    set({ themeMode: mode, darkMode: resolveIsDark(mode) });
  },

  toggleDarkMode: () =>
    set((state) => {
      const next = !state.darkMode;
      const mode: ThemeMode = next ? 'dark' : 'light';
      localStorage.setItem('platform_themeMode', mode);
      return { themeMode: mode, darkMode: next };
    }),
}));
