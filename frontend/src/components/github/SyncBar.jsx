import { useState } from "react";
import { BrowserOpenURL } from "../../api";
import GitHubConnectModal from "./connect/GitHubConnectModal";

const ago = (ts) => {
  if (!ts) return "never";
  const secs = Math.round((Date.now() - ts) / 1000);
  if (secs < 10) return "just now";
  if (secs < 60) return `${secs}s ago`;
  if (secs < 3600) return `${Math.round(secs / 60)}m ago`;
  return `${Math.round(secs / 3600)}h ago`;
};

// The strip above the board: whether it is linked, what the last pass
// did, and a way to force one. Both "link a board" and "settings" open
// the same intelligent connection modal — it already knows how to show
// an already-linked project, so there is no separate drawer to keep in
// sync with it.
export default function SyncBar({ projectId, sync, state, result, error, onSyncNow, onLinked, onError }) {
  const [open, setOpen] = useState(false);

  if (!sync?.enabled) {
    return (
      <div className="gh-bar">
        <span className="gh-bar-label">GitHub</span>
        <span className="gh-muted">Not linked to a board</span>
        <div className="spacer" />
        <button className="btn small" onClick={() => setOpen(true)}>
          Connect GitHub
        </button>
        {open && (
          <GitHubConnectModal
            projectId={projectId}
            sync={sync}
            syncState={state}
            onClose={() => setOpen(false)}
            onSyncNow={onSyncNow}
            onLinked={(cfg) => {
              setOpen(false);
              onLinked && onLinked(cfg);
            }}
            onError={onError}
          />
        )}
      </div>
    );
  }

  const conflicts = result?.conflicts || 0;

  return (
    <div className={`gh-bar linked ${state}`}>
      <div className="gh-bar-info">
        <span className={`gh-dot-state ${state}`} />
        <button className="gh-bar-link" onClick={() => sync.projectUrl && BrowserOpenURL(sync.projectUrl)}>
          {sync.projectTitle || "Board"}
        </button>

        {state === "syncing" && <span className="gh-muted">Syncing…</span>}
        {state === "error" && <span className="gh-warn-inline" title={error}>{error}</span>}
        {state === "idle" && (
          <span className="gh-muted">
            Synced {ago(sync.lastSyncAt || result?.at)}
            {result
              ? ` · ${result.pulled + result.created} in, ${result.pushed + result.uploaded} out`
              : ""}
          </span>
        )}

        {conflicts > 0 && (
          <span className="gh-badge conflict" title="Both sides changed since the last sync">
            {conflicts} conflict{conflicts > 1 ? "s" : ""}
          </span>
        )}
      </div>

      <div className="gh-bar-actions">
        {sync.projectUrl && (
          <button
            className="btn small ghost gh-manage-views"
            title="Add or rearrange board views on GitHub"
            onClick={() => BrowserOpenURL(sync.projectUrl)}
          >
            Manage views on GitHub ↗
          </button>
        )}
        <button className="btn small" disabled={state === "syncing"} onClick={onSyncNow}>
          Sync now
        </button>
        <button className="btn small ghost" onClick={() => setOpen(true)}>
          Settings
        </button>
      </div>

      {open && (
        <GitHubConnectModal
          projectId={projectId}
          sync={sync}
          syncState={state}
          syncError={error}
          onClose={() => setOpen(false)}
          onSyncNow={onSyncNow}
          onLinked={(cfg) => {
            onLinked && onLinked(cfg);
          }}
          onError={onError}
        />
      )}
    </div>
  );
}
