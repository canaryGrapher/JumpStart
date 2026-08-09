// Product analytics for the JumpStart desktop app.
//
// There is no analytics SDK in this bundle, and no GA or Clarity. Wails
// ships a webview, not a browser: a JS SDK would need CSP exemptions, would
// have to be kept in step with the release cadence, would deliver nothing
// while the machine is offline, and would be blind to everything that
// happens in Startup and Shutdown.
//
// This module exposes a small PostHog-shaped API (capture / optIn / …) that
// still routes every call through Go bindings. Identity, consent, category
// gating, redaction, and the offline queue all live in internal/analytics.
import {
  TrackEvent,
  TrackEventOnce,
  TrackModelSelected,
  SetUpdateChannel,
  GetAnalyticsSettings,
  SetAnalyticsEnabled,
  SetAnalyticsDetailLevel,
  SetAnalyticsCategories,
} from "../wailsjs/go/main/App";

function safe(promise) {
  if (promise && typeof promise.catch === "function") promise.catch(() => {});
  return promise;
}

/** @deprecated Prefer capture — kept as an alias for existing call sites. */
export function track(name, props = {}) {
  return capture(name, props);
}

/** @deprecated Prefer captureOnce. */
export function trackOnce(key, name, props = {}) {
  return captureOnce(key, name, props);
}

/** Fire one event (PostHog-shaped capture). */
export function capture(event, props = {}) {
  safe(TrackEvent(event, props));
}

/** Fire at most one event per key for the rest of this app session. */
export function captureOnce(key, event, props = {}) {
  safe(TrackEventOnce(key, event, props));
}

export function trackPanel(panel) {
  captureOnce(`panel:${panel}`, "panel_opened", { panel });
}

export function trackModelSelected(model) {
  if (!model) return;
  safe(TrackModelSelected(model));
}

export function reportUpdateChannel(beta) {
  safe(SetUpdateChannel(!!beta));
}

export function getSettings() {
  return GetAnalyticsSettings();
}

export function optIn() {
  return safe(SetAnalyticsEnabled(true));
}

export function optOut() {
  return safe(SetAnalyticsEnabled(false));
}

export function setDetailLevel(level) {
  return safe(SetAnalyticsDetailLevel(level));
}

export function setCategories(categories) {
  return safe(SetAnalyticsCategories(categories || {}));
}
