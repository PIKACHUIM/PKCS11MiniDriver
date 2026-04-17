import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5174,
    proxy: {
      // 开发时代理 API 请求到 server-card :1027
      '/api': {
        target: 'http://localhost:1027',
        changeOrigin: true,
      },
      '/crl': {
        target: 'http://localhost:1027',
        changeOrigin: true,
      },
      '/ocsp': {
        target: 'http://localhost:1027',
        changeOrigin: true,
      },
      '/acme': {
        target: 'http://localhost:1027',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom', 'react-router-dom'],
          antd: ['antd', '@ant-design/icons'],
        },
      },
    },
  },
})
