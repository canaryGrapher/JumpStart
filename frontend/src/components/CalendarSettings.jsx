import { useEffect, useState } from "react";
import QuarterEditor from "./QuarterEditor";
import { GetQuarterSettings, SetQuarterSettings } from "../api";
import { DEFAULT_QUARTERS } from "../dueDates";

// Settings → Calendar: which dates make up Q1–Q4 for the due-date filters,
// the AI, and agents (MCP). Projects can override this in their own settings.
export default function CalendarSettings({ onError }) {
  const [quarters, setQuarters] = useState(DEFAULT_QUARTERS);
  const [custom, setCustom] = useState(false);
  const [saved, setSaved] = useState(true);
  const [msg, setMsg] = useState("");

  useEffect(() => {
    let live = true;
    GetQuarterSettings()
      .then((s) => {
        if (!live) return;
        setQuarters(s.quarters?.length === 4 ? s.quarters : DEFAULT_QUARTERS);
        setCustom(!!s.custom);
      })
      .catch((e) => onError && onError(String(e)));
    return () => {
      live = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const edit = (next) => {
    setQuarters(next);
    setCustom(true);
    setSaved(false);
    setMsg("");
  };

  const save = async () => {
    try {
      await SetQuarterSettings(quarters);
      setSaved(true);
      setMsg("Saved.");
    } catch (e) {
      onError && onError(String(e));
    }
  };

  const reset = async () => {
    try {
      await SetQuarterSettings([]);
      setQuarters(DEFAULT_QUARTERS);
      setCustom(false);
      setSaved(true);
      setMsg("Using calendar-year quarters.");
    } catch (e) {
      onError && onError(String(e));
    }
  };

  return (
    <div className="prefs-section">
      <div className="prefs-row col">
        <label>Quarter dates</label>
        <p className="prefs-hint">
          Used by the Q1–Q4 due-date filters. Set these to your fiscal calendar
          — for a July–June year, Q1 runs Jul 1 to Sep 30. The same dates apply
          to the in-app AI and to agents. A project can override them in its
          own settings.
        </p>
        <QuarterEditor value={quarters} onChange={edit} />
        <div className="prefs-actions">
          <button className="btn primary small" onClick={save} disabled={saved}>
            Save
          </button>
          <button className="btn small" onClick={reset} disabled={!custom && saved}>
            Use calendar year
          </button>
          {msg && <span className="ai-status">{msg}</span>}
        </div>
      </div>
    </div>
  );
}
