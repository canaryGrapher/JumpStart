import { useMemo, useState } from "react";
import { Badge, Button } from "@pikoloo/darwin-ui";
import { BrowserOpenURL } from "../../api";
import GitHubConnectModal from "./connect/GitHubConnectModal";
import ConflictsModal from "./ConflictsModal";
import { conflictReasonSummary } from "./conflictReasons";

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
export default function SyncBar({
  projectId,
  sync,
  state,
  result,
  error,
  progress,
  tasks = [],
  columns,
  onSyncNow,
  onLinked,
  onColumnsChange,
  onResolvedConflicts,
  onError,
}) {
  const [open, setOpen] = useState(false);
  const [conflictsOpen, setConflictsOpen] = useState(false);
  const conflicted = useMemo(
    () => (tasks || []).filter((t) => t.github?.conflict),
    [tasks]
  );

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
            columns={columns}
            onClose={() => setOpen(false)}
            onSyncNow={onSyncNow}
            onLinked={(cfg) => {
              setOpen(false);
              onLinked && onLinked(cfg);
            }}
            onColumnsChange={onColumnsChange}
            onError={onError}
          />
        )}
      </div>
    );
  }

  // Only live conflict badges — the last sync's conflicts count can
  // lag behind a Keep mine / Dismiss that already cleared the cards.
  const conflicts = conflicted.length;
  const conflictHint =
    conflicted.length === 1
      ? conflictReasonSummary(conflicted[0].github) || "1 field differs from GitHub"
      : `${conflicts} cards differ from GitHub`;
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
          <button
            type="button"
            className="gh-conflicts-badge-btn"
            title={conflictHint}
            onClick={() => setConflictsOpen(true)}
          >
            <Badge variant="warning">
              {conflicts} conflict{conflicts > 1 ? "s" : ""}
            </Badge>
          </button>
        )}
      </div>

      <div className="gh-bar-actions">
        {conflicts > 0 && (
          <Button
            size="sm"
            variant="secondary"
            glass
            title="Accept GitHub or Overwrite for many cards at once"
            onClick={() => setConflictsOpen(true)}
          >
            Resolve conflicts
          </Button>
        )}
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
          columns={columns}
          onClose={() => setOpen(false)}
          onSyncNow={onSyncNow}
          onLinked={(cfg) => {
            onLinked && onLinked(cfg);
          }}
          onColumnsChange={onColumnsChange}
          onError={onError}
        />
      )}

      {conflictsOpen && (
        <ConflictsModal
          projectId={projectId}
          tasks={tasks}
          onClose={() => setConflictsOpen(false)}
          onError={onError}
          onResolved={(ids, keepLocal) => {
            onResolvedConflicts && onResolvedConflicts(ids, keepLocal);
          }}
        />
      )}
    </div>
  );
}
