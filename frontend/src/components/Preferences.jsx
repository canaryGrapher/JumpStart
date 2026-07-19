import { useEffect, useState } from "react";
import ThemeToggle from "./ThemeToggle";
import SearchableSelect from "./SearchableSelect";
import { getAISettings, setAISettings, listModels, DEFAULT_HOST } from "../ai";
import { SaveGitToken, HasGitToken, DeleteGitToken } from "../api";
import UpdateSettings from "./UpdateSettings";

export const ACCENTS = [
  "forest",
  "teal",
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
          <SearchableSelect
            value={model}
            options={models}
            onChange={pickModel}
            placeholder="Select a model…"
          />
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

// A single provider row: masked status + save/remove, never redisplays the token.
function GitTokenRow({ provider, label, onError }) {
  const [hasToken, setHasToken] = useState(false);
  const [token, setToken] = useState("");
  const [busy, setBusy] = useState(false);
  const [checked, setChecked] = useState(false);

  const refresh = () =>
    HasGitToken(provider)
      .then((v) => {
        setHasToken(!!v);
        setChecked(true);
      })
      .catch((e) => onError && onError(String(e)));

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const save = async () => {
    const t = token.trim();
    if (!t) return;
    setBusy(true);
    try {
      await SaveGitToken(provider, t);
      setToken("");
      await refresh();
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const remove = async () => {
    setBusy(true);
    try {
      await DeleteGitToken(provider);
      await refresh();
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="prefs-row col">
      <label>{label}</label>
      <div className="row">
        <input
          type="password"
          placeholder={hasToken ? "•••••••••••••••• (saved)" : "Paste a personal access token"}
          value={token}
          onChange={(e) => setToken(e.target.value)}
        />
        <button className="btn small primary" disabled={busy || !token.trim()} onClick={save}>
          Save
        </button>
        <button className="btn small" disabled={busy || !hasToken} onClick={remove}>
          Remove
        </button>
      </div>
      {checked && (
        <span className={`ai-status ${hasToken ? "ok" : ""}`}>
          {hasToken ? "Token saved" : "Not set"}
        </span>
      )}
    </div>
  );
}

function GitSettings({ onError }) {
  return (
    <div className="prefs-section">
      <div className="prefs-row col">
        <span className="row-hint">
          Tokens are stored securely and used for pushing, pulling, and publishing releases to
          private remotes. They are never shown again once saved.
        </span>
      </div>
      <GitTokenRow provider="github" label="GitHub personal access token" onError={onError} />
      <GitTokenRow provider="gitlab" label="GitLab personal access token" onError={onError} />
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

  const categories = [
    { id: "appearance", label: "Appearance" },
    { id: "ai", label: "AI" },
    { id: "git", label: "Git" },
    { id: "updates", label: "Updates" },
  ];

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal prefs" onClick={(e) => e.stopPropagation()}>
        <div className="prefs-layout">
          <nav className="prefs-nav">
            <h2>Settings</h2>
            {categories.map((c) => (
              <button
                key={c.id}
                className={`prefs-nav-item ${tab === c.id ? "active" : ""}`}
                onClick={() => setTab(c.id)}
              >
                {c.label}
              </button>
            ))}
          </nav>

          <div className="prefs-content">
            <div className="prefs-content-body">
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
              ) : tab === "ai" ? (
                <AISettings onError={onError} />
              ) : tab === "git" ? (
                <GitSettings onError={onError} />
              ) : (
                <UpdateSettings onError={onError} />
              )}
            </div>

            <div className="modal-actions">
              <button className="btn primary" onClick={onClose}>
                Done
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
