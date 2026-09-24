/*
 * eDEX-UI-GO frontend entry point.
 *
 * Loads the original eDEX-UI stylesheets (in the order of the original
 * ui.html), connects to the Go backend, installs the Node/Electron
 * replacements the original code expects, then runs the ported renderer.
 */

// Dependency css
import "augmented-ui/augmented.css";
import "@xterm/xterm/css/xterm.css";
// Main css
import "./css/main.css";
import "./css/modal.css";
import "./css/boot_screen.css";
import "./css/media_player.css";
// Main modules css
import "./css/main_shell.css";
import "./css/filesystem.css";
import "./css/keyboard.css";
// Secondary modules css
import "./css/mod_column.css";
import "./css/mod_clock.css";
import "./css/mod_sysinfo.css";
import "./css/mod_hardwareInspector.css";
import "./css/mod_cpuinfo.css";
import "./css/mod_netstat.css";
import "./css/mod_conninfo.css";
import "./css/mod_globe.css";
import "./css/mod_ramwatcher.css";
import "./css/mod_toplist.css";
import "./css/mod_fuzzyFinder.css";
import "./css/mod_processlist.css";
// Extra css
import "./css/extra_ratios.css";
import "./css/edex_go.css";

import {call, connect, fileURL, ipcRenderer, isWails, openTTY} from "./host/bridge.js";
import {createNodeShims} from "./host/node.js";
import {initDropdown} from "./host/dropdown.js";

import {Modal} from "./classes/modal.class.js";
import {Terminal} from "./classes/terminal.class.js";
import {DocReader} from "./classes/docReader.class.js";
import {MediaPlayer} from "./classes/mediaPlayer.class.js";
import {FilesystemDisplay} from "./classes/filesystem.class.js";
import {Keyboard} from "./classes/keyboard.class.js";
import {UpdateChecker} from "./classes/updateChecker.class.js";
import {Clock} from "./classes/clock.class.js";
import {Sysinfo} from "./classes/sysinfo.class.js";
import {HardwareInspector} from "./classes/hardwareInspector.class.js";
import {Cpuinfo} from "./classes/cpuinfo.class.js";
import {Netstat} from "./classes/netstat.class.js";
import {Conninfo} from "./classes/conninfo.class.js";
import {LocationGlobe} from "./classes/locationGlobe.class.js";
import {RAMwatcher} from "./classes/ramwatcher.class.js";
import {Toplist} from "./classes/toplist.class.js";
import {FuzzyFinder} from "./classes/fuzzyFinder.class.js";
import {AudioManager} from "./classes/audiofx.class.js";

async function start() {
    await connect();
    const boot = await call("boot");
    const shims = createNodeShims(boot);

    window.edexBoot = boot;
    window.edexHost = {call, ipcRenderer, openTTY, fileURL, isWails};
    Object.assign(window, {
        require: shims.require,
        process: shims.process,
        __dirname: "",
        electronWin: shims.electronWin
    });
    // The original classes were loaded as global scripts.
    Object.assign(window, {
        Modal, Terminal, DocReader, MediaPlayer, FilesystemDisplay, Keyboard, UpdateChecker,
        Clock, Sysinfo, HardwareInspector, Cpuinfo, Netstat, Conninfo, LocationGlobe,
        RAMwatcher, Toplist, FuzzyFinder, AudioManager
    });

    await import("./renderer.js");
    initDropdown();
}

start().catch(e => {
    console.error(e);
    const screen = document.getElementById("boot_screen");
    if (screen) {
        screen.style.color = "red";
        screen.innerText = `eDEX-UI-GO failed to start: ${e && e.message ? e.message : e}`;
    }
});
