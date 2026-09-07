import { useEffect, useRef, useState } from "react";
import {
  InstallUpdate,
  RestartApp,
  BrowserOpenURL,
  EventsOn,
  EventsOff,
} from "../api";
import ReleaseNotes from "./update/ReleaseNotes";
import { track } from "../analytics";

// Module-level install coordination so React Strict Mode remounts share one
// in-flight InstallUpdate and remember which versions already finished.
const readyVersions = new Set();
const inFlight = new Map(); // version -> Promise

function ensureInstall(version) {
  if (readyVersions.has(version)) {
    return Promise.resolve();
  }
  const existing = inFlight.get(version);
  if (existing) return existing;

  const pending = InstallUpdate()
    .then(() => {
      readyVersions.add(version);
      inFlight.delete(version);
    })
    .catch((err) => {
      inFlight.delete(version);
      throw err;
    });
  inFlight.set(version, pending);
  return pending;
}

// Compact notice in the bottom-left when a newer GitHub release exists.
// Download starts automatically; the user only needs to restart when ready.
export default function UpdateBanner({ update, onDismiss }) {
  const [phase, setPhase] = useState("idle"); // idle | installing | ready | error
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState("");
  const [showNotes, setShowNotes] = useState(false);
  const dismissedRef = useRef(false);

  useEffect(() => {
    const offProgress = EventsOn("update:progress", (pct) =>
      setProgress(pct || 0)
    );
    const offReady = EventsOn("update:ready", () => {
      if (dismissedRef.current) return;
      setPhase("ready");
    });
    return () => {
      EventsOff("update:progress");
      EventsOff("update:ready");
      if (typeof offProgress === "function") offProgress();
      if (typeof offReady === "function") offReady();
    };
  }, []);

  useEffect(() => {
    dismissedRef.current = false;
  }, [update?.latestVersion]);

  const install = async (version, { force = false } = {}) => {
    if (force) {
      readyVersions.delete(version);
      inFlight.delete(version);
    } else if (readyVersions.has(version)) {
      if (!dismissedRef.current) setPhase("ready");
      return;
    }

    setPhase("installing");
    setProgress(0);
    setError("");
    try {
      await ensureInstall(version);
      if (dismissedRef.current) return;
      setPhase("ready");
    } catch (e) {
      const msg = String(e);
      if (/already in progress/i.test(msg)) {
        // Another path holds the Go mutex; wait on progress / ready events.
        setPhase("installing");
        return;
      }
      if (dismissedRef.current) return;
      setError(msg);
      setPhase("error");
    }
  };

  useEffect(() => {
    if (!update?.latestVersion) return;
    install(update.latestVersion);
  }, [update?.latestVersion]);

  if (!update) return null;

  const restart = async () => {
    try {
      await RestartApp();
    } catch (e) {
      setError(String(e));
      setPhase("error");
    }
  };

  const canShowNotes = phase !== "installing";

  // Dismissals are the counterweight to update_installed: a version that is
  // repeatedly waved away is a release-notes problem, not an updater one.
  const dismiss = () => {
    dismissedRef.current = true;
    track("update_dismissed", {
      latest_version: update.latestVersion,
      prerelease: !!update.prerelease,
      saw_notes: showNotes,
      phase,
    });
    onDismiss();
  };

  const toggleNotes = () => {
    if (!showNotes) {
      track("update_notes_opened", { latest_version: update.latestVersion });
    }
    setShowNotes((s) => !s);
  };

  const statusText = () => {
    if (phase === "ready") {
      return (
        <>
          <strong>Update installed</strong> — restart to run JumpStart{" "}
          {update.latestVersion}.
        </>
      );
    }
    if (phase === "error") {
      return (
        <>
          <strong>Update failed</strong> — {error || "please try again"}.
        </>
      );
    }
    if (phase === "installing") {
      return (
        <>
          <strong>Update available</strong> — downloading JumpStart{" "}
          {update.latestVersion}
          {update.prerelease && <span className="beta-tag">Beta</span>}
          …
        </>
      );
    }
    return (
      <>
        <strong>Update available</strong> — JumpStart {update.latestVersion}{" "}
        is out (you have {update.currentVersion}).
        {update.prerelease && <span className="beta-tag">Beta</span>}
      </>
    );
  };

  const phaseClass =
    phase === "ready" ? " is-ready" : phase === "error" ? " is-error" : "";

  return (
    <div className={`update-banner${phaseClass}`} role="status" aria-live="polite">
      <button
        className="update-banner-close"
        type="button"
        title="Dismiss"
        aria-label="Dismiss update"
        onClick={dismiss}
      >
        ✕
      </button>

      {showNotes && canShowNotes && (
        <div className="update-banner-notes">
          <div className="update-banner-notes-head">
            <strong>{update.releaseName || `JumpStart ${update.latestVersion}`}</strong>
            <button
              className="link-btn"
              type="button"
              onClick={() => BrowserOpenURL(update.releaseUrl)}
            >
              View on GitHub
            </button>
          </div>
          <ReleaseNotes notes={update.releaseNotes} />
        </div>
      )}

      <p className="update-banner-text">{statusText()}</p>

      {phase === "installing" && (
        <div className="update-banner-progress-row">
          <div className="update-banner-progress" aria-hidden="true">
            <i style={{ width: `${progress}%` }} />
          </div>
          <span className="update-banner-pct">{progress}%</span>
        </div>
      )}

      {(canShowNotes || phase === "ready" || phase === "error") && (
        <div className="update-banner-actions">
          {canShowNotes && (
            <button
              className="link-btn"
              type="button"
              aria-expanded={showNotes}
              onClick={toggleNotes}
            >
              {showNotes ? "Hide notes" : "What's new"}
            </button>
          )}

          {phase === "ready" && (
            <button className="btn small primary" type="button" onClick={restart}>
              Restart now
            </button>
          )}

          {phase === "error" && (
            <div className="update-banner-cta">
              <button
                className="btn small"
                type="button"
                onClick={() => BrowserOpenURL(update.releaseUrl)}
              >
                Download manually
              </button>
              <button
                className="btn small primary"
                type="button"
                onClick={() => install(update.latestVersion, { force: true })}
              >
                Retry
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
