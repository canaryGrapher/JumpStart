import { useState } from "react";
import { CreateRelease, BrowserOpenURL } from "../api";

export default function ReleaseModal({ projectRoot, onClose, onError, onInfo }) {
  const [tagName, setTagName] = useState("");
  const [name, setName] = useState("");
  const [body, setBody] = useState("");
  const [draft, setDraft] = useState(false);
  const [prerelease, setPrerelease] = useState(false);
  const [busy, setBusy] = useState(false);
  const [releaseUrl, setReleaseUrl] = useState("");
  const [error, setError] = useState("");

  const valid = tagName.trim() && name.trim();

  const submit = async () => {
    if (!valid || busy) return;
    setBusy(true);
    setError("");
    try {
      const url = await CreateRelease(projectRoot, {
        tagName: tagName.trim(),
        name: name.trim(),
        body,
        draft,
        prerelease,
      });
      setReleaseUrl(url);
      onInfo && onInfo("Release published");
    } catch (e) {
      const msg = String(e);
      setError(msg);
      onError && onError(`Release failed: ${msg}`);
    } finally {
      setBusy(false);
    }
  };

  const onKeyDown = (e) => {
    if (e.key === "Escape") onClose();
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()} onKeyDown={onKeyDown}>
        <h2>Create Release</h2>

        {releaseUrl ? (
          <>
            <p className="sub">Release published successfully.</p>
            <button className="git-remote-link" onClick={() => BrowserOpenURL(releaseUrl)}>
              {releaseUrl}
            </button>
            <div className="modal-actions">
              <button className="btn primary" onClick={onClose}>
                Done
              </button>
            </div>
          </>
        ) : (
          <>
            <div className="field">
              <label htmlFor="rel-tag">Tag name</label>
              <input
                id="rel-tag"
                placeholder="v1.0.0"
                value={tagName}
                onChange={(e) => setTagName(e.target.value)}
                autoFocus
              />
            </div>

            <div className="field">
              <label htmlFor="rel-name">Release name</label>
              <input
                id="rel-name"
                placeholder="Version 1.0.0"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>

            <div className="field">
              <label htmlFor="rel-body">Notes</label>
              <textarea
                id="rel-body"
                rows={6}
                placeholder="What's new in this release..."
                value={body}
                onChange={(e) => setBody(e.target.value)}
              />
            </div>

            <label className="check-row">
              <input type="checkbox" checked={draft} onChange={(e) => setDraft(e.target.checked)} />
              Draft
            </label>
            <label className="check-row">
              <input
                type="checkbox"
                checked={prerelease}
                onChange={(e) => setPrerelease(e.target.checked)}
              />
              Pre-release
            </label>

            {error && (
              <span className="error">
                {error}
                {/error|token|auth/i.test(error) && (
                  <> — set a token for this provider in Preferences → Git integration.</>
                )}
              </span>
            )}

            <div className="modal-actions">
              <button className="btn" onClick={onClose}>
                Cancel
              </button>
              <button className="btn primary" disabled={!valid || busy} onClick={submit}>
                {busy ? "Publishing…" : "Publish Release"}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
