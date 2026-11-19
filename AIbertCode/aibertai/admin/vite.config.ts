import path from "path"
import react from "@vitejs/plugin-react"
import { defaultAllowedOrigins, defineConfig } from "vite"

const backend = process.env.AIBERTAI_DEV_BACKEND_URL ?? process.env.MONKEYAI_DEV_BACKEND_URL ?? "http://127.0.0.1:8080"
const apiBackend = process.env.AIBERTAI_DEV_API_URL ?? process.env.MONKEYAI_DEV_API_URL ?? backend

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  build: { target: "chrome86", cssTarget: "chrome86" },
  server: {
    allowedHosts: [".trycloudflare.com"],
    // Remote desktop preflights must reach the backend's endpoint-specific CORS policy.
    cors: { origin: defaultAllowedOrigins, preflightContinue: true },
    watch: {
      awaitWriteFinish: { stabilityThreshold: 200, pollInterval: 50 },
    },
    proxy: {
      "^/api/v1/endpoints/connect$": {
        target: apiBackend,
        changeOrigin: true,
        secure: false,
        ws: true,
      },
      "^/api/v1/endpoint-bridge": {
        target: apiBackend,
        changeOrigin: true,
        secure: false,
        ws: true,
      },
      "^/api/v1/endpoints/[^/]+/proxy/": {
        target: apiBackend,
        changeOrigin: true,
        secure: false,
        ws: true,
      },
      "/endpoints": { target: apiBackend, changeOrigin: true, secure: false },
      "/api": { target: apiBackend, changeOrigin: true, secure: false },
      "/oauth": backend,
      "/mcp": backend,
      "/v1": backend,
      "/.well-known/oauth-authorization-server": {
        target: apiBackend,
        changeOrigin: true,
        secure: false,
      },
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
})
