import { useState } from "react";
import { GitHubSetRemote } from "../../../api";

// Picking an existing repository is safe on its own — it only decides
// where issues open. Pointing this project's local git remote at it is
// a bigger deal (it can replace whatever "origin" is already set), so
// that step needs an explicit yes, never a silent side effect of
// "Continue" on the repo picker.
export default function ConfirmRemoteView({ projectId, fullName, onDone, onError }) {
  const [busy, setBusy] = useState(false);

  const linkRemote = async () => {
    setBusy(true);
    try {
      await GitHubSetRemote(projectId, fullName);
      onDone();
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="gh-step gh-confirm">
      <div className="gh-step-head">
        <h3>Link this project's Git remote too?</h3>
        <p className="gh-muted">
          <strong>{fullName}</strong> will be used for syncing issues either way. Setting it as
          this project's <code>origin</code> remote will replace any remote already configured.
        </p>
      </div>

      <div className="gh-actions-row">
        <button className="btn primary gh-cta" disabled={busy} onClick={linkRemote}>
          {busy ? "Linking…" : "Yes, link repo"}
        </button>
        <button className="btn small ghost" disabled={busy} onClick={onDone}>
          No, just sync issues
        </button>
      </div>
    </div>
  );
}
