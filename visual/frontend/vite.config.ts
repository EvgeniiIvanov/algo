// Конфиг Vite: dev-сервер фронта + прокси /api на Go-бэкенд.
// В prod-режиме собранный dist/ раздаётся отдельно (вне scope этого тикета).
import { defineConfig } from "vite";

const backend = process.env.ALGO_BACKEND ?? "http://localhost:8080";

export default defineConfig({
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      "/api": {
        target: backend,
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: "dist",
    sourcemap: true,
  },
  test: {
    environment: "jsdom",
    include: ["tests/**/*.test.ts"],
  },
});