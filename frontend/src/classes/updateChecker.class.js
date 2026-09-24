/*
 * updateChecker.class.js
 *
 * Ported from eDEX-UI (https://github.com/GitSquared/edex-ui)
 * Copyright (c) 2017-2021 Gabriel 'Squared' SAILLARD <gabriel@saillard.dev>
 * Licensed under the GNU GPL v3.
 *
 * Modified for eDEX-UI-GO on 2026-09-24:
 *  - The latest release is queried by the Go backend; checks the eDEX-UI-GO repository.
 *  - Converted from a CommonJS script to an ES module.
 */
class UpdateChecker {
    constructor() {
        let electron = require("electron");
        let current = window.edexBoot.version;

        this._failed = false;
        this._fail = e => {
            this._failed = true;
            electron.ipcRenderer.send("log", "note", "UpdateChecker: Could not fetch latest release from GitHub's API.");
            electron.ipcRenderer.send("log", "debug", `Error: ${e}`);
        };

        window.edexHost.call("update.latest").then(release => {
            if (release.tag_name.slice(1) === current) {
                electron.ipcRenderer.send("log", "info", "UpdateChecker: Running latest version.");
            } else if (Number(release.tag_name.slice(1).replace(/\./g, "")) < Number(current.replace("-pre", "").replace(/\./g, ""))) {
                electron.ipcRenderer.send("log", "info", "UpdateChecker: Running an unreleased, development version.");
            } else {
                new Modal({
                    type: "info",
                    title: "New version available",
                    message: `eDEX-UI-GO <strong>${window._escapeHtml(release.tag_name)}</strong> is now available.<br/>Head over to <a href="#" onclick="electron.shell.openExternal(${window._escapeHtml(JSON.stringify(release.html_url))})">github.com</a> to download the latest version.`
                });
                electron.ipcRenderer.send("log", "info", `UpdateChecker: New version ${release.tag_name} available.`);
            }
        }).catch(e => {
            this._fail(e);
        });
    }
}

export {
    UpdateChecker
};
