import { useEffect, useState } from "react";
import ThemeToggle from "./ThemeToggle";
import { getAISettings, setAISettings, listModels, DEFAULT_HOST } from "../ai";

export const ACCENTS = [
  "blue",
  "purple",
  "pink",
  "red",
  "orange",
  "yellow",
  "green",
  "graphite",
];

// AI section: Ollama host + auto-detected model picker.
function AISettings({ onError }) {
  const initial = getAISettings();
  const [host, setHost] = useState(initial.host);
  const [model, setModel] = useState(initial.model);
  const [models, setModels] = useState([]);
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState("");

  const refresh = async () => {
    setLoading(true);
    setStatus("");
    setAISettings({ host });
    try {
      const list = await listModels(host);
      setModels(list || []);
      if (!list || list.length === 0) {
        setStatus("No models installed. Run `ollama pull llama3.2` first.");
      } else {
        setStatus(`Found ${list.length} model(s).`);
        if (!model || !list.includes(model)) {
          setModel(list[0]);
          setAISettings({ model: list[0] });
        }
      }
    } catch (e) {
      setStatus("Couldn't reach Ollama.");
      onError && onError(String(e));
    } finally {
      setLoading(false);
    }
  };

  // Try once on open so the current model list is visible.
  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const pickModel = (m) => {
    setModel(m);
    setAISettings({ model: m });
  };

  return (
    <div className="prefs-section">
      <div className="prefs-row col">
        <label>Ollama host</label>
        <div className="row">
          <input
            className="ai-host"
            value={host}
            placeholder={DEFAULT_HOST}
            onChange={(e) => setHost(e.target.value)}
            onBlur={() => setAISettings({ host })}
          />
          <button className="btn small" onClick={refresh} disabled={loading}>
            {loading ? "…" : "Refresh"}
          </button>
        </div>
      </div>

      <div className="prefs-row col">
        <label>Model</label>
        {models.length > 0 ? (
          <select
            className="ai-model"
            value={model}
            onChange={(e) => pickModel(e.target.value)}
          >
            {models.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
        ) : (
          <input
            className="ai-model"
            value={model}
            placeholder="e.g. llama3.2"
            onChange={(e) => pickModel(e.target.value)}
          />
        )}
        {status && <span className="ai-status">{status}</span>}
      </div>
    </div>
  );
}

export default function Preferences({
  theme,
  onThemeChange,
  accent,
  onAccentChange,
  onError,
  onClose,
}) {
  const [tab, setTab] = useState("appearance");

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal prefs" onClick={(e) => e.stopPropagation()}>
        <h2>Preferences</h2>

        <div className="prefs-tabs">
          <button
            className={tab === "appearance" ? "active" : ""}
            onClick={() => setTab("appearance")}
          >
            Appearance
          </button>
          <button
            className={tab === "ai" ? "active" : ""}
            onClick={() => setTab("ai")}
          >
            AI
          </button>
        </div>

        {tab === "appearance" ? (
          <>
            <div className="prefs-row">
              <label>Appearance</label>
              <ThemeToggle theme={theme} onChange={onThemeChange} />
            </div>

            <div className="prefs-row">
              <label>Accent color</label>
              <div className="swatches">
                {ACCENTS.map((a) => (
                  <button
                    key={a}
                    className={`swatch ${accent === a ? "active" : ""}`}
                    data-swatch={a}
                    title={a}
                    aria-label={a}
                    aria-pressed={accent === a}
                    onClick={() => onAccentChange(a)}
                  />
                ))}
              </div>
            </div>
          </>
        ) : (
          <AISettings onError={onError} />
        )}

        <div className="modal-actions">
          <button className="btn primary" onClick={onClose}>
            Done
          </button>
        </div>
      </div>
    </div>
  );
}
