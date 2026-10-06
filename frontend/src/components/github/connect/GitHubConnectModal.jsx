import { useMemo, useState } from "react";
import useDetection from "./useDetection";
import DetectingView from "./DetectingView";
import ExistingView from "./ExistingView";
import CreateView from "./CreateView";
import ConnectView from "./ConnectView";
import ConfirmRemoteView from "./ConfirmRemoteView";
import { createPortal } from "react-dom";

// The GitHub connection experience for a project: on open it inspects
// the local git state and the token's view of GitHub, then lands on
// whichever of three things is actually true — already connected and
// ready to sync, a local project waiting to be pushed up, or no GitHub
// account at all — instead of asking the user to fill out a form that
// describes what the app could have found out for itself.
export default function GitHubConnectModal({
  projectId,
  sync,
  syncState,
  syncError,
  columns,
  onClose,
  onLinked,
  onColumnsChange,
  onSyncNow,
  onError,
}) {
  const { detection, loading, error: detectError, refresh } = useDetection(projectId);
  const [createdRepo, setCreatedRepo] = useState(null);
  // Set while an existing repository has been picked but the user has
  // not yet said whether it should also become this project's git
  // remote. Cleared (and folded into createdRepo) once they answer.
  const [pendingRepo, setPendingRepo] = useState(null);
  const [localError, setLocalError] = useState(null);

  const fail = (msg) => {
    setLocalError(msg);
    onError && onError(msg);
  };

  const handleLinked = (cfg) => {
    setCreatedRepo(null);
    onLinked && onLinked(cfg);
    if (cfg) onClose && onClose();
  };

  const handleContinueExisting = (fullName) => {
    // The picker resolved an existing repository the account can see.
    // Before folding it into the "existing" card, ask whether it should
    // also become this project's git remote — that step can replace an
    // existing one, so it is never automatic.
    const [owner, repo] = fullName.split("/");
    setPendingRepo({ owner, repo, fullName, url: `https://github.com/${fullName}` });
  };

  const settlePendingRepo = () => {
    setCreatedRepo(pendingRepo);
    setPendingRepo(null);
  };

  const phase = useMemo(() => {
    if (pendingRepo) return "confirm-remote";
    if (createdRepo) return "existing";
    if (sync?.enabled) return "existing";
    if (loading || !detection) return "detecting";
    if (detection.isGitHub && detection.repoExists) return "existing";
    if (!detection.connected) return "connect";
    return "create";
  }, [pendingRepo, createdRepo, sync?.enabled, loading, detection]);

  const existingProps = createdRepo
    ? createdRepo
    : sync?.enabled
    ? {
        owner: sync.owner || (sync.repo || "").split("/")[0],
        repo: (sync.repo || "").split("/")[1],
        fullName: sync.repo,
        url: sync.repo ? `https://github.com/${sync.repo}` : null,
      }
    : {
        owner: detection?.owner,
        repo: detection?.repo,
        fullName: detection?.fullName,
        url: detection?.repoUrl,
      };

  return createPortal(
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal gh-connect-modal" onClick={(e) => e.stopPropagation()}>
        <div className="gh-modal-head">
          <span className="gh-modal-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 .5C5.7.5.5 5.7.5 12c0 5 3.3 9.3 7.9 10.8.6.1.8-.3.8-.6v-2.2c-3.2.7-3.9-1.4-3.9-1.4-.5-1.3-1.3-1.7-1.3-1.7-1.1-.7.1-.7.1-.7 1.2.1 1.8 1.2 1.8 1.2 1 1.8 2.7 1.3 3.4 1 .1-.8.4-1.3.7-1.6-2.6-.3-5.3-1.3-5.3-5.7 0-1.3.4-2.3 1.2-3.1-.1-.3-.5-1.5.1-3 0 0 1-.3 3.3 1.2a11.2 11.2 0 015.9 0c2.3-1.5 3.3-1.2 3.3-1.2.6 1.5.2 2.7.1 3 .8.8 1.2 1.9 1.2 3.1 0 4.4-2.7 5.4-5.3 5.7.4.4.8 1.1.8 2.2v3.3c0 .3.2.7.8.6A11.5 11.5 0 0023.5 12C23.5 5.7 18.3.5 12 .5z" />
            </svg>
          </span>
          <h2>GitHub</h2>
          <button className="icon-btn gh-modal-close" onClick={onClose} aria-label="Close">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
              <path d="M6 6l12 12M18 6L6 18" />
            </svg>
          </button>
        </div>

        <div className="gh-modal-body">
          {phase === "detecting" && <DetectingView detection={detection} loading={loading} />}

          {phase === "existing" && (
            <ExistingView
              projectId={projectId}
              sync={createdRepo ? null : sync}
              syncState={syncState}
              syncError={createdRepo ? "" : syncError}
              columns={columns}
              onLinked={handleLinked}
              onColumnsChange={onColumnsChange}
              onSyncNow={onSyncNow}
              onError={fail}
              owner={existingProps.owner}
              repo={existingProps.repo}
              fullName={existingProps.fullName}
              repoUrl={existingProps.url}
            />
          )}

          {phase === "create" && (
            <CreateView
              detection={detection}
              projectId={projectId}
              isGitRepo={detection?.isGitRepo}
              onCreated={(repo) =>
                setCreatedRepo({ owner: repo.owner, repo: repo.name, fullName: repo.fullName, url: repo.url })
              }
              onContinue={handleContinueExisting}
              onError={fail}
            />
          )}

          {phase === "connect" && (
            <ConnectView onConnected={refresh} onContinue={handleContinueExisting} onError={fail} />
          )}

          {phase === "confirm-remote" && (
            <ConfirmRemoteView
              projectId={projectId}
              fullName={pendingRepo.fullName}
              onDone={settlePendingRepo}
              onError={fail}
            />
          )}

          {(localError || detectError) && <span className="error">{localError || detectError}</span>}
        </div>
      </div>
    </div>,
    document.body
  );
}
