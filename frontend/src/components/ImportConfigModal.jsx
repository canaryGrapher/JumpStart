import { useState } from "react";
import {
  ImportConfigText,
  PickConfigFile,
  ReadConfigFile,
  ClipboardSetText,
} from "../api";
import { buildConfPrompt } from "../confPrompt";
import BlockBuilder from "./import/BlockBuilder";
import { serialize, parse } from "./import/configModel";

// Import dialog: build the config as blocks or paste/edit raw JSON,
// optionally load it from a file, then import. Blocks and JSON stay in
// sync when you switch tabs.
export default function ImportConfigModal({ onClose, onInfo, onError, onReload }) {
  const [tab, setTab] = useState("blocks");
  const [projects, setProjects] = useState([]);
  const [jsonText, setJsonText] = useState(serialize([]));
  const [error, setError] = useState(null);
  const [busy, setBusy] = useState(false);

  const goJson = () => {
    setJsonText(serialize(projects));
    setError(null);
    setTab("json");
  };

  const goBlocks = () => {
    const { projects: parsed, error: err } = parse(jsonText);
    if (err) {
      setError(err);
      return;
    }
    setProjects(parsed);
    setError(null);
    setTab("blocks");
  };

  // Current config as text, from whichever tab is active.
  const currentText = () => (tab === "json" ? jsonText : serialize(projects));

  const loadFile = async () => {
    try {
      const path = await PickConfigFile();
      if (!path) return;
      const text = await ReadConfigFile(path);
      const { projects: parsed, error: err } = parse(text);
      setJsonText(text);
      if (err) {
        setError(err);
        setTab("json");
      } else {
        setProjects(parsed);
        setError(null);
        setTab("blocks");
      }
    } catch (e) {
      setError(String(e));
    }
  };

  const copyPrompt = async () => {
    try {
      await ClipboardSetText(buildConfPrompt());
      onInfo("Prompt copied. Paste it into your AI agent, then paste the result back here.");
    } catch (e) {
      onError(String(e));
    }
  };

  const doImport = async () => {
    setBusy(true);
    setError(null);
    try {
      const msg = await ImportConfigText(currentText());
      onInfo(msg);
      onReload();
      onClose();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal import-modal" onClick={(e) => e.stopPropagation()}>
        <h2>Import config</h2>
        <p className="sub">
          Build your projects as blocks, or paste the config JSON. A copy is saved with the
          app when you import.
        </p>

        <div className="import-toolbar">
          <div className="seg">
            <button className={tab === "blocks" ? "on" : ""} onClick={goBlocks}>
              Blocks
            </button>
            <button className={tab === "json" ? "on" : ""} onClick={goJson}>
              JSON
            </button>
          </div>
          <div className="import-toolbar-right">
            <button className="btn" onClick={loadFile}>
              Load from file
            </button>
            <button className="btn" onClick={copyPrompt}>
              Copy prompt
            </button>
          </div>
        </div>

        {tab === "blocks" ? (
          <BlockBuilder projects={projects} onChange={setProjects} />
        ) : (
          <textarea
            className="json-editor"
            spellCheck={false}
            value={jsonText}
            onChange={(e) => setJsonText(e.target.value)}
            placeholder='{ "projects": [ ... ] }'
          />
        )}

        {error && <span className="error">{error}</span>}

        <div className="modal-actions">
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn primary" onClick={doImport} disabled={busy}>
            {busy ? "Importing..." : "Import"}
          </button>
        </div>
      </div>
    </div>
  );
}
