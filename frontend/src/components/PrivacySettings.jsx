import { useEffect, useState } from "react";
import {
  GetAnalyticsSettings,
  SetAnalyticsEnabled,
  SetAnalyticsDetailLevel,
  SetAnalyticsCategories,
  BrowserOpenURL,
} from "../api";
import Switch from "./Switch";

const PRIVACY_URL = "https://jumpstart.workvar.com/#/privacy";

const LEVELS = [
  { id: "full", label: "Full" },
  { id: "balanced", label: "Balanced" },
  { id: "minimal", label: "Minimal" },
];

const CATEGORY_LABELS = [
  { id: "lifecycle", label: "Lifecycle" },
  { id: "onboarding", label: "Onboarding" },
  { id: "processes", label: "Processes" },
  { id: "git_docker", label: "Git & Docker" },
  { id: "kanban", label: "Kanban" },
  { id: "ai", label: "AI" },
  { id: "updates", label: "Updates & banners" },
  { id: "ui_panels", label: "UI panels" },
];

const defaultCategories = () =>
  Object.fromEntries(CATEGORY_LABELS.map((c) => [c.id, true]));

// Privacy settings: master switch, detail-level presets, and per-category
// toggles. Turning the master off stops collection and discards the queue.
export default function PrivacySettings({ onError }) {
  const [enabled, setEnabled] = useState(true);
  const [configured, setConfigured] = useState(true);
  const [detailLevel, setDetailLevel] = useState("full");
  const [categories, setCategories] = useState(defaultCategories);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    GetAnalyticsSettings()
      .then((s) => {
        setEnabled(!!(s && s.enabled));
        setConfigured(!!(s && s.configured));
        setDetailLevel((s && s.detailLevel) || "full");
        setCategories({ ...defaultCategories(), ...((s && s.categories) || {}) });
      })
      .catch((e) => onError && onError(String(e)));
  }, [onError]);

  const toggle = async (next) => {
    setEnabled(next);
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

  const applyLevel = async (level) => {
    const prev = detailLevel;
    setDetailLevel(level);
    setBusy(true);
    try {
      await SetAnalyticsDetailLevel(level);
      const s = await GetAnalyticsSettings();
      setDetailLevel((s && s.detailLevel) || level);
      setCategories({ ...defaultCategories(), ...((s && s.categories) || {}) });
    } catch (e) {
      setDetailLevel(prev);
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const toggleCategory = async (id, next) => {
    const prev = categories;
    const updated = { ...categories, [id]: next };
    setCategories(updated);
    setDetailLevel("custom");
    setBusy(true);
    try {
      await SetAnalyticsCategories(updated);
      const s = await GetAnalyticsSettings();
      setDetailLevel((s && s.detailLevel) || "custom");
      setCategories({ ...defaultCategories(), ...((s && s.categories) || {}) });
    } catch (e) {
      setCategories(prev);
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const levelActive = detailLevel === "custom" ? null : detailLevel;
  const prefsDisabled = busy || !enabled;

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

      <div className={`prefs-row col ${prefsDisabled ? "prefs-dimmed" : ""}`}>
        <label>Detail level</label>
        <div className="prefs-seg" role="group" aria-label="Analytics detail level">
          {LEVELS.map((l) => (
            <button
              key={l.id}
              type="button"
              className={`prefs-seg-btn ${levelActive === l.id ? "active" : ""}`}
              disabled={prefsDisabled}
              onClick={() => applyLevel(l.id)}
            >
              {l.label}
            </button>
          ))}
        </div>
        {detailLevel === "custom" && (
          <span className="row-hint">Custom — individual categories below.</span>
        )}
        {detailLevel === "balanced" && (
          <span className="row-hint">Balanced turns off UI panel reach events.</span>
        )}
        {detailLevel === "minimal" && (
          <span className="row-hint">Minimal keeps lifecycle and update events only.</span>
        )}
      </div>

      <div className={`prefs-row col ${prefsDisabled ? "prefs-dimmed" : ""}`}>
        <label>What to share</label>
        <div className="analytics-cats">
          {CATEGORY_LABELS.map((c) => (
            <div className="prefs-row analytics-cat" key={c.id}>
              <label>{c.label}</label>
              <Switch
                checked={!!categories[c.id]}
                disabled={prefsDisabled}
                onChange={(v) => toggleCategory(c.id, v)}
              />
            </div>
          ))}
        </div>
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
