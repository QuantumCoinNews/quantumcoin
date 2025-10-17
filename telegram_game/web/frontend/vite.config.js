import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
    const env = loadEnv(mode, process.cwd(), "");
    // Değiştirilebilir hedef: .env.local dosyasında VITE_PROXY_TARGET tanımlayabilirsin.
    const proxyTarget = env.VITE_PROXY_TARGET || "http://localhost:8080";

    return {
        plugins: [react()],
        server: {
            host: true,
            port: 5173,
            proxy: {
                // Frontend'ten /api/* çağrılarını backend'e yönlendir
                "/api": {
                    target: proxyTarget,
                    changeOrigin: true
                    // ws: true,          // websocket gerekiyorsa aç
                    // rewrite: p => p    // gerekliyse yeniden yazım ekle
                }
            }
        },
        preview: {
            host: true,
            port: 5173,
            open: true
        }
    };
});
