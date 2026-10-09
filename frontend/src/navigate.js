// Cross-component navigation requests (from the command palette, deep
// links, Raycast): open a project, optionally a task inside it. App.jsx
// selects the project, ProjectView switches to Tasks, and TaskTracker opens
// the task once its board is loaded. A request made before those components
// mount is kept until one of them takes it.
const EVENT = "jumpstart:navigate";
let pending = null;

export function openInApp(projectId, taskId = "") {
  pending = { projectId, taskId };
  window.dispatchEvent(new CustomEvent(EVENT, { detail: pending }));
}

// Returns and clears the pending task request for this project, if any.
export function takePendingTask(projectId) {
  if (pending && pending.projectId === projectId && pending.taskId) {
    const id = pending.taskId;
    pending = null;
    return id;
  }
  return "";
}

export function peekPending() {
  return pending;
}

export function onNavigate(fn) {
  const h = (e) => fn(e.detail);
  window.addEventListener(EVENT, h);
  return () => window.removeEventListener(EVENT, h);
}

// The palette is opened by ⌘K, the sidebar search, or the system hotkey.
const PALETTE = "jumpstart:palette";
export const openPalette = (query = "") => window.dispatchEvent(new CustomEvent(PALETTE, { detail: { query } }));
export function onOpenPalette(fn) {
  const h = (e) => fn(e.detail || {});
  window.addEventListener(PALETTE, h);
  return () => window.removeEventListener(PALETTE, h);
}
