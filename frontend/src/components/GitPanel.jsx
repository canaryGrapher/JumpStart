import { useEffect, useState } from "react";
import {
  GitStatus,
  GitInit,
  GitFetch,
  GitPull,
  GitPush,
  GitCommit,
  GitAddRemote,
  BrowserOpenURL,
} from "../api";
import ReleaseModal from "./ReleaseModal";
import BranchTimeline from "./git/BranchTimeline";
import BranchManager from "./git/BranchManager";
import DiffModal from "./git/DiffModal";

const fmtTime = (iso) => {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  return d.toLocaleString();
};

export default function GitPanel({ projectRoot, onError, onInfo }) {
  const [status, setStatus] = useState(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [remoteUrl, setRemoteUrl] = useState("");
  const [commitMsg, setCommitMsg] = useState("");
  const [showRelease, setShowRelease] = useState(false);
  const [showDiff, setShowDiff] = useState(false);
  const [gitKey, setGitKey] = useState(0);

  const bumpGit = () => setGitKey((k) => k + 1);

  const load = () => {
    setLoading(true);
    return GitStatus(projectRoot)
      .then(setStatus)
      .catch((e) => onError(String(e)))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectRoot]);

  const run = async (label, action, successMsg) => {
    if (busy) return;
    setBusy(true);
    try {
      const result = await action();
      onInfo(successMsg || label);
      await load();
      bumpGit();
      return result;
    } catch (e) {
      onError(`${label} failed: ${String(e)}`);
    } finally {
      setBusy(false);
    }
  };

  const doInit = () => run("Initialize repository", () => GitInit(projectRoot), "Repository initialized");
  const doFetch = () => run("Fetch", () => GitFetch(projectRoot), "Fetched latest refs");
  const doPull = () => run("Pull", () => GitPull(projectRoot), "Pulled successfully");
  const doPush = () => run("Push", () => GitPush(projectRoot), "Pushed successfully");

  const doAddRemote = () => {
    const url = remoteUrl.trim();
    if (!url) return;
    run("Add remote", () => GitAddRemote(projectRoot, url), "Remote added").then(() => setRemoteUrl(""));
  };

  const doCommit = () => {
    const msg = commitMsg.trim();
    if (!msg) return;
    run("Commit", () => GitCommit(projectRoot, msg), "Changes committed").then((hash) => {
      if (hash) setCommitMsg("");
    });
  };

  if (loading && !status) {
    return (
      <div className="panel git-panel">
        <h3>Git</h3>
        <div className="sub">Checking repository status…</div>
      </div>
    );
  }

  if (!status) return null;

  return (
    <div className="panel git-panel">
      <div className="git-panel-head">
        <h3>Git</h3>
        {status.initialized && (
          <span className={`status-pill ${status.clean ? "on" : "off"}`}>
            <span className="dot" />
            {status.clean ? "Clean" : "Dirty"}
          </span>
        )}
      </div>

      {!status.initialized ? (
        <>
          <div className="sub">This project folder is not a git repository yet.</div>
          <div className="actions">
            <button className="btn primary" disabled={busy} onClick={doInit}>
              Initialize repository
            </button>
          </div>
        </>
      ) : (
        <>
          <div className="git-meta">
            <div className="git-meta-row">
              <span className="git-meta-label">Branch</span>
              <span className="git-meta-value">{status.branch || "(unknown)"}</span>
            </div>
            {status.hasRemote ? (
              <div className="git-meta-row">
                <span className="git-meta-label">Remote</span>
                <button
                  className="git-remote-link"
                  onClick={() => BrowserOpenURL(status.remoteUrl)}
                  title="Open remote in browser"
                >
                  {status.remoteUrl}
                </button>
              </div>
            ) : (
              <div className="git-meta-row col">
                <span className="git-meta-label">Remote</span>
                <span className="row-hint">No remote configured yet.</span>
                <div className="row">
                  <input
                    placeholder="https://github.com/you/repo.git"
                    value={remoteUrl}
                    onChange={(e) => setRemoteUrl(e.target.value)}
                  />
                  <button className="btn small" disabled={busy || !remoteUrl.trim()} onClick={doAddRemote}>
                    Add remote
                  </button>
                </div>
              </div>
            )}
            <div className="git-meta-row">
              <span className="git-meta-label">Ahead / Behind</span>
              <span className="git-meta-value">
                <span className="kb-pill">{status.ahead || 0} ahead</span>{" "}
                <span className="kb-pill">{status.behind || 0} behind</span>
              </span>
            </div>
            {status.lastCommit && (
              <div className="git-meta-row col">
                <span className="git-meta-label">Last commit</span>
                <span className="row-hint">
                  {status.lastCommit} {status.lastCommitTime && `· ${fmtTime(status.lastCommitTime)}`}
                </span>
              </div>
            )}
          </div>

          <div className="actions">
            <button className="btn small" disabled={busy} onClick={doFetch}>
              Fetch
            </button>
            <button className="btn small" disabled={busy} onClick={doPull}>
              Pull
            </button>
            <button className="btn small" disabled={busy || !status.hasRemote} onClick={doPush}>
              Push
            </button>
            <button
              className="btn small ai"
              disabled={!status.hasRemote}
              onClick={() => setShowRelease(true)}
              title={status.hasRemote ? "Publish a release" : "Add a remote first"}
            >
              Create Release
            </button>
          </div>

          <div className="git-commit-row">
            <input
              placeholder="Commit message"
              value={commitMsg}
              onChange={(e) => setCommitMsg(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && doCommit()}
            />
            <button className="btn small primary" disabled={busy || !commitMsg.trim()} onClick={doCommit}>
              Commit
            </button>
          </div>

          <div className="git-section">
            <div className="git-section-head">
              <span className="git-meta-label">Branches</span>
            </div>
            <BranchManager
              projectRoot={projectRoot}
              current={status.branch}
              refreshKey={gitKey}
              busy={busy}
              onError={onError}
              onInfo={onInfo}
              onChanged={() => {
                load();
                bumpGit();
              }}
            />
          </div>

          <div className="git-section">
            <div className="git-section-head">
              <span className="git-meta-label">History</span>
              <button className="btn tiny" onClick={() => setShowDiff(true)}>
                View changes
              </button>
            </div>
            <BranchTimeline projectRoot={projectRoot} refreshKey={gitKey} onError={onError} />
          </div>
        </>
      )}

      {showDiff && <DiffModal projectRoot={projectRoot} onClose={() => setShowDiff(false)} />}

      {showRelease && (
        <ReleaseModal
          projectRoot={projectRoot}
          onClose={() => setShowRelease(false)}
          onError={onError}
          onInfo={onInfo}
        />
      )}
    </div>
  );
}
