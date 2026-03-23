import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 3000,
    proxy: {
      '/api/teams': {
        target: 'http://team-service:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/teams/, ''),
      },
      '/api/fields': {
        target: 'http://field-service:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/fields/, ''),
      },
      '/api/schedule': {
        target: 'http://schedule-service:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/schedule/, ''),
      },
    },
  },
})
