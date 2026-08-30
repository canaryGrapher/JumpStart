import { useEffect, useState } from "react";
import {
  GitHubGetProject,
  GitHubSetFieldValue,
  GitHubResolveConflict,
  BrowserOpenURL,
} from "../../api";
import { NATIVE, blankValue } from "./fieldTypes";
import FieldEditor from "./FieldEditor";

// The GitHub section of the task modal: the link to the issue or draft,
// then an editor for every field on the board. Field writes go straight
// to GitHub rather than waiting for the task's Save, because a board
// field is remote state and the round trip is what confirms it landed.
export default function GitHubFields({ task, sync, projectId, onError }) {
  const [board, setBoard] = useState(null);
  const [values, setValues] = useState(task.fields || {});
  const [busy, setBusy] = useState("");

  useEffect(() => {
    if (!sync?.enabled || !sync.projectId) return;
    GitHubGetProject(sync.projectId)
      .then(setBoard)
      .catch((e) => onError && onError(String(e)));
  }, [sync?.projectId, sync?.enabled, onError]);

  useEffect(() => {
    setValues(task.fields || {});
  }, [task.id, task.fields]);

  if (!sync?.enabled) return null;

  const link = task.github;

  const commit = async (field, next) => {
    setValues((v) => ({ ...v, [field.id]: next }));
    if (!link?.itemId) return;
    setBusy(field.id);
    try {
      await GitHubSetFieldValue(projectId, task.id, next);
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy("");
    }
  };

  const resolve = async (keepLocal) => {
    try {
      await GitHubResolveConflict(projectId, task.id, keepLocal);
    } catch (e) {
      onError && onError(String(e));
    }
  };

  // Status is driven by the Kanban column, so editing it here would give
  // the user two controls that fight each other.
  const editable = (board?.fields || []).filter(
    (f) => !NATIVE.has(f.dataType) && f.id !== sync.statusFieldId
  );

  return (
    <div className="gh-panel">
      <div className="gh-panel-head">
        <span className="gh-panel-title">GitHub</span>
        {link?.conflict && (
          <span className="gh-badge conflict" title="Local and GitHub both changed since the last sync">
            Conflict
          </span>
        )}
        {link?.pending && <span className="gh-badge pending">Not pushed yet</span>}
        {!link?.itemId && <span className="gh-muted">Syncs on the next pass</span>}
      </div>

      {link?.itemId && (
        <div className="gh-link-row">
          <span className={`gh-badge type-${(link.contentType || "").toLowerCase()}`}>
            {link.contentType === "DraftIssue" ? "Draft" : link.contentType}
          </span>
          {link.number > 0 && <span className="gh-num">#{link.number}</span>}
          {link.repo && <span className="gh-muted">{link.repo}</span>}
          {link.state && <span className={`gh-state ${link.state.toLowerCase()}`}>{link.state}</span>}
          {link.url && (
            <button className="btn small" onClick={() => BrowserOpenURL(link.url)}>
              Open on GitHub
            </button>
          )}
        </div>
      )}

      {link?.conflict && (
        <div className="gh-conflict-row">
          <span className="gh-muted">
            This card changed here and on GitHub since the last sync. The newer
            edit was kept.
          </span>
          <div className="row">
            <button className="btn small" onClick={() => resolve(true)}>
              Keep mine, push it
            </button>
            <button className="btn small ghost" onClick={() => resolve(false)}>
              Dismiss
            </button>
          </div>
        </div>
      )}

      {!board && <div className="gh-muted">Loading board fields…</div>}

      {editable.map((f) => (
        <div key={f.id} className={busy === f.id ? "gh-busy" : ""}>
          <FieldEditor
            field={f}
            value={values[f.id] || blankValue(f)}
            disabled={busy === f.id || !link?.itemId}
            onChange={(next) => commit(f, next)}
          />
        </div>
      ))}
    </div>
  );
}
