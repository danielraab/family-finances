import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import viteReact from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const config = defineConfig({
  resolve: { tsconfigPaths: true },
  plugins: [
    tailwindcss(),
    tanstackRouter({ target: "react", autoCodeSplitting: true }),
    viteReact(),
  ],
  server: {
    port: 3000,
    // The client ships no backend URL and calls the Go backend at relative
    // /api/... paths. In dev the two toolchains run on separate ports, so
    // proxy /api through to the backend (replaces the old next.config rewrite).
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: false,
      },
    },
  },
  build: {
    // Keep the historical output path: the Go backend embeds `static/out`,
    // populated from `frontend/out/` by the Docker build.
    outDir: "out",
  },
});

export default config;
