import { useEffect, useState } from "react";
import {
  ExportProjectJSON,
  PreviewProjectJSON,
  ApplyProjectJSON,
  SaveTextFile,
  ClipboardSetText,
} from "../api";

const slug = (s) => (s || "project").toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");

// Export a project's full JSON, or edit/paste JSON and merge it back in.
// Imports always show a preview of added, updated and removed tasks first.
export default function ProjectJsonModal({ project, onClose, onApplied, onError }) {
  const [tab, setTab] = useState("export");
  const [includeEnv, setIncludeEnv] = useState(false);
  const [exported, setExported] = useState("");
  const [text, setText] = useState("");
  const [mode, setMode] = useState("merge");
  const [preview, setPreview] = useState(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState("");

  useEffect(() => {
    let live = true;
    ExportProjectJSON(project.id, includeEnv)
      .then((j) => {
        if (!live) return;
        setExported(j);
        setText((t) => t || j);
      })
      .catch((e) => live && setError(String(e)));
    return () => {
      live = false;
    };
  }, [project.id, includeEnv]);

  useEffect(() => {
    const onKey = (e) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  // Any edit invalidates the last preview.
  useEffect(() => setPreview(null), [text, mode]);

  const doPreview = async () => {
    setError("");
    setBusy(true);
    try {
      const pv = (await PreviewProjectJSON(project.id, text, mode)) || {};
      const list = (x) => (Array.isArray(x) ? x : []);
      setPreview({
        added: list(pv.added), updated: list(pv.updated), removed: list(pv.removed),
        project: list(pv.project), sprints: list(pv.sprints), warnings: list(pv.warnings), unchanged: pv.unchanged || 0,
      });
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const apply = async () => {
    setBusy(true);
    setError("");
    try {
      const pv = await ApplyProjectJSON(project.id, text, mode);
      onApplied && onApplied(pv);
      onClose();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const saveFile = async () => {
    try {
      const path = await SaveTextFile(`${slug(project.name)}.jumpstart.json`, exported, "Export project JSON");
      if (path) setNote(`Saved to ${path}`);
    } catch (e) {
      onError && onError(String(e));
    }
  };
  const copy = async () => {
    try {
      await ClipboardSetText(exported);
      setNote("Copied to the clipboard.");
    } catch (e) {
      onError && onError(String(e));
    }
  };

  const changes = preview
    ? preview.added.length + preview.updated.length + preview.removed.length + preview.project.length + preview.sprints.length
    : 0;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal import-modal project-json" role="dialog" aria-label="Project JSON" onClick={(e) => e.stopPropagation()}>
        <h2>Project JSON</h2>
        <p className="sub">
          Every field of “{project.name}” and its tasks. Edit and import to update tasks or add new ones in bulk.
        </p>

        <div className="import-toolbar">
          <div className="seg">
            <button className={tab === "export" ? "on" : ""} onClick={() => setTab("export")}>
              Export
            </button>
            <button className={tab === "import" ? "on" : ""} onClick={() => setTab("import")}>
              Import / Update
            </button>
          </div>
          {tab === "export" ? (
            <div className="import-toolbar-right">
              <label className="check-row">
                <input type="checkbox" checked={includeEnv} onChange={(e) => setIncludeEnv(e.target.checked)} />
                Include environment values
              </label>
              <button className="btn" onClick={copy}>
                Copy
              </button>
              <button className="btn" onClick={saveFile}>
                Save file…
              </button>
            </div>
          ) : (
            <div className="import-toolbar-right">
              <div className="seg" role="radiogroup" aria-label="Import mode">
                <button className={mode === "merge" ? "on" : ""} onClick={() => setMode("merge")} title="Update tasks by id and add new ones">
                  Add &amp; update
                </button>
                <button className={mode === "replace" ? "on" : ""} onClick={() => setMode("replace")} title="Also remove tasks missing from the JSON">
                  Replace
                </button>
              </div>
              <button className="btn" onClick={() => setText(exported)}>
                Reset to current
              </button>
            </div>
          )}
        </div>

        {tab === "export" ? (
          <>
            {includeEnv && (
              <p className="warn-note">Environment values often contain secrets. Share this export carefully.</p>
            )}
            <textarea className="json-editor" spellCheck={false} readOnly value={exported} aria-label="Exported JSON" />
          </>
        ) : (
          <>
            <textarea
              className="json-editor"
              spellCheck={false}
              value={text}
              onChange={(e) => setText(e.target.value)}
              aria-label="JSON to import"
            />
            {preview && (
              <div className="json-preview" aria-live="polite">
                {changes === 0 ? (
                  <p>No changes: everything already matches.</p>
                ) : (
                  <ul>
                    {preview.project.length > 0 && <li>Project: {preview.project.join(", ")}</li>}
                    {preview.sprints.length > 0 && <li>New sprints: {preview.sprints.join(", ")}</li>}
                    {preview.added.length > 0 && <li className="add">Add {preview.added.length}: {preview.added.join(", ")}</li>}
                    {preview.updated.length > 0 && (
                      <li className="upd">
                        Update {preview.updated.length}:{" "}
                        {preview.updated.map((u) => `${u.title} (${u.fields.join(", ")})`).join("; ")}
                      </li>
                    )}
                    {preview.removed.length > 0 && <li className="del">Remove {preview.removed.length}: {preview.removed.join(", ")}</li>}
                  </ul>
                )}
                {preview.unchanged > 0 && <p className="sub">{preview.unchanged} unchanged.</p>}
                {preview.warnings.map((w) => (
                  <p className="warn-note" key={w}>
                    {w}
                  </p>
                ))}
              </div>
            )}
          </>
        )}

        {error && <span className="error">{error}</span>}
        {note && <span className="sub">{note}</span>}

        <div className="modal-actions">
          <button className="btn" onClick={onClose}>
            {tab === "export" ? "Close" : "Cancel"}
          </button>
          {tab === "import" &&
            (preview ? (
              <button className="btn primary" onClick={apply} disabled={busy || changes === 0}>
                {busy ? "Applying…" : `Apply ${changes} change${changes === 1 ? "" : "s"}`}
              </button>
            ) : (
              <button className="btn primary" onClick={doPreview} disabled={busy || !text.trim()}>
                {busy ? "Checking…" : "Preview changes"}
              </button>
            ))}
        </div>
      </div>
    </div>
  );
}
