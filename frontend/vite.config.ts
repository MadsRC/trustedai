// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:9999",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ""),
      },
      "/sso": {
        target: "http://localhost:9999",
        changeOrigin: true,
      },
      // Proxy gRPC/Connect endpoints to backend
      "/llmgw.v1": {
        target: "http://localhost:9999",
        changeOrigin: true,
      },
    },
  },
});
