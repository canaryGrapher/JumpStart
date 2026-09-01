import { useEffect, useState } from "react";
import ThemeToggle from "./ThemeToggle";
import SearchableSelect from "./SearchableSelect";
import { getAISettings, setAISettings, listModels, DEFAULT_HOST } from "../ai";
import About from "./about/About";
import AccountsSettings from "./AccountsSettings";
import ContributeSettings from "./contribute/ContributeSettings";
import PrivacySettings from "./PrivacySettings";
import { track, trackPanel, trackModelSelected } from "../analytics";

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
    trackModelSelected(m);
  };

  return (
    <div className="prefs-section">
      <div className="prefs-row col">
        <label>Ollama host</label>
        <input
          className="ai-host"
          value={host}
          placeholder={DEFAULT_HOST}
          onChange={(e) => setHost(e.target.value)}
          onBlur={() => setAISettings({ host })}
        />
        <div className="prefs-actions">
          <button className="btn" onClick={refresh} disabled={loading}>
            {loading ? "…" : "Refresh models"}
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

function PrefsTab({ title, children }) {
  return (
    <div className="prefs-tab-panel">
      <header className="prefs-tab-head">
        <h3 className="prefs-tab-title">{title}</h3>
      </header>
      <div className="prefs-tab-body">{children}</div>
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
    { id: "accounts", label: "Accounts" },
    { id: "privacy", label: "Privacy" },
    { id: "contribute", label: "Contribute" },
    { id: "about", label: "About" },
  ];

  const openTab = (id) => {
    setTab(id);
    trackPanel(`prefs_${id}`);
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal prefs" onClick={(e) => e.stopPropagation()}>
        <div className="prefs-layout">
          <nav className="prefs-nav">
            <h2>Settings</h2>
            {categories.map((c) => (
              <button
                key={c.id}
                type="button"
                className={`prefs-nav-item ${tab === c.id ? "active" : ""}`}
                onClick={() => openTab(c.id)}
              >
                {c.label}
              </button>
            ))}
          </nav>

          <div className="prefs-content">
            <div className="prefs-content-body">
              {tab === "appearance" ? (
                <PrefsTab title="Appearance">
                  <div className="prefs-section">
                    <div className="prefs-row">
                      <label>Theme</label>
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
                            onClick={() => {
                              onAccentChange(a);
                              track("accent_changed", { to: a });
                            }}
                          />
                        ))}
                      </div>
                    </div>
                  </div>
                </PrefsTab>
              ) : tab === "ai" ? (
                <PrefsTab title="AI">
                  <AISettings onError={onError} />
                </PrefsTab>
              ) : tab === "accounts" ? (
                <PrefsTab title="Accounts">
                  <AccountsSettings onError={onError} />
                </PrefsTab>
              ) : tab === "privacy" ? (
                <PrefsTab title="Privacy">
                  <PrivacySettings onError={onError} />
                </PrefsTab>
              ) : tab === "contribute" ? (
                <PrefsTab title="Contribute">
                  <ContributeSettings onError={onError} onConnectGitHub={() => setTab("accounts")} />
                </PrefsTab>
              ) : (
                <PrefsTab title="About">
                  <About onError={onError} />
                </PrefsTab>
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
