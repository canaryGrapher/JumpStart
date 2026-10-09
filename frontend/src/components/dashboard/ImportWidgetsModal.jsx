import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { ParseWidgetImport } from "../../api";

// Paste or load a .jumpstart-widget.json file, review what it contains
// (code widgets are flagged), then add the widgets to the dashboard.
export default function ImportWidgetsModal({ onAdd, onClose }) {
  const [text, setText] = useState("");
  const [parsed, setParsed] = useState(null);
  const [error, setError] = useState("");

  const check = async (value = text) => {
    setError("");
    setParsed(null);
    try {
      setParsed(await ParseWidgetImport(value));
    } catch (e) {
      setError(String(e));
    }
  };

  const loadFile = (file) => {
    if (!file) return;
    if (file.size > 1024 * 1024) return setError("Widget files are limited to 1 MB.");
    const reader = new FileReader();
    reader.onload = () => {
      setText(String(reader.result || ""));
      check(String(reader.result || ""));
    };
    reader.readAsText(file);
  };

  useEffect(() => {
    const onKey = (e) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  // Portaled to <body>: the dashboard's animated container would otherwise
  // become the containing block for the fixed overlay.
  return createPortal(
    <div className="modal-overlay" onMouseDown={onClose}>
      <div className="modal we-modal" role="dialog" aria-label="Import widgets" onMouseDown={(e) => e.stopPropagation()}>
        <h2>Import widgets</h2>
        <p className="prefs-hint">Paste a widget file (.jumpstart-widget.json) or a single widget object, or choose a file.</p>
        <input type="file" accept=".json,application/json" aria-label="Widget file" onChange={(e) => loadFile(e.target.files?.[0])} />
        <textarea
          aria-label="Widget JSON"
          rows={10}
          spellCheck={false}
          value={text}
          onChange={(e) => {
            setText(e.target.value);
            setParsed(null);
          }}
          placeholder='{"jumpstartWidget": 1, "widgets": [ … ]}'
        />
        {error && <p className="we-error" role="alert">{error}</p>}
        {parsed && (
          <div className="we-import-preview">
            <strong>{parsed.widgets.length} widget{parsed.widgets.length === 1 ? "" : "s"}:</strong>
            <ul>
              {parsed.widgets.map((w) => (
                <li key={w.id}>
                  {w.title || w.type} <span className="prefs-hint">({w.type}{w.type === "html" ? ", contains code" : ""})</span>
                </li>
              ))}
            </ul>
            {parsed.warnings.map((w) => (
              <p key={w} className="we-warn">{w}</p>
            ))}
          </div>
        )}
        <div className="modal-actions">
          <span style={{ flex: 1 }} />
          <button type="button" className="btn" onClick={onClose}>Cancel</button>
          {!parsed ? (
            <button type="button" className="btn primary" disabled={!text.trim()} onClick={() => check()}>Check</button>
          ) : (
            <button type="button" className="btn primary" onClick={() => onAdd(parsed.widgets)}>
              Add {parsed.widgets.length} widget{parsed.widgets.length === 1 ? "" : "s"}
            </button>
          )}
        </div>
      </div>
    </div>,
    document.body
  );
}
