import { useEffect, useState } from "react";
import { GetAppVersion, CheckForUpdate, BrowserOpenURL } from "../api";
import { isBetaEnabled, setBetaEnabled } from "../updateChannel";
import Switch from "./Switch";
import ReleaseNotes from "./update/ReleaseNotes";

// Preferences pane: current version, update channel, and manual update check.
export default function UpdateSettings({ onError }) {
  const [version, setVersion] = useState("");
  const [beta, setBeta] = useState(isBetaEnabled);
  const [checking, setChecking] = useState(false);
  const [result, setResult] = useState(null); // update.Info | "uptodate"

  useEffect(() => {
    GetAppVersion().then(setVersion).catch(() => {});
  }, []);

  const check = async (useBeta = beta) => {
    setChecking(true);
    setResult(null);
    try {
      const info = await CheckForUpdate(useBeta);
      setResult(info && info.available ? info : "uptodate");
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setChecking(false);
    }
  };

  // Flipping the channel re-checks straight away, so enabling beta surfaces a
  // pending pre-release without the user hunting for the button.
  const toggleBeta = (on) => {
    setBeta(on);
    setBetaEnabled(on);
    check(on);
  };

  return (
    <div className="prefs-section">
      <div className="prefs-row col">
        <label>Software update</label>
        <div className="row">
          <span className="row-hint">Current version: {version || "…"}</span>
          <button className="btn small" onClick={() => check()} disabled={checking}>
            {checking ? "Checking…" : "Check for Updates"}
          </button>
        </div>
      </div>

      <div className="prefs-row">
        <label>Beta updates</label>
        <Switch checked={beta} onChange={toggleBeta} />
      </div>
      <div className="prefs-row col">
        <span className="row-hint">
          Receive pre-release builds (tags ending in <code>-beta</code>) as soon as they
          are published. Beta builds get new features first and may be less stable. With
          this off, only stable releases are offered.
        </span>
      </div>

      {result === "uptodate" && (
        <div className="prefs-row col">
          <span className="ai-status ok">
            You're on the latest {beta ? "beta" : "stable"} version.
          </span>
        </div>
      )}

      {result && result !== "uptodate" && (
        <div className="prefs-row col update-available">
          <div className="row">
            <span className="ai-status">
              Version {result.latestVersion} is available.
            </span>
            {result.prerelease && <span className="beta-tag">Beta</span>}
            <button
              className="btn small primary"
              onClick={() => BrowserOpenURL(result.releaseUrl)}
            >
              Download
            </button>
          </div>
          <label className="release-notes-label">What's new</label>
          <ReleaseNotes notes={result.releaseNotes} className="boxed" />
        </div>
      )}
    </div>
  );
}
