// Global store for the terminal dock: every process log, script run, and
// test run that would otherwise print its output inline on a card instead
// pops up here as a floating window with a tab on the bottom taskbar. Kept
// as a plain module (no React) so any component can open or close a
// terminal without threading callbacks through the whole tree; components
// read the list via hooks/useTerminalDock.js.
import { EventsOn } from "./api";

const listeners = new Set();
const exitUnsubs = new Map(); // window id -> EventsOn unsubscribe
let windows = [];
let orderSeq = 0;

function emit() {
  listeners.forEach((listener) => listener());
}

function patch(id, fields) {
  windows = windows.map((w) => (w.id === id ? { ...w, ...fields } : w));
  emit();
}

// Script and test runs are one-shot processes: watch their exit event so
// the taskbar tab and window header can show a live/succeeded/failed dot
// even while the window itself is closed or minimized.
function watchExit(id, procId) {
  if (exitUnsubs.has(id)) return;
  const off = EventsOn(`exit:${procId}`, (code) => {
    patch(id, { running: false, exitCode: code });
  });
  exitUnsubs.set(id, off);
}

function unwatchExit(id) {
  const off = exitUnsubs.get(id);
  if (off) {
    off();
    exitUnsubs.delete(id);
  }
}

export function getWindows() {
  return windows;
}

export function subscribeTerminalDock(listener) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

// Opens a new terminal window, or un-minimizes and refocuses an existing
// one with the same id (e.g. re-clicking "Logs" on a process, or reopening
// a past script run from its history chip).
export function openTerminal({ id, title, procId, source, command, running, exitCode }) {
  const existing = windows.find((w) => w.id === id);
  if (existing) {
    windows = windows.map((w) =>
      w.id === id ? { ...w, minimized: false, order: ++orderSeq } : w
    );
    emit();
    return;
  }

  const isRunning = running !== undefined ? running : source === "process";
  windows = [
    ...windows,
    {
      id,
      title,
      procId,
      source,
      command,
      startedAt: Date.now(),
      minimized: false,
      order: ++orderSeq,
      running: isRunning,
      exitCode: isRunning ? undefined : exitCode,
    },
  ];
  if (source !== "process" && isRunning) watchExit(id, procId);
  emit();
}

export function minimizeTerminal(id) {
  patch(id, { minimized: true });
}

export function restoreTerminal(id) {
  windows = windows.map((w) =>
    w.id === id ? { ...w, minimized: false, order: ++orderSeq } : w
  );
  emit();
}

export function toggleTerminal(id) {
  const w = windows.find((w) => w.id === id);
  if (!w) return;
  if (w.minimized) restoreTerminal(id);
  else minimizeTerminal(id);
}

export function closeTerminal(id) {
  unwatchExit(id);
  windows = windows.filter((w) => w.id !== id);
  emit();
}
