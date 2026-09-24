/*
 * Frame clock of the always-running animations (eDEX-UI-GO addition): the
 * globe and the smoothie charts of the CPU and network panels.
 *
 * - Normal mode keeps the original timing: each animation requests its own
 *   frames (globe ~30 fps, charts 30 and 40 fps).
 * - Eco mode (ecoMode setting) draws them all on the same ticks at ECO_FPS.
 *   WebKit composites the whole page for every frame in which anything
 *   changed, so aligned ticks matter: three 10 fps loops out of phase would
 *   still make up to 30 frames per second.
 * - Paused (the window is covered, see occlusion in the backend) holds every
 *   frame until the window can be seen again. Hidden or minimized windows do
 *   not need it: the webview already stops requestAnimationFrame for them.
 */

export const ECO_FPS = 10;

let eco = false;
let paused = false;

let nextId = 1;
const pending = new Map(); // id -> {cb, native, timer}
let ecoQueue = [];
let ecoFrame = 0;
let nextTick = 0;

function run(id, now) {
    const entry = pending.get(id);
    if (!entry) return;
    pending.delete(id);
    try {
        entry.cb(now);
    } catch (e) {
        console.error(e);
    }
}

// A vsync slightly early still counts as the tick.
const slack = 4;

function ecoLoop(now) {
    ecoFrame = 0;
    if (paused) return;
    if (now < nextTick - slack) {
        ecoFrame = requestAnimationFrame(ecoLoop);
        return;
    }
    nextTick = Math.max(nextTick + 1000 / ECO_FPS, now);
    const ids = ecoQueue;
    ecoQueue = [];
    ids.forEach(id => run(id, now));
    if (ecoQueue.length && !ecoFrame) ecoFrame = requestAnimationFrame(ecoLoop);
}

function schedule(id) {
    const entry = pending.get(id);
    if (!entry || paused) return;
    if (eco) {
        ecoQueue.push(id);
        if (!ecoFrame) ecoFrame = requestAnimationFrame(ecoLoop);
    } else if (entry.after) {
        entry.timer = setTimeout(() => {
            entry.timer = 0;
            if (paused || eco) return schedule(id);
            entry.native = requestAnimationFrame(now => run(id, now));
        }, entry.after);
    } else {
        entry.native = requestAnimationFrame(now => run(id, now));
    }
}

/*
 * requestFrame is requestAnimationFrame for the animations above. In normal
 * mode, after (ms) waits before requesting the frame, like the original globe
 * loop did with setTimeout.
 */
export function requestFrame(cb, after = 0) {
    const id = nextId++;
    pending.set(id, {cb, after, native: 0, timer: 0});
    schedule(id);
    return id;
}

export function cancelFrame(id) {
    const entry = pending.get(id);
    if (!entry) return;
    if (entry.native) cancelAnimationFrame(entry.native);
    if (entry.timer) clearTimeout(entry.timer);
    pending.delete(id);
    ecoQueue = ecoQueue.filter(i => i !== id);
}

// Drops the scheduled frames and schedules them again in the new state.
function reschedule() {
    for (const [id, entry] of pending) {
        if (entry.native) cancelAnimationFrame(entry.native);
        if (entry.timer) clearTimeout(entry.timer);
        entry.native = entry.timer = 0;
    }
    ecoQueue = [];
    if (ecoFrame) cancelAnimationFrame(ecoFrame);
    ecoFrame = 0;
    for (const id of pending.keys()) schedule(id);
}

export function setEco(on) {
    if (eco === !!on) return;
    eco = !!on;
    reschedule();
}

export function setPaused(on) {
    if (paused === !!on) return;
    paused = !!on;
    reschedule();
}

export function isEco() {
    return eco;
}

// Smoothie charts schedule their frames through AnimateCompatibility.
export function paceSmoothie(smoothie) {
    smoothie.SmoothieChart.AnimateCompatibility.requestAnimationFrame = cb => requestFrame(cb);
    smoothie.SmoothieChart.AnimateCompatibility.cancelAnimationFrame = cancelFrame;
}
