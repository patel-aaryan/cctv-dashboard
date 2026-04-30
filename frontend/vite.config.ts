import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const apiTarget = env.VITE_API_URL || 'http://localhost:8080';

  return {
    plugins: [sveltekit()],
    server: {
      port: 5173,
      // Proxy /api and /media to the Go backend in dev so the SPA can use
      // relative URLs in code while still hitting the real API locally.
      proxy: {
        '/api':   { target: apiTarget, changeOrigin: false },
        '/media': { target: apiTarget, changeOrigin: false }
      }
    }
  };
});
