import { useEffect, useState } from "react";
import { GetAppVersion, CheckForUpdate, BrowserOpenURL } from "../api";

// Preferences pane: current version and manual update check.
export default function UpdateSettings({ onError }) {
  const [version, setVersion] = useState("");
  const [checking, setChecking] = useState(false);
  const [result, setResult] = useState(null); // update.Info | "uptodate"

  useEffect(() => {
    GetAppVersion().then(setVersion).catch(() => {});
  }, []);

  const check = async () => {
    setChecking(true);
    setResult(null);
    try {
      const info = await CheckForUpdate();
      setResult(info && info.available ? info : "uptodate");
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setChecking(false);
    }
  };

  return (
    <div className="prefs-section">
      <div className="prefs-row col">
        <label>Software update</label>
        <div className="row">
          <span className="row-hint">Current version: {version || "…"}</span>
          <button className="btn small" onClick={check} disabled={checking}>
            {checking ? "Checking…" : "Check for Updates"}
          </button>
        </div>
        {result === "uptodate" && (
          <span className="ai-status ok">You're on the latest version.</span>
        )}
        {result && result !== "uptodate" && (
          <div className="row">
            <span className="ai-status">
              Version {result.latestVersion} is available.
            </span>
            <button
              className="btn small primary"
              onClick={() => BrowserOpenURL(result.releaseUrl)}
            >
              Download
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
