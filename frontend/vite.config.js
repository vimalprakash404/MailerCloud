import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(({ mode }) => {
  // Load env file from parent directory or local frontend directory
  const env = loadEnv(mode, process.cwd(), '')
  const port = parseInt(env.VITE_PORT) || 3000
  const apiTarget = env.VITE_API_URL || 'http://localhost:8080'

  return {
    plugins: [vue()],
    server: {
      port: port,
      proxy: {
        '/events': {
          target: apiTarget,
          changeOrigin: true,
        },
        '/campaigns': {
          target: apiTarget,
          changeOrigin: true,
          ws: true,
        },
      },
    },
  }
})

