import { useState } from "react";
import { Badge, Button } from "@pikoloo/darwin-ui";
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
export default function SyncBar({ projectId, sync, state, result, error, progress, onSyncNow, onLinked, onError }) {
  const [open, setOpen] = useState(false);

  if (!sync?.enabled) {
    return (
      <div className="gh-bar">
        <span className="gh-bar-label">GitHub</span>
        <span className="gh-muted">Not linked to a board</span>
        <div className="spacer" />
        <Button size="sm" variant="secondary" glass onClick={() => setOpen(true)}>
          Connect GitHub
        </Button>
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
  const syncingLabel =
    progress?.total > 0 && progress.done >= 1
      ? `Syncing ${progress.done}/${progress.total} tasks`
      : "Syncing…";

  return (
    <div className={`gh-bar linked ${state}`}>
      <div className="gh-bar-info">
        <span className={`gh-dot-state ${state}`} />
        <button className="gh-bar-link" onClick={() => sync.projectUrl && BrowserOpenURL(sync.projectUrl)}>
          {sync.projectTitle || "Board"}
        </button>

        {state === "syncing" && <span className="gh-muted">{syncingLabel}</span>}
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
          <Badge variant="warning" title="Both sides changed since the last sync">
            {conflicts} conflict{conflicts > 1 ? "s" : ""}
          </Badge>
        )}
      </div>

      <div className="gh-bar-actions">
        {sync.projectUrl && (
          <Button
            size="sm"
            variant="ghost"
            title="Add or rearrange board views on GitHub"
            onClick={() => BrowserOpenURL(sync.projectUrl)}
          >
            Manage views on GitHub ↗
          </Button>
        )}
        <Button
          size="sm"
          variant="secondary"
          glass
          disabled={state === "syncing"}
          loading={state === "syncing"}
          loadingText="Syncing…"
          onClick={onSyncNow}
        >
          Sync now
        </Button>
        <Button size="sm" variant="ghost" onClick={() => setOpen(true)}>
          Settings
        </Button>
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
