import { useEffect, useMemo, useState } from "react";
import { GitHubListBranches, GitHubCreatePullRequest } from "../../../api";
import BranchStatus from "./BranchStatus";
import { createPortal } from "react-dom";

const optionLabel = (b) => {
  const where = b.local && b.remote ? "local & GitHub" : b.local ? "local only" : "GitHub only";
  return `${b.name}${b.current ? " (current)" : ""} — ${where}`;
};

// Base and head are pickers built from every branch JumpStart can see —
// local, GitHub, or both — rather than only what happens to be checked
// out here. GitHub needs both ends to already exist on the remote, so a
// local-only branch is still listed (picking one explains why the form
// won't submit) but never silently sent as-is.
export default function NewPullRequestModal({ projectId, onClose, onCreated, onError }) {
  const [branches, setBranches] = useState(null);
  const [base, setBase] = useState("");
  const [head, setHead] = useState("");
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [draft, setDraft] = useState(false);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    GitHubListBranches(projectId)
      .then((list) => {
        setBranches(list || []);
        const onGitHub = (list || []).filter((b) => b.remote);
        const current = onGitHub.find((b) => b.current) || (list || []).find((b) => b.current);
        const trunk = onGitHub.find((b) => b.name === "main" || b.name === "master");
        setHead(current?.name || onGitHub[0]?.name || list?.[0]?.name || "");
        setBase(trunk?.name || onGitHub.find((b) => b.name !== current?.name)?.name || "");
      })
      .catch((e) => onError && onError(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  const byName = useMemo(() => {
    const m = {};
    for (const b of branches || []) m[b.name] = b;
    return m;
  }, [branches]);

  const headBranch = byName[head];
  const baseBranch = byName[base];
  const readyToSubmit =
    title.trim() && base && head && base !== head && headBranch?.remote && baseBranch?.remote;

  const submit = async () => {
    if (!readyToSubmit) return;
    setBusy(true);
    try {
      const pr = await GitHubCreatePullRequest(projectId, base, head, title.trim(), body.trim(), draft);
      onCreated(pr);
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  return createPortal(
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal gh-mini-modal" onClick={(e) => e.stopPropagation()}>
        <h2>New pull request</h2>

        {branches && branches.length === 0 ? (
          <p className="gh-muted">No branches found, locally or on GitHub.</p>
        ) : (
          <>
            <div className="field">
              <label>Head branch (your changes)</label>
              <select value={head} onChange={(e) => setHead(e.target.value)} disabled={!branches}>
                {(branches || []).map((b) => (
                  <option key={b.name} value={b.name}>
                    {optionLabel(b)}
                  </option>
                ))}
              </select>
              <BranchStatus branch={headBranch} />
              {headBranch && !headBranch.remote && (
                <span className="hint">Push this branch to GitHub before opening a pull request from it.</span>
              )}
            </div>

            <div className="field">
              <label>Base branch (merge into)</label>
              <select value={base} onChange={(e) => setBase(e.target.value)} disabled={!branches}>
                {(branches || []).map((b) => (
                  <option key={b.name} value={b.name}>
                    {optionLabel(b)}
                  </option>
                ))}
              </select>
              <BranchStatus branch={baseBranch} />
              {base && head && base === head && <span className="hint">Pick two different branches.</span>}
            </div>

            <div className="field">
              <label>Title</label>
              <input value={title} placeholder="Short summary of the change" onChange={(e) => setTitle(e.target.value)} />
            </div>

            <div className="field">
              <label>Description</label>
              <textarea
                rows={6}
                value={body}
                placeholder="What changed and why…"
                onChange={(e) => setBody(e.target.value)}
              />
            </div>

            <label className="gh-check">
              <input type="checkbox" checked={draft} onChange={(e) => setDraft(e.target.checked)} />
              Open as draft
            </label>
          </>
        )}

        <div className="modal-actions">
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn primary" disabled={busy || !readyToSubmit} onClick={submit}>
            {busy ? "Creating…" : "Create pull request"}
          </button>
        </div>
      </div>
    </div>,
    document.body
  );
}
