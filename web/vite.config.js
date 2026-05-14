import path from "node:path";
import { fileURLToPath } from "node:url";

import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

const currentDir = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig(({ mode }) => {
  const rootEnv = loadEnv(mode, path.resolve(currentDir, ".."), "VITE_");
  const hcaptchaSiteKey =
    process.env.VITE_HCAPTCHA_SITE_KEY || rootEnv.VITE_HCAPTCHA_SITE_KEY;

  return {
    plugins: [react(), tailwindcss()],
    define: hcaptchaSiteKey
      ? {
          "import.meta.env.VITE_HCAPTCHA_SITE_KEY": JSON.stringify(hcaptchaSiteKey),
        }
      : undefined,
    server: {
      port: 5174,
      proxy: {
        "/api": {
          target: "http://localhost:8088",
          changeOrigin: true,
        },
        "/health": {
          target: "http://localhost:8088",
          changeOrigin: true,
        },
      },
    },
  };
});
