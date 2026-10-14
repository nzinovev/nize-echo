import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// In development the Go server proxies /admin/ here, so the admin and the
// API share one origin. The build is embedded into the Go binary later.
export default defineConfig({
  base: "/admin/",
  plugins: [react()],
  server: {
    port: 5173,
    strictPort: true,
  },
});
