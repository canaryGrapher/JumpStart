import { useMemo, useState } from "react";
import { createPortal } from "react-dom";
import { GitHubResolveConflicts } from "../../api";
import { conflictReasonSummary, formatConflictLine } from "./conflictReasons";

// Bulk resolve for cards where local pending edits and GitHub diverge.
// Same choices as the per-task row (Accept GitHub / Overwrite GitHub).
export default function ConflictsModal({
  projectId,
  tasks,
  onResolved,
  onClose,
  onError,
}) {
  const conflicted = useMemo(
    () => (tasks || []).filter((t) => t.github?.conflict),
    [tasks]
  );
  const [selected, setSelected] = useState(() =>
    Object.fromEntries(conflicted.map((t) => [t.id, true]))
  );
  const [busy, setBusy] = useState("");

  const ids = conflicted.filter((t) => selected[t.id]).map((t) => t.id);
  const allSelected = conflicted.length > 0 && ids.length === conflicted.length;

  const toggleAll = () => {
    if (allSelected) {
      setSelected({});
      return;
    }
    setSelected(Object.fromEntries(conflicted.map((t) => [t.id, true])));
  };

  const resolve = async (keepLocal) => {
    if (!ids.length || busy) return;
    setBusy(keepLocal ? "keep" : "accept");
    try {
      const n = await GitHubResolveConflicts(projectId, ids, keepLocal);
      onResolved && onResolved(ids, keepLocal, n);
      onClose && onClose();
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy("");
    }
  };

  return createPortal(
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal gh-conflicts-modal"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="gh-conflicts-title"
      >
        <h2 id="gh-conflicts-title">Resolve conflicts</h2>
        <p className="gh-muted">
          These cards have local edits that differ from GitHub. Accept GitHub
          to pull remote (source of truth), or Overwrite GitHub to push your
          local copy.
        </p>

        {conflicted.length === 0 ? (
          <p className="gh-muted">No conflicts right now.</p>
        ) : (
          <>
            <div className="gh-conflicts-toolbar">
              <label className="gh-conflicts-check">
                <input
                  type="checkbox"
                  checked={allSelected}
                  onChange={toggleAll}
                  disabled={!!busy}
                />
                Select all ({conflicted.length})
              </label>
              <span className="gh-muted">{ids.length} selected</span>
            </div>
            <ul className="gh-conflicts-list">
              {conflicted.map((t) => (
                <li key={t.id}>
                  <label className="gh-conflicts-check">
                    <input
                      type="checkbox"
                      checked={!!selected[t.id]}
                      disabled={!!busy}
                      onChange={(e) =>
                        setSelected((s) => ({ ...s, [t.id]: e.target.checked }))
                      }
                    />
                    <span className="gh-conflicts-main">
                      <span className="gh-conflicts-title">
                        {t.title || "Untitled"}
                      </span>
                      {t.github?.number > 0 && (
                        <span className="gh-muted">#{t.github.number}</span>
                      )}
                      {(t.github?.conflictFields || []).length > 0 ? (
                        <ul className="gh-conflict-reasons compact">
                          {(t.github.conflictFields || []).slice(0, 3).map((f) => (
                            <li key={f.field || f.label}>{formatConflictLine(f)}</li>
                          ))}
                          {(t.github.conflictFields || []).length > 3 && (
                            <li className="gh-muted">
                              +{t.github.conflictFields.length - 3} more
                            </li>
                          )}
                        </ul>
                      ) : (
                        <span className="gh-conflicts-why gh-muted">
                          {conflictReasonSummary(t.github) || "Fields differ"}
                        </span>
                      )}
                    </span>
                  </label>
                </li>
              ))}
            </ul>
          </>
        )}

        <div className="modal-actions">
          <button className="btn" onClick={onClose} disabled={!!busy}>
            Cancel
          </button>
          <button
            className="btn ghost"
            disabled={!!busy || !ids.length}
            onClick={() => resolve(true)}
          >
            {busy === "keep" ? "…" : "Overwrite GitHub"}
          </button>
          <button
            className="btn primary"
            disabled={!!busy || !ids.length}
            onClick={() => resolve(false)}
          >
            {busy === "accept" ? "…" : "Accept GitHub"}
          </button>
        </div>
      </div>
    </div>,
    document.body
  );
}
