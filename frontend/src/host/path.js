/*
 * Minimal implementation of Node's "path" module (the functions used by the
 * eDEX-UI renderer), for POSIX and Windows paths.
 */

function normalizeParts(parts, allowAboveRoot) {
    const out = [];
    for (const p of parts) {
        if (!p || p === ".") continue;
        if (p === "..") {
            if (out.length > 0 && out[out.length - 1] !== "..") out.pop();
            else if (allowAboveRoot) out.push("..");
        } else {
            out.push(p);
        }
    }
    return out;
}

export function makePath(win32) {
    const sep = win32 ? "\\" : "/";
    const splitRe = win32 ? /[\\/]+/ : /\/+/;

    // Returns [root, rest]: "/" on POSIX, "C:\" or "\\" on Windows.
    function splitRoot(p) {
        if (!win32) return p.startsWith("/") ? ["/", p.slice(1)] : ["", p];
        const drive = /^([a-zA-Z]:)([\\/]?)/.exec(p);
        if (drive) return [drive[1] + (drive[2] ? "\\" : ""), p.slice(drive[0].length)];
        if (/^[\\/]/.test(p)) return ["\\", p.slice(1)];
        return ["", p];
    }

    function normalize(p) {
        if (p === "") return ".";
        const [root, rest] = splitRoot(p);
        const parts = normalizeParts(rest.split(splitRe), root === "");
        const body = parts.join(sep);
        if (root) return root + body;
        return body || ".";
    }

    function isAbsolute(p) {
        const [root] = splitRoot(p);
        return win32 ? /[\\/]$/.test(root) : root === "/";
    }

    function join(...parts) {
        const filtered = parts.filter(p => typeof p === "string" && p !== "");
        if (filtered.length === 0) return ".";
        return normalize(filtered.join(sep));
    }

    function resolve(...parts) {
        let resolved = "";
        for (let i = parts.length - 1; i >= 0; i--) {
            const p = parts[i];
            if (!p) continue;
            resolved = resolved ? p + sep + resolved : p;
            if (isAbsolute(p)) break;
        }
        return normalize(resolved || sep);
    }

    function basename(p, ext) {
        const trimmed = p.replace(win32 ? /[\\/]+$/ : /\/+$/, "");
        let base = trimmed.split(splitRe).pop() || "";
        if (ext && base.endsWith(ext) && base !== ext) base = base.slice(0, -ext.length);
        return base;
    }

    function dirname(p) {
        const [root, rest] = splitRoot(p);
        const parts = rest.split(splitRe).filter(Boolean);
        parts.pop();
        if (parts.length === 0) return root || ".";
        return root + parts.join(sep);
    }

    function extname(p) {
        const base = basename(p);
        const i = base.lastIndexOf(".");
        return i <= 0 ? "" : base.slice(i);
    }

    return {sep, normalize, isAbsolute, join, resolve, basename, dirname, extname};
}
