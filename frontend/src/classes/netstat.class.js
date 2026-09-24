/*
 * netstat.class.js
 *
 * Ported from eDEX-UI (https://github.com/GitSquared/edex-ui)
 * Copyright (c) 2017-2021 Gabriel 'Squared' SAILLARD <gabriel@saillard.dev>
 * Licensed under the GNU GPL v3.
 *
 * Modified for eDEX-UI-GO on 2026-09-24:
 *  - External IP discovery, TCP ping and GeoIP lookups are performed by the Go backend
 *  -   (replacing the https/net Node modules and the geolite2-redist/maxmind packages).
 *  - Converted from a CommonJS script to an ES module.
 */
class Netstat {
    constructor(parentId) {
        if (!parentId) throw "Missing parameters";

        // Create DOM
        this.parent = document.getElementById(parentId);
        this.parent.innerHTML += `<div id="mod_netstat">
            <div id="mod_netstat_inner">
                <h1>NETWORK STATUS<i id="mod_netstat_iname"></i></h1>
                <div id="mod_netstat_innercontainer">
                    <div>
                        <h1>STATE</h1>
                        <h2>UNKNOWN</h2>
                    </div>
                    <div>
                        <h1>IPv4</h1>
                        <h2>--.--.--.--</h2>
                    </div>
                    <div>
                        <h1>PING</h1>
                        <h2>--ms</h2>
                    </div>
                </div>
            </div>
        </div>`;

        this.offline = false;
        this.lastconn = {finished: false}; // Prevent geoip lookup attempt until maxminddb is loaded
        this.iface = null;
        this.failedAttempts = {};
        this.runsBeforeGeoIPUpdate = 0;

        // Init updaters
        this.updateInfo();
        this.infoUpdater = setInterval(() => {
            this.updateInfo();
        }, 2000);

        // Init GeoIP integrated backend
        // Lookups are answered from a cache filled asynchronously by the Go
        // backend: an unknown address returns null and is resolved for the
        // next call.
        this._geoCache = new Map();
        this.geoLookup = {
            get: ip => {
                if (this._geoCache.has(ip)) return this._geoCache.get(ip);
                this._geoCache.set(ip, null);
                window.edexHost.call("net.geoLookup", [ip]).then(res => {
                    this._geoCache.set(ip, res[ip] || null);
                }).catch(() => {
                    this._geoCache.delete(ip);
                });
                return null;
            }
        };
        this.lastconn.finished = true;
    }
    updateInfo() {
        window.si.networkInterfaces().then(async data => {
            let offline = false;

            let net = data[0];
            let netID = 0;

            if (typeof window.settings.iface === "string") {
                while (net.iface !== window.settings.iface) {
                    netID++;
                    if (data[netID]) {
                        net = data[netID];
                    } else {
                        // No detected interface has the custom iface name, fallback to automatic detection on next loop
                        window.settings.iface = false;
                        return false;
                    }
                }
            } else {
                // Find the first external, IPv4 connected networkInterface that has a MAC address set

                while (net.operstate !== "up" || net.internal === true || net.ip4 === "" || net.mac === "") {
                    netID++;
                    if (data[netID]) {
                        net = data[netID];
                    } else {
                        // No external connection!
                        this.iface = null;
                        document.getElementById("mod_netstat_iname").innerText = "Interface: (offline)";

                        this.offline = true;
                        document.querySelector("#mod_netstat_innercontainer > div:first-child > h2").innerHTML = "OFFLINE";
                        document.querySelector("#mod_netstat_innercontainer > div:nth-child(2) > h2").innerHTML = "--.--.--.--";
                        document.querySelector("#mod_netstat_innercontainer > div:nth-child(3) > h2").innerHTML = "--ms";
                        break;
                    }
                }
            }

            if (net.ip4 !== this.internalIPv4) this.runsBeforeGeoIPUpdate = 0;

            this.iface = net.iface;
            this.internalIPv4 = net.ip4;
            document.getElementById("mod_netstat_iname").innerText = "Interface: "+net.iface;

            if (net.ip4 === "127.0.0.1") {
                offline = true;
            } else {
                if (this.runsBeforeGeoIPUpdate === 0 && this.lastconn.finished) {
                    this.lastconn = {finished: false};
                    window.edexHost.call("net.externalIP", net.ip4).then(data => {
                        this.ipinfo = {
                            ip: data.ip,
                            geo: data.geo
                        };

                        let ip = this.ipinfo.ip;
                        document.querySelector("#mod_netstat_innercontainer > div:nth-child(2) > h2").innerHTML = window._escapeHtml(ip);

                        // Retry sooner while the GeoIP database is not available yet
                        this.runsBeforeGeoIPUpdate = (data.geo) ? 10 : 3;
                    }).catch(e => {
                        this.failedAttempts[e] = (this.failedAttempts[e] || 0) + 1;
                        if (this.failedAttempts[e] > 2) return false;
                        console.warn(e);
                        window.edexHost.ipcRenderer.send("log", "note", "NetStat: Error getting data from myexternalip.com");
                        window.edexHost.ipcRenderer.send("log", "debug", `Error: ${e}`);
                    }).finally(() => {
                        this.lastconn.finished = true;
                    });
                } else if (this.runsBeforeGeoIPUpdate !== 0) {
                    this.runsBeforeGeoIPUpdate = this.runsBeforeGeoIPUpdate - 1;
                }

                let p = await this.ping(window.settings.pingAddr || "1.1.1.1", 80, net.ip4).catch(() => { offline = true });

                this.offline = offline;
                if (offline) {
                    document.querySelector("#mod_netstat_innercontainer > div:first-child > h2").innerHTML = "OFFLINE";
                    document.querySelector("#mod_netstat_innercontainer > div:nth-child(2) > h2").innerHTML = "--.--.--.--";
                    document.querySelector("#mod_netstat_innercontainer > div:nth-child(3) > h2").innerHTML = "--ms";
                } else {
                    document.querySelector("#mod_netstat_innercontainer > div:first-child > h2").innerHTML = "ONLINE";
                    document.querySelector("#mod_netstat_innercontainer > div:nth-child(3) > h2").innerHTML = Math.round(p)+"ms";
                }
            }
        });
    }
    ping(target, port, local) {
        // TCP connection time, measured by the backend (1.9s timeout)
        return window.edexHost.call("net.ping", target, port, local);
    }
}

export {
    Netstat
};
