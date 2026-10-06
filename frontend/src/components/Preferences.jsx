import { useEffect, useState } from "react";
import SearchableSelect from "./SearchableSelect";
import ThemeToggle from "./ThemeToggle";
import { getAISettings, setAISettings, listModels, DEFAULT_HOST } from "../ai";
import About from "./about/About";
import AccountsSettings from "./AccountsSettings";
import AgentsSettings from "./AgentsSettings";
import ContributeSettings from "./contribute/ContributeSettings";
import PrivacySettings from "./PrivacySettings";
import { trackPanel, trackModelSelected } from "../analytics";
import Icon, { ICONS } from "./Icon";

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

// Pane categories, System Settings style: each gets a colored rounded-square
// glyph tile in the sidebar.
const CATEGORIES = [
  { id: "appearance", label: "Appearance", icon: ICONS.appearance, tint: "blue" },
  { id: "accounts", label: "Accounts", icon: ICONS.person, tint: "teal" },
  { id: "ai", label: "AI", icon: ICONS.sparkles, tint: "purple" },
  { id: "agents", label: "Agents", icon: ICONS.bolt, tint: "orange" },
  { id: "privacy", label: "Privacy", icon: ICONS.hand, tint: "indigo" },
  { id: "contribute", label: "Contribute", icon: ICONS.heart, tint: "pink" },
  { id: "about", label: "About", icon: ICONS.info, tint: "gray" },
];

function PrefsTab({ title, children }) {
  return (
    <div className="prefs-tab-panel" aria-label={title}>
      <div className="prefs-tab-body">{children}</div>
    </div>
  );
}

export default function Preferences({ theme, onThemeChange, onError, onClose }) {
  const [tab, setTab] = useState("appearance");
  const current = CATEGORIES.find((c) => c.id === tab) || CATEGORIES[0];

  const openTab = (id) => {
    setTab(id);
    trackPanel(`prefs_${id}`);
  };

  // Esc closes, like a macOS settings window's ⌘W.
  useEffect(() => {
    const onKey = (e) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal prefs"
        role="dialog"
        aria-label="Settings"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="prefs-layout">
          <nav className="prefs-nav">
            <div className="prefs-nav-chrome">
              <button
                type="button"
                className="prefs-close"
                aria-label="Close Settings"
                title="Close"
                onClick={onClose}
              >
                <Icon d={ICONS.close} />
              </button>
            </div>
            {CATEGORIES.map((c) => (
              <button
                key={c.id}
                type="button"
                className={`prefs-nav-item ${tab === c.id ? "active" : ""}`}
                onClick={() => openTab(c.id)}
              >
                <span className={`prefs-glyph tint-${c.tint}`}>
                  <Icon d={c.icon} />
                </span>
                {c.label}
              </button>
            ))}
          </nav>

          <div className="prefs-content">
            <header className="prefs-toolbar">
              <h2>{current.label}</h2>
            </header>
            <div className="prefs-content-body" key={tab}>
              {tab === "appearance" ? (
                <PrefsTab title="Appearance">
                  <div className="prefs-section">
                    <div className="prefs-row">
                      <label>Theme</label>
                      <ThemeToggle theme={theme} onChange={onThemeChange} />
                    </div>
                  </div>
                </PrefsTab>
              ) : tab === "ai" ? (
                <PrefsTab title="AI">
                  <AISettings onError={onError} />
                </PrefsTab>
              ) : tab === "agents" ? (
                <PrefsTab title="Agents">
                  <AgentsSettings onError={onError} />
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
          </div>
        </div>
      </div>
    </div>
  );
}
