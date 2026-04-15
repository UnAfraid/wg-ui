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
    rollupOptions: {
      output: {
        manualChunks: {
          "vendor-react": ["react", "react-dom", "react-router-dom"],
          "vendor-apollo": ["@apollo/client", "graphql", "graphql-ws"],
          "vendor-ui": [
            "@radix-ui/react-accordion",
            "@radix-ui/react-alert-dialog",
            "@radix-ui/react-avatar",
            "@radix-ui/react-checkbox",
            "@radix-ui/react-collapsible",
            "@radix-ui/react-context-menu",
            "@radix-ui/react-dialog",
            "@radix-ui/react-dropdown-menu",
            "@radix-ui/react-hover-card",
            "@radix-ui/react-label",
            "@radix-ui/react-menubar",
            "@radix-ui/react-navigation-menu",
            "@radix-ui/react-popover",
            "@radix-ui/react-progress",
            "@radix-ui/react-radio-group",
            "@radix-ui/react-scroll-area",
            "@radix-ui/react-select",
            "@radix-ui/react-separator",
            "@radix-ui/react-slider",
            "@radix-ui/react-slot",
            "@radix-ui/react-switch",
            "@radix-ui/react-tabs",
            "@radix-ui/react-toast",
            "@radix-ui/react-toggle",
            "@radix-ui/react-toggle-group",
            "@radix-ui/react-tooltip",
            "recharts",
            "sonner",
            "cmdk",
            "vaul",
            "react-day-picker",
            "react-hook-form",
            "react-resizable-panels",
            "embla-carousel-react",
            "react-qr-code",
            "input-otp",
          ],
          "vendor-utils": [
            "zod",
            "date-fns",
            "clsx",
            "tailwind-merge",
            "class-variance-authority",
            "lucide-react",
          ],
        },
      },
    },
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
