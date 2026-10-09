import { useEffect, useState } from "react";
import { GetHotkeyStatus, OCREngines, RereadAllImages } from "../api";
import useAppSettings, { saveAppSettings } from "../appSettings";
import { getAISettings, listModels } from "../ai";

const ENGINE_LABELS = {
  vision: "macOS Vision (on-device)",
  tesseract: "Tesseract",
  ollama: "Ollama vision model",
  off: "Off",
};

// Settings > Search: the palette shortcut, how text is read from images
// (OCR), and how ambiguous dates like 10/9 are read.
export default function SearchSettings({ onError }) {
  const settings = useAppSettings();
  const [engines, setEngines] = useState([]);
  const [hotkey, setHotkey] = useState(null);
  const [key, setKey] = useState(settings.hotkeyKey);
  const [models, setModels] = useState([]);
  const [note, setNote] = useState("");

  const refresh = () => {
    OCREngines().then(setEngines).catch(() => {});
    GetHotkeyStatus().then(setHotkey).catch(() => {});
  };
  useEffect(refresh, [settings.ocrEngine, settings.ocrModel, settings.hotkeyMode, settings.hotkeyKey]);
  useEffect(() => setKey(settings.hotkeyKey), [settings.hotkeyKey]);
  useEffect(() => {
    if (settings.ocrEngine !== "ollama") return;
    listModels(settings.ocrHost || getAISettings().host)
      .then((m) => setModels(m || []))
      .catch(() => setModels([]));
  }, [settings.ocrEngine, settings.ocrHost]);

  const save = (patch) => saveAppSettings(patch).catch((e) => onError && onError(String(e)));
  const status = (id) => engines.find((e) => e.engine === id);
  const pdf = status("pdf");
  const current = status(settings.ocrEngine);

  return (
    <>
      <div className="prefs-section">
        <div className="prefs-row">
          <label htmlFor="hotkey-mode">Search shortcut</label>
          <select id="hotkey-mode" value={settings.hotkeyMode} onChange={(e) => save({ hotkeyMode: e.target.value })}>
            <option value="app">⌘K inside JumpStart</option>
            <option value="system">⌘K, plus a system-wide shortcut</option>
            <option value="off">Off</option>
          </select>
        </div>
        {settings.hotkeyMode === "system" && (
          <div className="prefs-row">
            <label htmlFor="hotkey-key">System-wide shortcut</label>
            <span className="prefs-inline">
              <input id="hotkey-key" value={key} onChange={(e) => setKey(e.target.value)} placeholder="cmd+shift+j" spellCheck={false} />
              <button type="button" className="btn small" disabled={key === settings.hotkeyKey} onClick={() => save({ hotkeyKey: key.trim() })}>
                Apply
              </button>
            </span>
          </div>
        )}
        <p className="prefs-hint">
          {settings.hotkeyMode === "off"
            ? "The palette still opens from the sidebar search."
            : settings.hotkeyMode === "system"
              ? hotkey?.registered
                ? `Press ${settings.hotkeyKey} anywhere to bring JumpStart forward with the search open.`
                : hotkey?.error || "Registering the shortcut…"
              : "Press ⌘K anywhere in JumpStart to search. The Raycast extension is the other way to search from anywhere."}
        </p>
      </div>

      <div className="prefs-section">
        <div className="prefs-row">
          <label htmlFor="ocr-engine">Read text in images</label>
          <select id="ocr-engine" value={settings.ocrEngine} onChange={(e) => save({ ocrEngine: e.target.value })}>
            {Object.entries(ENGINE_LABELS).map(([id, label]) => {
              const st = status(id);
              return (
                <option key={id} value={id}>
                  {label}
                  {st && !st.available && id !== "ollama" ? " (not available)" : ""}
                </option>
              );
            })}
          </select>
        </div>
        {settings.ocrEngine === "ollama" && (
          <>
            <div className="prefs-row">
              <label htmlFor="ocr-model">Vision model</label>
              <select id="ocr-model" value={settings.ocrModel || ""} onChange={(e) => save({ ocrModel: e.target.value })}>
                <option value="">Choose…</option>
                {models.map((m) => (
                  <option key={m} value={m}>{m}</option>
                ))}
              </select>
            </div>
            <div className="prefs-row">
              <label htmlFor="ocr-host">Ollama address</label>
              <input id="ocr-host" defaultValue={settings.ocrHost || ""} placeholder={getAISettings().host}
                onBlur={(e) => e.target.value !== (settings.ocrHost || "") && save({ ocrHost: e.target.value.trim() })} />
            </div>
          </>
        )}
        <p className="prefs-hint">
          {current?.detail} Text found in screenshots and photos is searchable from the palette, the MCP search tool and Raycast.
          It is read once in the background and stays on this Mac.
        </p>
        <p className="prefs-hint">{pdf?.available ? "PDF text is searchable." : pdf?.detail}</p>
        <div className="prefs-row">
          <label>Re-read images</label>
          <span className="prefs-inline">
            <button
              type="button"
              className="btn small"
              onClick={async () => {
                const n = await RereadAllImages(false);
                setNote(n ? `Reading ${n} image${n === 1 ? "" : "s"} again in the background.` : "Reading images in the background.");
              }}
            >
              Read again
            </button>
            {note && <span className="prefs-hint">{note}</span>}
          </span>
        </div>
      </div>

      <div className="prefs-section">
        <div className="prefs-row">
          <label htmlFor="date-order">Dates like 10/9</label>
          <select id="date-order" value={settings.dateOrder} onChange={(e) => save({ dateOrder: e.target.value })}>
            <option value="mdy">Month first (October 9)</option>
            <option value="dmy">Day first (10 September)</option>
          </select>
        </div>
        <p className="prefs-hint">
          Search understands dates such as today, next friday, this quarter, Oct 9, 2026-10-09 and 10/9/2026, and qualifiers
          such as status:todo, @name, #label, due:next week, has:file and is:overdue.
        </p>
      </div>
    </>
  );
}
