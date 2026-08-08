// Product analytics for the JumpStart desktop app.
//
// There is no analytics SDK in this bundle, and no GA or Clarity. Wails
// ships a webview, not a browser: a JS SDK would need CSP exemptions, would
// have to be kept in step with the release cadence, would deliver nothing
// while the machine is offline, and would be blind to everything that
// happens in Startup and Shutdown.
//
// The Go process already sees every user action, because every action
// crosses the Wails binding boundary. So Go owns ingestion (PostHog, via
// internal/analytics) and this file is a thin bridge for the handful of
// events only the UI can observe: which panel was opened, whether an AI
// suggestion was kept, which banner was clicked.
//
// Everything here routes through the same consent gate and the same
// redaction as the Go side. There is exactly one way out of this app.
import {
  TrackEvent,
  TrackEventOnce,
  TrackModelSelected,
  SetUpdateChannel,
} from "../wailsjs/go/main/App";

// Analytics must never break a UI interaction, so every call swallows its
// own failures. A dropped event is not worth a broken button.
function safe(promise) {
  if (promise && typeof promise.catch === "function") promise.catch(() => {});
}

/** Fire one event. Property values must be counts, booleans, or short
 *  bounded strings — never paths, names, or free text. */
export function track(name, props = {}) {
  safe(TrackEvent(name, props));
}

/** Fire at most one event per key for the rest of this app session. */
export function trackOnce(key, name, props = {}) {
  safe(TrackEventOnce(key, name, props));
}

/** Record that a panel was opened, once per panel per session.
 *
 *  Deliberately deduplicated: this is a reach metric ("what fraction of
 *  users ever open the Docker panel?"), and firing on every click would
 *  multiply event volume without answering a different question. */
export function trackPanel(panel) {
  trackOnce(`panel:${panel}`, "panel_opened", { panel });
}

/** Record which local model the user selected.
 *
 *  The raw tag is handed to Go rather than parsed here: a model name can be
 *  a private fine-tune, so only Go's bounded family/size split is ever
 *  emitted. Never call track("ai_model_selected", { model }) directly. */
export function trackModelSelected(model) {
  if (!model) return;
  safe(TrackModelSelected(model));
}

/** Tell the Go side which update channel this user is on, so it can break
 *  every event down by the beta cohort. The channel preference lives in
 *  localStorage, so Go cannot know it without being told. */
export function reportUpdateChannel(beta) {
  safe(SetUpdateChannel(!!beta));
}
