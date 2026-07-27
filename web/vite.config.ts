import react from "@vitejs/plugin-react";
import { configDefaults, defineConfig } from "vitest/config";

const apiOrigin = process.env.FOLIOPATH_API_ORIGIN ?? "http://127.0.0.1:8080";

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": {
        target: apiOrigin,
      },
      "/health": {
        target: apiOrigin,
      },
    },
  },
  build: {
    outDir: "../internal/webassets/dist",
    emptyOutDir: true,
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    exclude: [...configDefaults.exclude, "tests/e2e/**"],
    css: true,
  },
});
