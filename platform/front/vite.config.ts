import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import {defineConfig, loadEnv} from 'vite';

export default defineConfig(({mode}) => {
  const env = loadEnv(mode, '.', '');
  return {
    base: '/platform/',
    plugins: [react(), tailwindcss()],
    define: {
      'process.env.GEMINI_API_KEY': JSON.stringify(env.GEMINI_API_KEY),
      'process.env.CLAUDE_API_KEY': JSON.stringify(env.CLAUDE_API_KEY),
      'process.env.CLAUDE_API_URL': JSON.stringify(env.CLAUDE_API_URL),
    },
    resolve: {
      alias: {
        '@': path.resolve(__dirname, '.'),
      },
    },
    server: {
      host: true,
      port: 3000,
      hmr: process.env.DISABLE_HMR !== 'true',
      proxy: {
        '/api': {
          target: 'http://127.0.0.1:28080',
          changeOrigin: true,
        },
      },
    },
    build: {
      target: 'esnext',
      minify: 'esbuild',
      sourcemap: false,
      chunkSizeWarningLimit: 600,
      rollupOptions: {
        output: {
          manualChunks: {
            // React 核心库
            'react-vendor': ['react', 'react-dom', 'react-router-dom'],
            // UI 图标库
            'lucide': ['lucide-react'],
            // 动画库
            'motion': ['motion'],
            // Google AI
            'google-ai': ['@google/genai'],
          },
        },
      },
    },
    esbuild: {
      // drop: ['console', 'debugger'], // 临时禁用以便调试
      drop: ['debugger'],
      legalComments: 'none',
    },
  };
});
