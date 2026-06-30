import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.behavior.test.ts', 'src/**/*.behavior.test.tsx'],
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@/components': path.resolve(__dirname, './src/components'),
      '@/pages': path.resolve(__dirname, './src/pages'),
      '@/api': path.resolve(__dirname, './src/api'),
      '@/store': path.resolve(__dirname, './src/store'),
      '@/hooks': path.resolve(__dirname, './src/hooks'),
      '@/utils': path.resolve(__dirname, './src/utils'),
      '@/types': path.resolve(__dirname, './src/types'),
      '@/styles': path.resolve(__dirname, './src/styles'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8089',
        changeOrigin: true,
      },
      '/uploads': {
        target: 'http://localhost:8089',
        changeOrigin: true,
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules')) {
            if (
              id.includes('/three/') ||
              id.includes('/@react-three/fiber/')
            ) {
              return 'three-vendor'
            }
            if (
              id.includes('/recharts/') ||
              id.includes('/victory-vendor/') ||
              id.includes('/d3-')
            ) {
              return 'chart-vendor'
            }
            if (id.includes('/react/') || id.includes('/react-dom/') || id.includes('/react-router-dom/')) {
              return 'react-vendor'
            }
            if (id.includes('/zustand/') || id.includes('/@tanstack/react-query/') || id.includes('/framer-motion/')) {
              return 'ui-vendor'
            }
            if (
              id.includes('/rehype-raw/') ||
              id.includes('/hast-util-raw/') ||
              id.includes('/parse5/')
            ) {
              return 'markdown-admin-preview'
            }
            if (
              id.includes('/react-markdown/') ||
              id.includes('/remark-gfm/')
            ) {
              return 'markdown-viewer'
            }
            if (
              id.includes('/rehype-highlight/') ||
              id.includes('/highlight.js/') ||
              id.includes('/lowlight/') ||
              id.includes('/fault/')
            ) {
              return 'markdown-highlight'
            }
          }
        },
      },
    },
  },
})
