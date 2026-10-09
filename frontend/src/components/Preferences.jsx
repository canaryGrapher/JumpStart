import { useEffect, useState } from "react";
import SearchSettings from "./SearchSettings";
import SearchableSelect from "./SearchableSelect";
import ThemeToggle from "./ThemeToggle";
import { getAISettings, setAISettings, listModels, modelInfo, DEFAULT_HOST } from "../ai";
import useAppSettings, { saveAppSettings, THINK_LEVELS } from "../appSettings";
import Switch from "./Switch";
import About from "./about/About";
import AccountsSettings from "./AccountsSettings";
import AgentsSettings from "./AgentsSettings";
import CalendarSettings from "./CalendarSettings";
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

      <ThinkingEffort model={model} onError={onError} />
    </div>
  );
}

// Reasoning effort sent to thinking-capable models as Ollama's `think`.
function ThinkingEffort({ model, onError }) {
  const settings = useAppSettings();
  const [info, setInfo] = useState(null);
  const idx = Math.max(0, THINK_LEVELS.findIndex((l) => l.id === settings.thinkLevel));

  useEffect(() => {
    let live = true;
    setInfo(null);
    if (!model) return undefined;
    modelInfo(model)
      .then((i) => live && setInfo(i))
      .catch(() => live && setInfo(undefined));
    return () => {
      live = false;
    };
  }, [model]);

  const note = !model
    ? "Pick a model to see whether it supports thinking."
    : info === null
      ? "Checking what this model supports…"
      : info === undefined
        ? "Couldn't ask Ollama what this model supports."
        : !info.thinking
          ? "This model doesn't reason before answering, so this setting has no effect on it."
          : info.thinkLevels
            ? "This model supports Low, Medium and High effort."
            : "This model only turns reasoning on or off: Low, Medium and High all mean on.";

  const set = (i) =>
    saveAppSettings({ thinkLevel: THINK_LEVELS[i].id }).catch((e) => onError && onError(String(e)));

  return (
    <div className="prefs-row col">
      <label htmlFor="think-effort">Thinking effort</label>
      <input
        id="think-effort"
        className="think-slider"
        type="range"
        min="0"
        max={THINK_LEVELS.length - 1}
        step="1"
        value={idx}
        onChange={(e) => set(Number(e.target.value))}
        aria-valuetext={THINK_LEVELS[idx].label}
      />
      <div className="think-ticks" aria-hidden>
        {THINK_LEVELS.map((l, i) => (
          <span key={l.id} className={i === idx ? "on" : ""}>
            {l.label}
          </span>
        ))}
      </div>
      <p className="prefs-hint">
        {THINK_LEVELS[idx].hint}. {note} You can override this per chat.
      </p>
    </div>
  );
}

// Settings > Tasks: how the task editor saves.
function TaskSettings({ onError }) {
  const settings = useAppSettings();
  const toggle = (on) => saveAppSettings({ autosave: on }).catch((e) => onError && onError(String(e)));
  return (
    <div className="prefs-section">
      <div className="prefs-row">
        <label>Autosave task edits</label>
        <Switch checked={!!settings.autosave} onChange={toggle} />
      </div>
      <p className="prefs-hint">
        When on, each change is saved as soon as you leave the field and the Save button is hidden.
        Cancel or Esc closes the editor and drops only the field you were still editing.
      </p>
    </div>
  );
}

// Pane categories, System Settings style: each gets a colored rounded-square
// glyph tile in the sidebar.
const CATEGORIES = [
  { id: "appearance", label: "Appearance", icon: ICONS.appearance, tint: "blue" },
  { id: "accounts", label: "Accounts", icon: ICONS.person, tint: "teal" },
  { id: "tasks", label: "Tasks", icon: ICONS.pencil, tint: "teal" },
  { id: "calendar", label: "Calendar", icon: ICONS.clock, tint: "red" },
  { id: "search", label: "Search", icon: ICONS.search, tint: "gray" },
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
              ) : tab === "tasks" ? (
                <PrefsTab title="Tasks">
                  <TaskSettings onError={onError} />
                </PrefsTab>
              ) : tab === "calendar" ? (
                <PrefsTab title="Calendar">
                  <CalendarSettings onError={onError} />
                </PrefsTab>
              ) : tab === "search" ? (
                <PrefsTab title="Search">
                  <SearchSettings onError={onError} />
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
