import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// The dev server proxies /ws and /api to the backend container, mirroring
// the prod nginx setup so the browser sees a single origin in both
// environments. BACKEND_URL overrides the target for bare-host development.
const backend = process.env.BACKEND_URL ?? 'http://backend:8080';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/ws': { target: backend, ws: true },
      '/api': { target: backend },
    },
  },
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
  },
});
