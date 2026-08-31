// The unified Git Changes & Branch Management modal: branches on the
// left, staged/unstaged changes with a diff viewer and commit box on the
// right. Switching branches, staging files, and committing all refresh
// through the same loadAll() so the two panels never show stale state
// relative to each other.
import { useCallback, useEffect, useState } from "react";
import {
  GitStatus,
  GitListBranches,
  GitWorkingChanges,
  GitCheckout,
  GitCreateBranch,
  GitDeleteBranch,
  GitStageFile,
  GitUnstageFile,
  GitStageAll,
  GitUnstageAll,
  GitRemoveIndexLock,
  GitCommit,
  GitPush,
  GitPublishBranch,
} from "../../api";
import BranchListPanel from "./BranchListPanel";
import FileList from "./FileList";
import FileDiffPane from "./FileDiffPane";
import CommitPanel from "./CommitPanel";
import GitStatusBanners from "./GitStatusBanners";

export default function GitChangesModal({ projectRoot, onClose, onError, onInfo }) {
  const [status, setStatus] = useState(null);
  const [branches, setBranches] = useState([]);
  const [changes, setChanges] = useState([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [selection, setSelection] = useState(null); // { path, staged }
  const [pushErr, setPushErr] = useState(null);
  // Set whenever a mutating action fails on a stale .git/index.lock, so
  // the modal can offer a one-click "remove it and retry" recovery
  // instead of just showing the raw git error in the toast.
  const [lockErr, setLockErr] = useState(null);

  const loadAll = useCallback(async () => {
    const [s, b, c] = await Promise.all([
      GitStatus(projectRoot),
      GitListBranches(projectRoot).catch(() => []),
      GitWorkingChanges(projectRoot).catch(() => []),
    ]);
    setStatus(s);
    setBranches(b || []);
    setChanges(c || []);
  }, [projectRoot]);

  useEffect(() => {
    setLoading(true);
    loadAll()
      .catch((e) => onError(String(e)))
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectRoot]);

  const refresh = () => loadAll().catch((e) => onError(String(e)));

  // Shared runner: every mutating action goes through this so busy state,
  // the success toast, and the post-action refresh stay in one place
  // instead of being repeated at each call site.
  const run = async (label, action, successMsg) => {
    setBusy(true);
    try {
      const result = await action();
      setLockErr(null);
      if (successMsg) onInfo(successMsg);
      await refresh();
      return result;
    } catch (e) {
      const msg = String(e);
      if (msg.toLowerCase().includes("index.lock")) {
        // Keep the retry action around so "Remove Lock File" can redo
        // whatever just failed, not just clear the index.
        setLockErr({ message: msg, retry: () => run(label, action, successMsg) });
      }
      onError(`${label} failed: ${msg}`);
      throw e;
    } finally {
      setBusy(false);
    }
  };

  // Deletes .git/index.lock (refused by the backend if it looks fresh
  // enough to belong to a git process that's still running) and then
  // replays whatever action tripped over it.
  const removeLock = async () => {
    setBusy(true);
    try {
      await GitRemoveIndexLock(projectRoot);
      onInfo("Removed stale lock file");
      const retry = lockErr?.retry;
      setLockErr(null);
      if (retry) await retry();
    } catch (e) {
      onError(`Remove lock file failed: ${String(e)}`);
    } finally {
      setBusy(false);
    }
  };

  const unstaged = changes.filter((c) => c.unstaged);
  const staged = changes.filter((c) => c.staged);
  const hasConflicts = changes.some((c) => c.conflicted);

  const selectFile = (path, isStaged) => setSelection({ path, staged: isStaged });

  const doSwitch = (name) =>
    run("Switch branch", () => GitCheckout(projectRoot, name), `Switched to ${name}`)
      .then(() => setSelection(null))
      .catch(() => {});
  const doCreate = (name) =>
    run("Create branch", () => GitCreateBranch(projectRoot, name, true), `Created ${name}`).catch(() => {});
  const doDeleteBranch = (name) =>
    run("Delete branch", () => GitDeleteBranch(projectRoot, name, false), `Deleted ${name}`).catch(() => {});

  // Staging/unstaging a file moves it to the other section; if it was
  // the one selected for the diff pane, follow it there instead of
  // leaving the pane pointed at a side that no longer shows the file.
  const stageOne = (path) =>
    run("Stage", () => GitStageFile(projectRoot, path))
      .then(() => setSelection((sel) => (sel?.path === path ? { path, staged: true } : sel)))
      .catch(() => {});
  const unstageOne = (path) =>
    run("Unstage", () => GitUnstageFile(projectRoot, path))
      .then(() => setSelection((sel) => (sel?.path === path ? { path, staged: false } : sel)))
      .catch(() => {});
  const stageAll = () =>
    run("Stage all", () => GitStageAll(projectRoot), "Staged all changes")
      .then(() => setSelection((sel) => (sel ? { path: sel.path, staged: true } : sel)))
      .catch(() => {});
  const unstageAll = () =>
    run("Unstage all", () => GitUnstageAll(projectRoot), "Unstaged all changes")
      .then(() => setSelection((sel) => (sel ? { path: sel.path, staged: false } : sel)))
      .catch(() => {});

  const doCommit = async (message) => {
    try {
      const hash = await run("Commit", () => GitCommit(projectRoot, message), "Changes committed");
      setSelection(null);
      return hash;
    } catch {
      return null;
    }
  };

  const doPush = async () => {
    setPushErr(null);
    try {
      await run("Push", () => GitPush(projectRoot), "Pushed successfully");
    } catch (e) {
      setPushErr(String(e));
    }
  };

  const doPublish = async () => {
    setPushErr(null);
    try {
      await run("Publish branch", () => GitPublishBranch(projectRoot, status.branch), "Branch published");
    } catch (e) {
      setPushErr(String(e));
    }
  };

  if (loading || !status) {
    return (
      <div className="modal-overlay" onClick={onClose}>
        <div className="modal git-changes-modal" onClick={(e) => e.stopPropagation()}>
          <div className="git-changes-loading">Loading repository…</div>
        </div>
      </div>
    );
  }

  const nothingToShow = staged.length === 0 && unstaged.length === 0;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal git-changes-modal" onClick={(e) => e.stopPropagation()}>
        <div className="git-changes-head">
          <div className="git-changes-head-title">
            <h2>Git Changes</h2>
            <span className="git-changes-branch-pill" title="Current branch">
              {status.branch || (status.detachedHead ? "detached HEAD" : "—")}
            </span>
            <span className={`status-pill ${status.clean ? "on" : "off"}`}>
              <span className="dot" />
              {status.clean ? "Clean" : "Uncommitted changes"}
            </span>
          </div>
          <button type="button" className="btn small" onClick={onClose}>
            Close
          </button>
        </div>

        <GitStatusBanners status={status} hasConflicts={hasConflicts} pushError={pushErr} />

        {lockErr && (
          <div className="git-banners">
            <div className="git-banner err git-banner-lock">
              <span>
                A leftover <code>.git/index.lock</code> file is blocking git — this usually means a git
                process (or JumpStart itself) was interrupted mid-operation.
              </span>
              <button type="button" className="btn small" disabled={busy} onClick={removeLock}>
                Remove Lock File
              </button>
            </div>
          </div>
        )}

        <div className="git-changes-body">
          <BranchListPanel
            branches={branches}
            current={status.branch}
            dirty={!status.clean}
            busy={busy}
            onSwitch={doSwitch}
            onCreate={doCreate}
            onDelete={doDeleteBranch}
          />

          <div className="git-changes-panel">
            {nothingToShow ? (
              <div className="git-changes-clean">Nothing to commit — the working tree is clean.</div>
            ) : (
              <div className="git-changes-files">
                <FileList
                  title="Unstaged Changes"
                  files={unstaged}
                  staged={false}
                  busy={busy}
                  selectedPath={selection?.path}
                  selectedStaged={selection?.staged}
                  onSelect={selectFile}
                  onToggle={stageOne}
                  onBulkAction={stageAll}
                  bulkLabel="Stage All"
                  emptyLabel="No unstaged changes."
                />
                <FileList
                  title="Staged Changes"
                  files={staged}
                  staged
                  busy={busy}
                  selectedPath={selection?.path}
                  selectedStaged={selection?.staged}
                  onSelect={selectFile}
                  onToggle={unstageOne}
                  onBulkAction={unstageAll}
                  bulkLabel="Unstage All"
                  emptyLabel="No staged changes yet."
                />
              </div>
            )}

            <FileDiffPane projectRoot={projectRoot} selection={selection} />
          </div>
        </div>

        <div className="git-changes-footer">
          <CommitPanel projectRoot={projectRoot} stagedCount={staged.length} busy={busy} onCommit={doCommit} />
          <div className="git-push-row">
            {!status.hasRemote ? (
              <span className="row-hint">No remote configured — add one from the Git panel to push.</span>
            ) : hasConflicts ? (
              <span className="row-hint">Resolve conflicts before pushing.</span>
            ) : status.hasUpstream ? (
              <button type="button" className="btn primary" disabled={busy} onClick={doPush}>
                Push Changes{status.ahead ? ` (${status.ahead})` : ""}
              </button>
            ) : (
              <button type="button" className="btn primary" disabled={busy} onClick={doPublish}>
                Publish Branch
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
