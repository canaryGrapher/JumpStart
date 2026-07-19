import { useEffect, useState } from "react";
import {
  InstallUpdate,
  RestartApp,
  BrowserOpenURL,
  EventsOn,
  EventsOff,
} from "../api";

// Bottom-of-window bar shown when a newer GitHub release exists. It can
// download and install the update in place, then relaunch the app.
export default function UpdateBanner({ update, onDismiss }) {
  const [phase, setPhase] = useState("idle"); // idle | installing | ready | error
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState("");

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

  return (
    <div className="update-banner" role="status">
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
          </>
        )}
      </span>

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
  );
}
