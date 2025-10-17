// Lightweight API client with timeout + basic retry.
// Base URL is configurable via Vite env: import.meta.env.VITE_API_BASE
// Fallbacks to relative path (/api) if not set.

const DEFAULT_TIMEOUT_MS = 10000; // 10s
const DEFAULT_RETRIES = 1;        // simple retry once for transient errors

function sleep(ms) {
    return new Promise((r) => setTimeout(r, ms));
}

function withTimeout(promise, ms = DEFAULT_TIMEOUT_MS) {
    const t = new Promise((_, rej) =>
        setTimeout(() => rej(new Error("Request timed out")), ms)
    );
    return Promise.race([promise, t]);
}

export function getBaseUrl() {
    const env = import.meta?.env;
    const fromEnv = env?.VITE_API_BASE?.trim();
    if (fromEnv) return fromEnv.replace(/\/+$/, "");
    return ""; // same-origin, e.g. /api/...
}

async function doFetch(path, { method = "GET", headers = {}, body } = {}) {
    const base = getBaseUrl();
    const url =
        path.startsWith("http://") || path.startsWith("https://")
            ? path
            : `${base}${path.startsWith("/") ? "" : "/"}${path}`;

    const init = {
        method,
        headers: {
            ...(body ? { "Content-Type": "application/json" } : {}),
            ...headers
        },
        body: body ? JSON.stringify(body) : undefined,
        credentials: "include" // if you use cookies/sessions
    };

    const res = await withTimeout(fetch(url, init));
    if (!res.ok) {
        const text = await res.text().catch(() => "");
        const err = new Error(`HTTP ${res.status} ${res.statusText} — ${text}`);
        err.status = res.status;
        throw err;
    }
    const contentType = res.headers.get("content-type") || "";
    if (contentType.includes("application/json")) {
        return res.json();
    }
    return res.text();
}

/**
 * Safe request with tiny retry for transient errors (>=500 or network).
 */
export async function request(path, options = {}, { retries = DEFAULT_RETRIES } = {}) {
    let lastErr;
    for (let attempt = 0; attempt <= retries; attempt++) {
        try {
            return await doFetch(path, options);
        } catch (err) {
            lastErr = err;
            const status = err?.status ?? 0;
            const retriable = status >= 500 || status === 0; // network or server error
            if (attempt < retries && retriable) {
                await sleep(300 * (attempt + 1)); // backoff
                continue;
            }
            break;
        }
    }
    throw lastErr;
}

// Helpers
export const api = {
    get: (path, opts = {}) => request(path, { method: "GET", ...opts }),
    post: (path, body, opts = {}) => request(path, { method: "POST", body, ...opts }),
    put: (path, body, opts = {}) => request(path, { method: "PUT", body, ...opts }),
    del: (path, opts = {}) => request(path, { method: "DELETE", ...opts })
};
