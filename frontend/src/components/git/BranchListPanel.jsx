import { useMemo, useState } from "react";
import ConfirmInline from "./ConfirmInline";

// Left panel of the Git Changes modal: search, the current branch
// pinned visually via the "current" styling, local branches you can
// switch to or delete, and read-only remote-tracking branches below.
// Switching while the working tree is dirty asks for confirmation
// instead of silently discarding anything — git itself will still
// refuse a switch that would overwrite local edits.
export default function BranchListPanel({ branches, current, dirty, busy, onSwitch, onCreate, onDelete }) {
  const [query, setQuery] = useState("");
  const [confirmSwitch, setConfirmSwitch] = useState(null);
  const [deleteTarget, setDeleteTarget] = useState(null);
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState("");

  const locals = useMemo(() => branches.filter((b) => !b.remote), [branches]);
  const remotes = useMemo(() => branches.filter((b) => b.remote), [branches]);

  const q = query.trim().toLowerCase();
  const matches = (b) => !q || b.name.toLowerCase().includes(q);
  const filteredLocals = locals.filter(matches);
  const filteredRemotes = remotes.filter(matches);

  const requestSwitch = (name) => {
    if (name === current || busy) return;
    setDeleteTarget(null);
    if (dirty) {
      setConfirmSwitch(name);
    } else {
      onSwitch(name);
    }
  };

  const confirmAndSwitch = () => {
    const name = confirmSwitch;
    setConfirmSwitch(null);
    onSwitch(name);
  };

  const requestDelete = (name, e) => {
    e.stopPropagation();
    setConfirmSwitch(null);
    setDeleteTarget(name);
  };

  const confirmAndDelete = () => {
    const name = deleteTarget;
    setDeleteTarget(null);
    onDelete(name);
  };

  const submitCreate = () => {
    const name = newName.trim();
    if (!name) return;
    onCreate(name);
    setNewName("");
    setCreating(false);
  };

  return (
    <div className="git-branch-panel">
      <div className="git-branch-panel-head">
        <span className="git-panel-title">Branches</span>
        <button
          type="button"
          className="btn tiny"
          disabled={busy}
          onClick={() => setCreating((v) => !v)}
        >
          {creating ? "Cancel" : "+ New"}
        </button>
      </div>

      {creating && (
        <div className="git-branch-create">
          <input
            autoFocus
            placeholder="feature/my-branch"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && submitCreate()}
            disabled={busy}
          />
          <button className="btn tiny primary" disabled={busy || !newName.trim()} onClick={submitCreate}>
            Create
          </button>
        </div>
      )}

      {locals.length + remotes.length > 6 && (
        <input
          className="git-branch-search"
          placeholder="Search branches…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
      )}

      {branches.length === 0 ? (
        <div className="git-branch-empty">No branches found.</div>
      ) : (
        <div className="git-branch-list">
          <div className="git-branch-group-label">Local</div>
          {filteredLocals.length === 0 && <div className="git-branch-empty">No local branches match.</div>}
          {filteredLocals.map((b) => (
            <div key={`l:${b.name}`} className="git-branch-item">
              <button
                type="button"
                className={`git-branch-row ${b.current ? "current" : ""}`}
                disabled={busy}
                onClick={() => requestSwitch(b.name)}
                title={b.subject}
              >
                <span className="git-branch-dot" />
                <span className="git-branch-name">{b.name}</span>
                {b.current && dirty && (
                  <span className="git-branch-dirty-tag" title="Uncommitted changes">
                    ●
                  </span>
                )}
                {b.current && <span className="git-branch-current-tag">Current</span>}
                {!b.current && (
                  <span
                    className="git-branch-delete"
                    role="button"
                    tabIndex={0}
                    title={`Delete ${b.name}`}
                    onClick={(e) => requestDelete(b.name, e)}
                  >
                    ×
                  </span>
                )}
              </button>
              {confirmSwitch === b.name && (
                <ConfirmInline
                  message={`You have uncommitted changes. Switch to "${b.name}" anyway?`}
                  confirmLabel="Switch"
                  busy={busy}
                  onConfirm={confirmAndSwitch}
                  onCancel={() => setConfirmSwitch(null)}
                />
              )}
              {deleteTarget === b.name && (
                <ConfirmInline
                  message={`Delete "${b.name}"? This cannot be undone.`}
                  confirmLabel="Delete"
                  danger
                  busy={busy}
                  onConfirm={confirmAndDelete}
                  onCancel={() => setDeleteTarget(null)}
                />
              )}
            </div>
          ))}

          {remotes.length > 0 && (
            <>
              <div className="git-branch-group-label">Remote</div>
              {filteredRemotes.length === 0 && (
                <div className="git-branch-empty">No remote branches match.</div>
              )}
              {filteredRemotes.map((b) => (
                <div key={`r:${b.name}`} className="git-branch-row remote" title={b.subject}>
                  <span className="git-branch-dot remote" />
                  <span className="git-branch-name">{b.name}</span>
                </div>
              ))}
            </>
          )}
        </div>
      )}
    </div>
  );
}
