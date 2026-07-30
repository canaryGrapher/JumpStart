// Update channel preference: "stable" (default) or "beta".
//
// On the beta channel the backend also considers GitHub tags ending in
// "-beta" (e.g. v1.5.0-beta.2), so testers get pre-release builds offered in
// the same banner as stable ones. Stored locally alongside theme/accent.

const KEY = "updateChannel";

export const isBetaEnabled = () => localStorage.getItem(KEY) === "beta";

export function setBetaEnabled(on) {
  localStorage.setItem(KEY, on ? "beta" : "stable");
  window.dispatchEvent(new CustomEvent("updatechannelchange", { detail: on }));
}

// Subscribe to channel flips so open update UI re-checks immediately.
export function onChannelChange(fn) {
  const handler = (e) => fn(!!e.detail);
  window.addEventListener("updatechannelchange", handler);
  return () => window.removeEventListener("updatechannelchange", handler);
}
