import { useEffect, useState } from "react";
import { GetAppVersion, CheckForUpdate, BrowserOpenURL } from "../api";
import { BANNER_URL_KEY } from "../hooks/useRemoteBanner";

// Preferences pane: current version, manual update check, and an optional
// override for the remote announcement banner URL.
export default function UpdateSettings({ onError }) {
  const [version, setVersion] = useState("");
  const [checking, setChecking] = useState(false);
  const [result, setResult] = useState(null); // update.Info | "uptodate"
  const [bannerUrl, setBannerUrl] = useState(
    () => localStorage.getItem(BANNER_URL_KEY) || ""
  );

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

  const saveBannerUrl = () => {
    const v = bannerUrl.trim();
    if (v) localStorage.setItem(BANNER_URL_KEY, v);
    else localStorage.removeItem(BANNER_URL_KEY);
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

      <div className="prefs-row col">
        <label>Announcement banner URL</label>
        <input
          value={bannerUrl}
          placeholder="Default: project repository banner.json"
          onChange={(e) => setBannerUrl(e.target.value)}
          onBlur={saveBannerUrl}
        />
        <span className="row-hint">
          Where JumpStart fetches remote announcements from. Leave empty to use
          the default.
        </span>
      </div>
    </div>
  );
}
