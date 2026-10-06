import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import {
  CreateBoardColumn,
  GitHubGetProject,
} from "../../api";

// Add a Kanban column and optionally wire it to a GitHub Status option
// (pick an existing one, or create a new status on the linked board).
export default function AddColumnModal({
  projectId,
  sync,
  onCreated,
  onClose,
  onError,
}) {
  const [label, setLabel] = useState("");
  const [mode, setMode] = useState("none"); // none | existing | create
  const [optionId, setOptionId] = useState("");
  const [newStatusName, setNewStatusName] = useState("");
  const [options, setOptions] = useState([]);
  const [busy, setBusy] = useState(false);
  const linked = !!sync?.enabled && !!sync?.projectId;

  useEffect(() => {
    if (!linked) return;
    GitHubGetProject(sync.projectId)
      .then((b) => {
        const f =
          (b.fields || []).find((x) => x.id === sync.statusFieldId) ||
          (b.fields || []).find(
            (x) => x.dataType === "SINGLE_SELECT" && /^status$/i.test(x.name || "")
          );
        setOptions(f?.options || []);
      })
      .catch((e) => onError && onError(String(e)));
  }, [linked, sync?.projectId, sync?.statusFieldId, onError]);

  useEffect(() => {
    if (mode === "create" && !newStatusName.trim() && label.trim()) {
      setNewStatusName(label.trim());
    }
  }, [mode, label, newStatusName]);

  const submit = async () => {
    const name = label.trim();
    if (!name || busy) return;
    if (mode === "existing" && !optionId) return;
    if (mode === "create" && !newStatusName.trim()) return;

    setBusy(true);
    try {
      const result = await CreateBoardColumn(
        projectId,
        name,
        mode === "existing" ? optionId : "",
        mode === "create" ? newStatusName.trim() : "",
        ""
      );
      onCreated && onCreated(result);
      onClose && onClose();
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  return createPortal(
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal gh-mini-modal"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        <h2>New column</h2>

        <div className="field">
          <label>Column name</label>
          <input
            autoFocus
            value={label}
            placeholder="e.g. Ready for review"
            disabled={busy}
            onChange={(e) => setLabel(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                submit();
              }
            }}
          />
        </div>

        {linked ? (
          <div className="field">
            <label>GitHub status</label>
            <select
              value={mode === "none" ? "" : mode === "create" ? "__create__" : optionId}
              disabled={busy}
              onChange={(e) => {
                const v = e.target.value;
                if (!v) {
                  setMode("none");
                  setOptionId("");
                  return;
                }
                if (v === "__create__") {
                  setMode("create");
                  setOptionId("");
                  if (!newStatusName.trim()) setNewStatusName(label.trim());
                  return;
                }
                setMode("existing");
                setOptionId(v);
              }}
            >
              <option value="">Don’t map yet</option>
              {(options || []).map((o) => (
                <option key={o.id} value={o.id}>
                  {o.name}
                </option>
              ))}
              <option value="__create__">Create new status…</option>
            </select>
          </div>
        ) : (
          <p className="gh-muted">
            Link a GitHub board later to map this column to a Status option.
          </p>
        )}

        {mode === "create" && (
          <div className="field">
            <label>New status name on GitHub</label>
            <input
              value={newStatusName}
              disabled={busy}
              placeholder="Status name on the board"
              onChange={(e) => setNewStatusName(e.target.value)}
            />
          </div>
        )}

        <div className="modal-actions">
          <button className="btn" onClick={onClose} disabled={busy}>
            Cancel
          </button>
          <button
            className="btn primary"
            disabled={
              busy ||
              !label.trim() ||
              (mode === "existing" && !optionId) ||
              (mode === "create" && !newStatusName.trim())
            }
            onClick={submit}
          >
            {busy ? "Adding…" : "Add column"}
          </button>
        </div>
      </div>
    </div>,
    document.body
  );
}
