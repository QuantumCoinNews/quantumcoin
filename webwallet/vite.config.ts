import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '127.0.0.1',
    proxy: {
      '^/api/(network/label$|wallet/balance/[1-9A-HJ-NP-Za-km-z]+$)': 'http://127.0.0.1:8082',
    },
  },
  build: { outDir: 'dist' }
})
