import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 8080,
    proxy: {
      '/api': 'http://localhost:8081',
      '/ws': {
        target: 'ws://localhost:8081',
        ws: true,
      },
    },
  },
})
