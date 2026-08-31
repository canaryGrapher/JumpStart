import { useState } from "react";
import { GitHubCreateIssue } from "../../../api";
import { createPortal } from "react-dom";

// Minimal on purpose: a title and an optional body is everything
// createIssue needs, and the repository is already decided (it's the
// project's linked one), so there is nothing else to ask.
export default function NewIssueModal({ projectId, onClose, onCreated, onError }) {
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (!title.trim()) return;
    setBusy(true);
    try {
      const issue = await GitHubCreateIssue(projectId, title.trim(), body.trim());
      onCreated(issue);
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  return createPortal(
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal gh-mini-modal" onClick={(e) => e.stopPropagation()}>
        <h2>New issue</h2>

        <div className="field">
          <label>Title</label>
          <input
            autoFocus
            value={title}
            placeholder="Something isn't working"
            onChange={(e) => setTitle(e.target.value)}
          />
        </div>

        <div className="field">
          <label>Description</label>
          <textarea
            rows={6}
            value={body}
            placeholder="Steps to reproduce, expected behavior, anything that helps…"
            onChange={(e) => setBody(e.target.value)}
          />
        </div>

        <div className="modal-actions">
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn primary" disabled={busy || !title.trim()} onClick={submit}>
            {busy ? "Creating…" : "Create issue"}
          </button>
        </div>
      </div>
    </div>,
    document.body
  );
}
