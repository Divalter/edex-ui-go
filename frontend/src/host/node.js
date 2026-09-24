/*
 * Replacements for the Node.js and Electron APIs used by the original
 * eDEX-UI renderer and classes, backed by the Go backend. They let the
 * ported code keep its original structure (require("fs"), electron.remote,
 * globalShortcut...).
 */

import {call, ipcRenderer, fileURL, isWails} from "./bridge.js";
import {makePath} from "./path.js";
import mimeDb from "mime-db";

import {Terminal as XTerminal} from "@xterm/xterm";
import {AttachAddon} from "@xterm/addon-attach";
import {FitAddon} from "@xterm/addon-fit";
import {WebglAddon} from "@xterm/addon-webgl";
import * as smoothieNs from "smoothie";
import * as howlerNs from "howler";
import color from "color";
import {nanoid} from "nanoid";
import prettyBytes from "pretty-bytes";

import {paceSmoothie} from "./frameclock.js";

import fileIcons from "../assets/file-icons.json";
import fileIconsMatch from "../assets/file-icons-match.js";
import grid from "../assets/grid.json";

// CommonJS packages: use the named exports when the bundler provides them.
const smoothie = smoothieNs.SmoothieChart ? smoothieNs : smoothieNs.default;
const howler = howlerNs.Howl ? howlerNs : howlerNs.default;
paceSmoothie(smoothie);

export function createNodeShims(boot) {
    const path = makePath(boot.platform === "win32");
    const runtime = window.runtime;

    /* fs */

    const statCache = new Map();
    const nodeError = (e, syscall, p) => {
        const err = new Error(`${e && e.message ? e.message : e}, ${syscall} '${p}'`);
        err.code = (String(err.message).match(/\b(E[A-Z]+)\b/) || [])[1];
        return err;
    };
    const makeStats = e => ({
        isDirectory: () => e.isDir,
        isFile: () => e.isFile,
        isSymbolicLink: () => e.isSymlink,
        size: e.size,
        mtime: new Date(e.mtime)
    });
    const lstatEntry = async p => {
        let e = statCache.get(p);
        statCache.delete(p);
        if (!e) e = await call("fs.lstat", p);
        if (e.error) throw new Error(`${e.error}: operation failed, lstat '${p}'`);
        return makeStats(e);
    };
    const readdir = async dir => {
        const entries = await call("fs.readdir", dir);
        entries.forEach(e => statCache.set(path.join(dir, e.name), e));
        return entries.map(e => e.name);
    };
    const callback = (promise, cb, syscall, p) => {
        promise.then(d => cb(null, d), e => cb(nodeError(e, syscall, p)));
    };
    const lastArg = args => (typeof args[args.length - 1] === "function" ? args[args.length - 1] : () => {});

    const fs = {
        readdir: (dir, ...rest) => callback(readdir(dir), lastArg(rest), "scandir", dir),
        lstat: (p, cb) => callback(lstatEntry(p), cb, "lstat", p),
        readFile: (p, ...rest) => callback(call("fs.readFile", p), lastArg(rest), "open", p),
        writeFile: (p, data, ...rest) => callback(call("fs.writeFile", p, String(data)), lastArg(rest), "open", p),
        // Writes are asynchronous: the renderer never reads the result back.
        writeFileSync: (p, data) => {
            call("fs.writeFile", p, String(data)).catch(e => console.error(`Cannot write ${p}`, e));
        },
        watch: (dir, listener) => {
            const watcher = {id: null, closed: false, close() {
                this.closed = true;
                if (this.id !== null) {
                    ipcRenderer.removeAllListeners(`fswatch-${this.id}`);
                    call("fs.unwatch", this.id).catch(() => {});
                }
            }};
            call("fs.watch", dir).then(id => {
                watcher.id = id;
                if (watcher.closed) {
                    watcher.close();
                    return;
                }
                ipcRenderer.on(`fswatch-${id}`, (e, type, name) => listener(type, name));
            }).catch(e => console.warn(`Cannot watch ${dir}`, e));
            return watcher;
        },
        promises: {
            readdir,
            readFile: p => call("fs.readFile", p),
            writeFile: (p, data) => call("fs.writeFile", p, String(data)),
            exists: p => call("fs.exists", p)
        }
    };

    /* os & process */

    const os = {
        platform: () => boot.platform,
        type: () => boot.osType,
        uptime: () => Date.now() / 1000 - boot.bootTime
    };

    const process = {
        platform: boot.platform,
        versions: {edex: boot.version, runtime: boot.runtime},
        argv: [],
        env: {}
    };

    /* Window (Electron's BrowserWindow) */

    let fullscreen = !!(boot.lastWindowState && boot.lastWindowState.useFullscreen !== false) && boot.settings.forceFullscreen !== false;
    let maximized = false;
    const leaveFullscreenHandlers = [];
    const refreshWindowState = () => {
        if (!runtime) return;
        runtime.WindowIsFullscreen().then(v => { fullscreen = v; }).catch(() => {});
        runtime.WindowIsMaximised().then(v => { maximized = v; }).catch(() => {});
    };
    refreshWindowState();
    window.addEventListener("resize", refreshWindowState);

    const electronWin = {
        minimize: () => runtime && runtime.WindowMinimise(),
        isFullScreen: () => fullscreen,
        isMaximized: () => maximized,
        unmaximize: () => runtime && runtime.WindowUnmaximise(),
        setFullScreen: v => {
            const was = fullscreen;
            fullscreen = !!v;
            if (runtime) {
                v ? runtime.WindowFullscreen() : runtime.WindowUnfullscreen();
            } else if (v) {
                document.documentElement.requestFullscreen().catch(() => {});
            } else if (document.fullscreenElement) {
                document.exitFullscreen();
            }
            if (was && !fullscreen) leaveFullscreenHandlers.forEach(fn => fn());
        },
        getSize: () => [window.outerWidth, window.outerHeight],
        setSize: (w, h) => runtime && runtime.WindowSetSize(w, h),
        on: (event, fn) => {
            if (event === "resize") window.addEventListener("resize", fn);
            if (event === "leave-full-screen") leaveFullscreenHandlers.push(fn);
        },
        webContents: {
            toggleDevTools: () => console.warn("Dev tools: use `wails dev` or a debug build (right click > Inspect Element)")
        }
    };

    /* Global shortcuts (Electron's globalShortcut) */

    // Registered accelerators are matched on keydown during the capture
    // phase, before xterm.js sees the event.
    const shortcuts = [];
    const parseAccelerator = acc => {
        const keys = acc.split("+");
        const key = keys.pop();
        const mods = keys.map(k => k.toLowerCase());
        return {
            ctrl: mods.includes("ctrl") || mods.includes("control") || mods.includes("commandorcontrol"),
            shift: mods.includes("shift"),
            alt: mods.includes("alt"),
            key: key.toLowerCase()
        };
    };
    const keyMatches = (key, e) => {
        if (/^[a-z]$/.test(key)) return e.code === "Key" + key.toUpperCase();
        if (/^[0-9]$/.test(key)) return e.code === "Digit" + key || e.code === "Numpad" + key;
        switch (key) {
            case "space": return e.code === "Space";
            case "tab": return e.key === "Tab";
            case "plus": return e.key === "+";
            case "enter": case "return": return e.key === "Enter";
            case "esc": case "escape": return e.key === "Escape";
            case "backspace": return e.key === "Backspace";
            case "delete": return e.key === "Delete";
        }
        return e.key.toLowerCase() === key || e.code.toLowerCase() === key;
    };
    window.addEventListener("keydown", e => {
        for (const s of shortcuts) {
            const a = s.accelerator;
            if (a.ctrl === e.ctrlKey && a.shift === e.shiftKey && a.alt === e.altKey && keyMatches(a.key, e)) {
                e.preventDefault();
                e.stopImmediatePropagation();
                s.fn();
                return;
            }
        }
    }, true);
    const globalShortcut = {
        register: (acc, fn) => shortcuts.push({accelerator: parseAccelerator(acc), fn}),
        unregisterAll: () => { shortcuts.length = 0; }
    };

    /* Electron */

    let displays = [{}];
    if (runtime) {
        runtime.ScreenGetAll().then(s => { if (s && s.length) displays = s; }).catch(() => {});
    }

    const openExternal = url => {
        if (runtime) runtime.BrowserOpenURL(url);
        else window.open(url, "_blank");
    };

    const remote = {
        app: {
            getVersion: () => boot.version,
            getPath: name => (name === "userData" ? boot.paths.settingsDir : ""),
            focus: () => {},
            relaunch: () => call("app.relaunch").catch(e => console.error(e)),
            quit: () => call("app.quit").catch(() => {})
        },
        process: {argv: []},
        screen: {getAllDisplays: () => displays},
        clipboard: {
            readText: () => (runtime ? runtime.ClipboardGetText() : navigator.clipboard.readText()),
            writeText: text => (runtime ? runtime.ClipboardSetText(text) : navigator.clipboard.writeText(text))
        },
        getCurrentWindow: () => electronWin,
        globalShortcut
    };

    const electron = {
        ipcRenderer,
        remote,
        shell: {
            openPath: p => call("shell.openPath", p).catch(e => console.error(e)),
            openExternal
        },
        webFrame: {setVisualZoomLevelLimits: () => {}}
    };

    /* mime-types */

    const extensions = {};
    for (const [type, entry] of Object.entries(mimeDb)) {
        (entry.extensions || []).forEach(ext => {
            if (!extensions[ext]) extensions[ext] = type;
        });
    }
    const mimeTypes = {
        lookup: p => extensions[String(p).split(".").pop().toLowerCase()] || false,
        charset: type => {
            if (!type) return false;
            const entry = mimeDb[type];
            if (entry && entry.charset) return entry.charset;
            return /^text\//i.test(type) ? "UTF-8" : false;
        }
    };

    /* require() */

    const noopAddon = class { activate() {} dispose() {} };
    const nanoidModule = Object.assign(() => nanoid(), {nanoid});
    const modules = {
        "path": path,
        "fs": fs,
        "os": os,
        "electron": electron,
        "@electron/remote": remote,
        "smoothie": smoothie,
        "howler": howler,
        "color": color,
        "nanoid": nanoidModule,
        "nanoid/non-secure": nanoidModule,
        "pretty-bytes": prettyBytes,
        "mime-types": mimeTypes,
        "xterm": {Terminal: XTerminal},
        "xterm-addon-attach": {AttachAddon},
        "xterm-addon-fit": {FitAddon},
        "xterm-addon-webgl": {WebglAddon},
        // Needs Node's font access; not available in a webview.
        "xterm-addon-ligatures": {LigaturesAddon: noopAddon},
        "assets/misc/file-icons-match.js": fileIconsMatch,
        "assets/icons/file-icons.json": fileIcons,
        "assets/misc/grid.json": grid,
        // Loaded by a <script> tag in index.html.
        "assets/vendor/encom-globe.js": {}
    };
    const require = name => {
        const key = String(name).replace(/\\/g, "/").replace(/^(\.\/)+/, "").replace(/^\/+/, "");
        if (key in modules) return modules[key];
        throw new Error(`Cannot find module '${name}'`);
    };

    return {path, fs, os, process, electron, remote, electronWin, require, fileURL, isWails};
}
