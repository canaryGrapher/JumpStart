import { useState } from "react";
import { GitHubUnlinkProject, BrowserOpenURL } from "../../../api";
import StatusMapEditor from "../StatusMapEditor";
import SyncWizard from "./SyncWizard";

// A sync error this specific means the linked board itself is gone —
// deleted on GitHub, or the token lost visibility into it — as opposed
// to a transient network/rate-limit failure that a plain retry fixes.
const looksLikeMissingBoard = (msg) => /not found|cannot see it|no longer (exists|accessible)|was deleted/i.test(msg || "");

// The repository is already known — either detected from the git remote
// or just created — so the only real decision left is which Projects
// board it syncs with. That decision (account, board, repo link,
// import, layout, review) is delegated entirely to SyncWizard; this
// view just decides *when* to show it: never yet linked, or the linked
// board has gone missing on GitHub's side and needs recovering. Once
// linked, this is the "manage" view: unlink, re-sync, remap columns.
export default function ExistingView({
  projectId,
  owner,
  repo,
  fullName,
  repoUrl,
  sync,
  onLinked,
  onSyncNow,
  onError,
  syncState,
  syncError,
}) {
  const boardMissing = !!sync?.enabled && syncState === "error" && looksLikeMissingBoard(syncError);
  const [busy, setBusy] = useState(false);

  const unlink = async () => {
    setBusy(true);
    try {
      await GitHubUnlinkProject(projectId);
      onLinked && onLinked(null);
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="gh-step gh-existing">
      <div className="gh-repo-card">
        <div className="gh-repo-card-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            <path d="M4 19.5A2.5 2.5 0 016.5 17H20 M6.5 2H20v20H6.5A2.5 2.5 0 014 19.5v-15A2.5 2.5 0 016.5 2z" />
          </svg>
        </div>
        <div className="gh-repo-card-body">
          <span className="gh-repo-card-title">
            {owner}
            <span className="gh-repo-card-sep">/</span>
            {repo}
          </span>
          <span className="gh-muted">GitHub repository {sync?.enabled ? "linked" : "found"}</span>
        </div>
        {repoUrl && (
          <button className="btn small ghost" onClick={() => BrowserOpenURL(repoUrl)}>
            Open
          </button>
        )}
      </div>

      <ul className="gh-checklist gh-checklist-static">
        <li className="gh-check-item done">
          <span className="gh-check-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round">
              <path d="M20 6L9 17l-5-5" />
            </svg>
          </span>
          <span className="gh-check-label">Git repository detected</span>
        </li>
        <li className="gh-check-item done">
          <span className="gh-check-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round">
              <path d="M20 6L9 17l-5-5" />
            </svg>
          </span>
          <span className="gh-check-label">GitHub remote connected</span>
        </li>
      </ul>

      {boardMissing ? (
        <div className="gh-board-missing">
          <div className="gh-warn">
            "{sync.projectTitle || "This board"}" could not be found on GitHub. It may have been deleted, or the
            connected token can no longer see it.
          </div>
          <SyncWizard
            projectId={projectId}
            owner={owner}
            repo={repo}
            fullName={fullName}
            initialTitle={sync.projectTitle || repo}
            onLinked={onLinked}
            onError={onError}
          />
          <button className="gh-link-btn" disabled={busy} onClick={unlink}>
            Unlink this project instead
          </button>
        </div>
      ) : sync?.enabled ? (
        <>
          <div className="gh-sync-status">
            <span className={`gh-dot-state ${syncState || "idle"}`} />
            <span className="gh-muted">{sync.projectTitle || "Board"} · #{sync.projectNumber}</span>
          </div>
          <div className="gh-actions-row">
            <button className="btn primary gh-cta" disabled={busy || syncState === "syncing"} onClick={onSyncNow}>
              {syncState === "syncing" ? "Syncing…" : "Sync Now"}
            </button>
            {sync.projectUrl && (
              <button className="btn small ghost" onClick={() => BrowserOpenURL(sync.projectUrl)}>
                Open board
              </button>
            )}
            <button className="btn small danger" disabled={busy} onClick={unlink}>
              Unlink
            </button>
          </div>
          <StatusMapEditor projectId={projectId} sync={sync} onError={onError} />
        </>
      ) : (
        <SyncWizard
          projectId={projectId}
          owner={owner}
          repo={repo}
          fullName={fullName}
          onLinked={onLinked}
          onError={onError}
        />
      )}
    </div>
  );
}
