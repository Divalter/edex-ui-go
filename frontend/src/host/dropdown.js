/*
 * Drop-down mode (eDEX-UI-GO addition): the backend toggles the window with
 * a global hotkey (F12 by default). The UI reports whether it has the focus,
 * so that the hotkey hides a focused window and raises an unfocused one, and
 * slides down when it is shown again.
 */

import {call, ipcRenderer, isWails} from "./bridge.js";

export function initDropdown() {
    if (!isWails) return;

    const report = () => call("window.focus", document.hasFocus()).catch(() => {});
    window.addEventListener("focus", report);
    window.addEventListener("blur", report);
    report();

    ipcRenderer.on("dropdown", (e, state) => {
        if (state !== "show") return;
        const root = document.documentElement;
        root.classList.remove("dropdown_show");
        void root.offsetWidth; // restart the animation
        root.classList.add("dropdown_show");
        if (window.audioManager) window.audioManager.expand.play();
        setTimeout(() => {
            if (window.term && window.term[window.currentTerm]) window.term[window.currentTerm].term.focus();
        }, 50);
    });
}
