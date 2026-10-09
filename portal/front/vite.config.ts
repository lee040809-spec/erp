import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/portal/',
  server: {
    port: 4000,
    proxy: {
      '/api/portal': {
        target: 'http://localhost:38080',
        changeOrigin: true,
      },
    },
  },
})