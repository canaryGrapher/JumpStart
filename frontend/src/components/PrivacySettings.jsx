import { useEffect, useState } from "react";
import { GetAnalyticsSettings, SetAnalyticsEnabled, BrowserOpenURL } from "../api";
import Switch from "./Switch";

const PRIVACY_URL = "https://jumpstart.workvar.com/#/privacy";

// Privacy settings: the single switch that controls whether JumpStart
// reports anonymous usage. Turning it off stops collection immediately and
// discards anything still buffered on disk.
export default function PrivacySettings({ onError }) {
  const [enabled, setEnabled] = useState(true);
  const [configured, setConfigured] = useState(true);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    GetAnalyticsSettings()
      .then((s) => {
        setEnabled(!!(s && s.enabled));
        setConfigured(!!(s && s.configured));
      })
      .catch((e) => onError && onError(String(e)));
  }, [onError]);

  const toggle = async (next) => {
    setEnabled(next); // optimistic: the switch should never feel laggy
    setBusy(true);
    try {
      await SetAnalyticsEnabled(next);
    } catch (e) {
      setEnabled(!next);
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="prefs-section">
      <div className="prefs-row">
        <label>Share anonymous usage data</label>
        <Switch checked={enabled} disabled={busy} onChange={toggle} />
      </div>

      <div className="prefs-row col">
        <span className="row-hint">
          Helps us see which features are used and where they fail. JumpStart
          sends counts and outcomes only — never file paths, project or process
          names, repository URLs, branch names, commit messages, environment
          variables, script contents, or anything you type into AI chat.
        </span>
      </div>

      {!configured && (
        <div className="prefs-row col">
          <span className="ai-status">
            This build has no analytics configured, so nothing is sent either
            way.
          </span>
        </div>
      )}

      <div className="prefs-row col">
        <button className="link-btn" onClick={() => BrowserOpenURL(PRIVACY_URL)}>
          Read the privacy policy
        </button>
      </div>
    </div>
  );
}
