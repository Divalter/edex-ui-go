#!/usr/bin/env python3
"""Startup, memory and idle CPU of eDEX-UI-GO or of the original eDEX-UI.

Usage (Linux, from any directory; both apps run fullscreen, one at a time):

    python3 measure.py go   ./build/bin/edex-ui-go
    python3 measure.py orig ./squashfs-root/edex-ui     # extracted eDEX-UI AppImage

Each app runs in a fresh configuration (nointro, no audio) and inside an
unprivileged network namespace (`unshare -rn`): the original eDEX-UI exposes
an unauthenticated terminal WebSocket on every interface, which must not be
reachable, and neither app gets network access, so both do the same work.

Reported: time until the terminal connects to the UI, CPU used by the whole
process tree over 30 s after a 20 s warm-up (percent of one core), and the
proportional set size (PSS) of the whole tree. See docs/PERFORMANCE.md.
"""
import os, sys, time, json, signal, subprocess, re, shutil

CLK = os.sysconf("SC_CLK_TCK")

def children_map():
    m = {}
    for d in os.listdir("/proc"):
        if not d.isdigit(): continue
        try:
            with open(f"/proc/{d}/stat") as f:
                s = f.read()
            ppid = int(s[s.rindex(")")+2:].split()[1])
            m.setdefault(ppid, []).append(int(d))
        except Exception:
            pass
    return m

def tree(root):
    m, out, stack = children_map(), [], [root]
    while stack:
        p = stack.pop(); out.append(p); stack += m.get(p, [])
    return out

def cpu_ticks(pids, per=None):
    t = 0
    for p in pids:
        try:
            s = open(f"/proc/{p}/stat").read()
            comm = s[s.index("(")+1:s.rindex(")")]
            f = s[s.rindex(")")+2:].split()
            v = int(f[11]) + int(f[12])
            t += v
            if per is not None: per[(p, comm)] = v
        except Exception:
            pass
    return t

def pss_kb(pids):
    total, per = 0, {}
    for p in pids:
        try:
            comm = open(f"/proc/{p}/comm").read().strip()
            for line in open(f"/proc/{p}/smaps_rollup"):
                if line.startswith("Pss:"):
                    kb = int(line.split()[1]); total += kb
                    per[comm] = per.get(comm, 0) + kb
        except Exception:
            pass
    return total, per

def run(name, cmd, env, ready_re, warmup=20, sample=30):
    log = open(f"{name}.log", "w")
    t0 = time.monotonic()
    proc = subprocess.Popen(["unshare", "-rn", "sh", "-c", "ip link set lo up && exec \"$@\"", "sh"] + cmd,
                            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, env=env, start_new_session=True, text=True)
    ready = None
    buf = []
    os.set_blocking(proc.stdout.fileno(), False)
    while time.monotonic() - t0 < 90:
        try:
            chunk = proc.stdout.read()
        except Exception:
            chunk = None
        if chunk:
            log.write(chunk); buf.append(chunk)
            if re.search(ready_re, "".join(buf)):
                ready = time.monotonic() - t0
                break
        if proc.poll() is not None: break
        time.sleep(0.02)
    res = {"name": name, "ready_s": round(ready, 2) if ready else None}
    if ready:
        time.sleep(warmup)
        pids = tree(proc.pid)
        per0, per1 = {}, {}
        c0, w0 = cpu_ticks(pids, per0), time.monotonic()
        time.sleep(sample)
        pids = tree(proc.pid)
        c1, w1 = cpu_ticks(pids, per1), time.monotonic()
        byc = {}
        for k, v in per1.items():
            byc[k[1]] = byc.get(k[1], 0) + v - per0.get(k, v)
        res["cpu_by_process_pct"] = {k: round(v / CLK / (w1 - w0) * 100, 1) for k, v in sorted(byc.items(), key=lambda x: -x[1]) if v}
        res["cpu_pct_of_one_core"] = round((c1 - c0) / CLK / (w1 - w0) * 100, 1)
        total, per = pss_kb(pids)
        res["pss_mb"] = round(total / 1024, 1)
        res["processes"] = len(pids)
        res["pss_by_process_mb"] = {k: round(v / 1024, 1) for k, v in sorted(per.items(), key=lambda x: -x[1])}
    try:
        os.killpg(proc.pid, signal.SIGTERM); time.sleep(2); os.killpg(proc.pid, signal.SIGKILL)
    except Exception:
        pass
    try:
        rest = proc.stdout.read()
        if rest: log.write(rest)
    except Exception:
        pass
    log.close()
    return res

if __name__ == "__main__":
    which = sys.argv[1]
    base = os.path.abspath("cfg-" + which)
    shutil.rmtree(base, ignore_errors=True)
    env = {k: v for k, v in os.environ.items() if not k.startswith("ELECTRON_")}
    env["XDG_CONFIG_HOME"] = base
    common = {"shellArgs": "", "keyboard": "en-US", "theme": "tron", "termFontSize": 15, "audio": False,
              "audioVolume": 1.0, "disableFeedbackAudio": True, "clockHours": 24, "pingAddr": "1.1.1.1",
              "port": 3000, "nointro": True, "nocursor": False, "forceFullscreen": True, "allowWindowed": False,
              "excludeThreadsFromToplist": True, "hideDotfiles": False, "fsListView": False,
              "experimentalGlobeFeatures": False, "experimentalFeatures": False, "shell": "bash"}
    if which == "go":
        d = os.path.join(base, "eDEX-UI-GO"); os.makedirs(d)
        s = dict(common, cwd=base, dropdownHotkey="")
        json.dump(s, open(os.path.join(d, "settings.json"), "w"))
        cmd = [sys.argv[2]]
        ready = r"connected to frontend"
    else:
        d = os.path.join(base, "eDEX-UI"); os.makedirs(d)
        json.dump(dict(common, cwd=base), open(os.path.join(d, "settings.json"), "w"))
        cmd = [sys.argv[2], "--no-sandbox"]
        ready = r"Connected to frontend"
    print(json.dumps(run(which, cmd, env, ready), indent=1))
