import { useEffect, useState } from "react";
import {
  InstallUpdate,
  RestartApp,
  BrowserOpenURL,
  EventsOn,
  EventsOff,
} from "../api";
import ReleaseNotes from "./update/ReleaseNotes";

// Bottom-of-window bar shown when a newer GitHub release exists. It can
// download and install the update in place, then relaunch the app.
export default function UpdateBanner({ update, onDismiss }) {
  const [phase, setPhase] = useState("idle"); // idle | installing | ready | error
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState("");
  const [showNotes, setShowNotes] = useState(false);

  useEffect(() => {
    const off = EventsOn("update:progress", (pct) => setProgress(pct || 0));
    return () => {
      EventsOff("update:progress");
      if (typeof off === "function") off();
    };
  }, []);

  if (!update) return null;

  const install = async () => {
    setPhase("installing");
    setProgress(0);
    setError("");
    try {
      await InstallUpdate();
      setPhase("ready");
    } catch (e) {
      setError(String(e));
      setPhase("error");
    }
  };

  const restart = async () => {
    try {
      await RestartApp();
    } catch (e) {
      setError(String(e));
      setPhase("error");
    }
  };

  const canShowNotes = phase !== "installing";

  return (
    <div className="update-banner" role="status">
      {showNotes && canShowNotes && (
        <div className="update-banner-notes">
          <div className="update-banner-notes-head">
            <strong>{update.releaseName || `JumpStart ${update.latestVersion}`}</strong>
            <button
              className="link-btn"
              onClick={() => BrowserOpenURL(update.releaseUrl)}
            >
              View on GitHub
            </button>
          </div>
          <ReleaseNotes notes={update.releaseNotes} />
        </div>
      )}

      <div className="update-banner-bar">
        <span className="update-banner-text">
          {phase === "ready" ? (
            <>
              <strong>Update installed</strong> — restart to run JumpStart{" "}
              {update.latestVersion}.
            </>
          ) : phase === "error" ? (
            <>
              <strong>Update failed</strong> — {error || "please try again"}.
            </>
          ) : (
            <>
              <strong>Update available</strong> — JumpStart {update.latestVersion}{" "}
              is out (you have {update.currentVersion}).
              {update.prerelease && <span className="beta-tag">Beta</span>}
            </>
          )}
        </span>

        {canShowNotes && (
          <button
            className="link-btn"
            aria-expanded={showNotes}
            onClick={() => setShowNotes((s) => !s)}
          >
            {showNotes ? "Hide notes" : "What's new"}
          </button>
        )}

        {phase === "installing" && (
          <div className="update-banner-progress" aria-hidden="true">
            <i style={{ width: `${progress}%` }} />
          </div>
        )}

        {phase === "installing" ? (
          <span className="update-banner-pct">{progress}%</span>
        ) : phase === "ready" ? (
          <button className="btn small primary" onClick={restart}>
            Restart now
          </button>
        ) : phase === "error" ? (
          <>
            <button
              className="btn small"
              onClick={() => BrowserOpenURL(update.releaseUrl)}
            >
              Download manually
            </button>
            <button className="btn small primary" onClick={install}>
              Retry
            </button>
          </>
        ) : (
          <button className="btn small primary" onClick={install}>
            Update now
          </button>
        )}

        {phase !== "installing" && (
          <button
            className="update-banner-close"
            title="Dismiss"
            onClick={onDismiss}
          >
            ✕
          </button>
        )}
      </div>
    </div>
  );
}
