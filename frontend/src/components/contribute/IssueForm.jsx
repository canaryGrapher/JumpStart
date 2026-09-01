import { useState } from "react";
import { SubmitIssue, CollectDiagnostics, BrowserOpenURL } from "../../api";

const KINDS = [
  { id: "bug", label: "Bug report" },
  { id: "feature", label: "Feature request" },
];

// In-app GitHub issue composer. Only rendered when a GitHub token is stored;
// the caller handles the disconnected case.
export default function IssueForm({ onCancel, onError }) {
  const [kind, setKind] = useState("bug");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [labels, setLabels] = useState("");
  const [attachLogs, setAttachLogs] = useState(false);
  const [busy, setBusy] = useState(false);
  const [created, setCreated] = useState(null);

  const submit = async () => {
    if (!title.trim()) return;
    setBusy(true);
    try {
      // Diagnostics are only read when the user has ticked the box, so
      // nothing leaves the machine without explicit consent.
      const diagnostics = attachLogs ? await CollectDiagnostics() : "";
      const url = await SubmitIssue({
        kind,
        title: title.trim(),
        description,
        labels: labels
          .split(",")
          .map((l) => l.trim())
          .filter(Boolean),
        diagnostics,
      });
      setCreated(url);
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  if (created) {
    return (
      <div className="prefs-row col">
        <span className="ai-status ok">Issue created.</span>
        <div className="prefs-actions">
          <button className="btn primary" onClick={() => BrowserOpenURL(created)}>
            View Issue
          </button>
          <button className="btn" onClick={onCancel}>
            Done
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="contrib-form">
      <div className="prefs-row col">
        <label>Type</label>
        <div className="prefs-seg" role="group" aria-label="Issue type">
          {KINDS.map((k) => (
            <button
              key={k.id}
              type="button"
              className={`prefs-seg-btn ${kind === k.id ? "active" : ""}`}
              onClick={() => setKind(k.id)}
            >
              {k.label}
            </button>
          ))}
        </div>
      </div>

      <div className="prefs-row col">
        <label>Title</label>
        <input
          value={title}
          placeholder={kind === "bug" ? "Short summary of the problem" : "What should JumpStart do?"}
          onChange={(e) => setTitle(e.target.value)}
        />
      </div>

      <div className="prefs-row col">
        <label>Description</label>
        <textarea
          className="contrib-desc"
          rows={7}
          value={description}
          placeholder={
            kind === "bug"
              ? "What happened, what you expected, and the steps to reproduce it."
              : "The problem you're trying to solve and how you'd like it to work."
          }
          onChange={(e) => setDescription(e.target.value)}
        />
      </div>

      <div className="prefs-row col">
        <label>Labels (optional)</label>
        <input
          value={labels}
          placeholder="comma separated, e.g. ui, git"
          onChange={(e) => setLabels(e.target.value)}
        />
        <span className="row-hint">
          {kind === "bug" ? '"bug"' : '"enhancement"'} is added automatically.
        </span>
      </div>

      <div className="prefs-row col">
        <label className="contrib-check">
          <input
            type="checkbox"
            checked={attachLogs}
            onChange={(e) => setAttachLogs(e.target.checked)}
          />
          Attach diagnostics
        </label>
        <span className="row-hint">
          Includes recent output from your running processes. Review it first if those logs
          might contain anything private. Your app version, OS, and architecture are always
          included.
        </span>
      </div>

      <div className="prefs-actions contrib-actions">
        <button className="btn primary" disabled={busy || !title.trim()} onClick={submit}>
          {busy ? "Submitting…" : "Submit Issue"}
        </button>
        <button className="btn" disabled={busy} onClick={onCancel}>
          Cancel
        </button>
      </div>
    </div>
  );
}
