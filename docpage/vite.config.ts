import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/PKCS11MiniDriver/',
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
  },
})
