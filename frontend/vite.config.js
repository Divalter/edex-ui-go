import { defineConfig } from "vite";

// Content Security Policy of the built UI, the equivalent of the one in the
// original ui.html. Inline handlers and styles are part of the original code
// ('unsafe-inline'), but nothing can be loaded from or sent to another origin,
// which limits what an injection could do.
const csp = [
    "default-src 'self'",
    "script-src 'self' 'unsafe-inline'",
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data: blob:",
    "media-src 'self' blob:",
    "font-src 'self' data:",
    "worker-src 'self' blob:",
    // ws: for the development server (edex-serve) only
    "connect-src 'self' ws://127.0.0.1:* ws://localhost:*",
    "object-src 'none'",
    "base-uri 'none'",
    "form-action 'none'"
].join("; ");

export default defineConfig({
    build: {
        target: "es2022",
        chunkSizeWarningLimit: 8000
    },
    plugins: [{
        name: "edex-csp",
        apply: "build",
        transformIndexHtml: html => html.replace(
            "<head>",
            `<head>\n        <meta http-equiv="Content-Security-Policy" content="${csp}">`
        )
    }]
});
