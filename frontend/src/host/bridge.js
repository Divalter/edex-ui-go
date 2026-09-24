/*
 * Transport between the UI and the Go backend.
 *
 * In the desktop app everything goes through the Wails IPC: RPC calls use
 * the Host.Call binding, backend messages arrive as Wails events, and local
 * files are served by the Wails asset handler. No network port is involved.
 *
 * When the UI is opened in a regular browser through the development server
 * (cmd/edex-serve), the same calls go over HTTP/WebSocket to the loopback
 * server, authenticated by the token passed in the page URL.
 */

const wailsHost = () => window.go && window.go.main && window.go.main.Host;

export const isWails = Boolean(wailsHost());

let devBase = "";
let devToken = "";

// channel -> array of {fn, once}
const listeners = new Map();
// tty port -> function(base64)
const ttyListeners = new Map();

function dispatch(channel, args) {
    const list = listeners.get(channel);
    if (!list) return;
    const event = {sender: ipcRenderer};
    for (const l of [...list]) {
        if (l.once) list.splice(list.indexOf(l), 1);
        try {
            l.fn(event, ...(args || []));
        } catch (e) {
            console.error(`ipc listener for ${channel} failed`, e);
        }
    }
}

/** Calls a backend RPC method (see internal/app/handlers.go). */
export async function call(name, ...args) {
    if (isWails) {
        return wailsHost().Call(name, args);
    }
    const res = await fetch(`${devBase}/rpc/${name}`, {
        method: "POST",
        headers: {"X-Edex-Token": devToken, "Content-Type": "application/json"},
        body: JSON.stringify(args)
    });
    const body = await res.json();
    if (body.error) throw new Error(body.error);
    return body.result;
}

/** Drop-in replacement for Electron's ipcRenderer. */
export const ipcRenderer = {
    on(channel, fn) {
        if (!listeners.has(channel)) listeners.set(channel, []);
        listeners.get(channel).push({fn, once: false});
        return ipcRenderer;
    },
    once(channel, fn) {
        if (!listeners.has(channel)) listeners.set(channel, []);
        listeners.get(channel).push({fn, once: true});
        return ipcRenderer;
    },
    removeAllListeners(channel) {
        listeners.delete(channel);
        return ipcRenderer;
    },
    send(channel, ...args) {
        // Replies addressed to the sender come back as the call result.
        call("ipc", channel, args).then(replies => {
            (replies || []).forEach(r => dispatch(r.channel, r.args));
        }).catch(e => console.error(`ipc ${channel} failed`, e));
    }
};

/** Connects the backend event stream. */
export async function connect() {
    if (isWails) {
        window.runtime.EventsOn("ipc", (channel, args) => dispatch(channel, args));
        window.runtime.EventsOn("tty", (port, b64) => {
            const fn = ttyListeners.get(port);
            if (fn) fn(b64);
        });
        return;
    }

    const params = new URLSearchParams(window.location.search);
    devToken = params.get("token") || "";
    devBase = window.location.origin;
    await new Promise((resolve, reject) => {
        const ws = new WebSocket(`${devBase.replace(/^http/, "ws")}/events?token=${devToken}`);
        ws.onopen = resolve;
        ws.onerror = () => reject(new Error("Cannot connect to the eDEX-UI-GO backend"));
        ws.onmessage = e => {
            const msg = JSON.parse(e.data);
            dispatch(msg.channel, msg.args);
        };
    });
}

/** URL of a local file, the replacement for file:// paths. */
export function fileURL(path) {
    const p = encodeURIComponent(path);
    return isWails ? `/edex-file?path=${p}` : `${devBase}/file?token=${devToken}&path=${p}`;
}

function base64ToBytes(b64) {
    const bin = atob(b64);
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
    return bytes;
}

function bytesToBase64(bytes) {
    let bin = "";
    for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
    return btoa(bin);
}

/*
 * WebSocket look-alike carrying a terminal over the Wails IPC, so that the
 * terminal class and xterm's AttachAddon work unchanged. Output batches are
 * acknowledged to let the backend apply flow control.
 */
class IpcTTYSocket extends EventTarget {
    constructor(port) {
        super();
        this.port = port;
        this.readyState = WebSocket.CONNECTING;
        this.binaryType = "arraybuffer";
        this.onopen = this.onclose = this.onerror = this.onmessage = null;
        this._queue = [];
        this._writing = false;
        // Output received before "open" is delivered right after it, as a
        // real WebSocket never emits messages before opening.
        this._early = [];

        ttyListeners.set(port, b64 => {
            const bytes = base64ToBytes(b64);
            if (this.readyState === WebSocket.OPEN) {
                this._message(bytes);
            } else {
                this._early.push(bytes);
            }
        });
        ipcRenderer.on(`tty_exit-${port}`, () => this._closed());

        call("tty.attach", port).then(() => {
            this.readyState = WebSocket.OPEN;
            this._fire(new Event("open"));
            this._early.forEach(bytes => this._message(bytes));
            this._early = [];
        }).catch(e => {
            this._fire(new ErrorEvent("error", {message: String(e)}));
            this._closed();
        });
    }

    _message(bytes) {
        this._fire(new MessageEvent("message", {data: bytes.buffer}));
        call("tty.ack", this.port, bytes.length).catch(() => {});
    }

    _fire(event) {
        this.dispatchEvent(event);
        const handler = this["on" + event.type];
        if (typeof handler === "function") handler.call(this, event);
    }

    send(data) {
        if (this.readyState !== WebSocket.OPEN) return;
        // Keep keystrokes in order: calls are sent one at a time and
        // consecutive text chunks are merged.
        const last = this._queue[this._queue.length - 1];
        if (typeof data === "string" && typeof last === "string") {
            this._queue[this._queue.length - 1] = last + data;
        } else {
            this._queue.push(typeof data === "string" ? data
                : data instanceof ArrayBuffer ? new Uint8Array(data)
                    : new Uint8Array(data.buffer, data.byteOffset, data.byteLength));
        }
        this._flush();
    }

    async _flush() {
        if (this._writing) return;
        this._writing = true;
        while (this._queue.length > 0) {
            const item = this._queue.shift();
            try {
                if (typeof item === "string") {
                    await call("tty.write", this.port, item);
                } else {
                    await call("tty.writeBinary", this.port, bytesToBase64(item));
                }
            } catch (e) {
                console.warn("tty write failed", e);
            }
        }
        this._writing = false;
    }

    close() {
        this._closed();
    }

    _closed() {
        if (this.readyState === WebSocket.CLOSED) return;
        this.readyState = WebSocket.CLOSED;
        ttyListeners.delete(this.port);
        ipcRenderer.removeAllListeners(`tty_exit-${this.port}`);
        this._fire(new CloseEvent("close", {code: 1000}));
    }
}

/** Opens the data stream of the terminal identified by port. */
export function openTTY(port) {
    if (isWails) return new IpcTTYSocket(port);
    return new WebSocket(`${devBase.replace(/^http/, "ws")}/tty/${port}?token=${devToken}`);
}
