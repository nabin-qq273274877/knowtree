import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// 开发模式：API 代理到本地 Go 服务（默认 127.0.0.1:6006）
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 6006,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:6006',
        changeOrigin: true,
      },
    },
  },
})
