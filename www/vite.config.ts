import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";
import { writeFileSync, mkdirSync } from "fs";

function gitignorePlugin(): Plugin {
  return {
    name: "generate-gitignore",
    closeBundle() {
      const distDir = path.resolve(__dirname, "dist");
      mkdirSync(distDir, { recursive: true });
      writeFileSync(path.join(distDir, ".gitignore"), "*\n!.gitignore\n");
    },
  };
}

export default defineConfig({
  plugins: [react(), gitignorePlugin()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "."),
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/query": {
        target: "http://localhost:8080",
        changeOrigin: true,
        ws: true,
      },
    },
  },
});
