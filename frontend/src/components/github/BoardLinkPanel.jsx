import { useEffect, useState } from "react";
import {
  GitHubListProjects,
  GitHubLinkProject,
  GitHubUnlinkProject,
  GitHubSuggestRepo,
  BrowserOpenURL,
} from "../../api";
import StatusMapEditor from "./StatusMapEditor";

// Pick the GitHub Projects v2 board this JumpStart project syncs with.
// The repo defaults to whatever the project's git remote points at, so
// linking is usually two clicks.
export default function BoardLinkPanel({ projectId, sync, onLinked, onError }) {
  const [owner, setOwner] = useState("");
  const [boards, setBoards] = useState([]);
  const [selected, setSelected] = useState(sync?.projectId || "");
  const [repo, setRepo] = useState(sync?.repo || "");
  const [asIssue, setAsIssue] = useState(!!sync?.createAsIssue);
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(false);

  const load = (login) => {
    setLoading(true);
    GitHubListProjects(login || "")
      .then((b) => setBoards(b || []))
      .catch((e) => onError && onError(String(e)))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load("");
    GitHubSuggestRepo(projectId)
      .then((r) => r && setRepo((cur) => cur || r))
      .catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  const link = async () => {
    if (!selected) return;
    setBusy(true);
    try {
      const cfg = await GitHubLinkProject(projectId, selected, repo.trim(), asIssue);
      onLinked && onLinked(cfg);
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setBusy(false);
    }
  };

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

  if (sync?.enabled) {
    return (
      <div className="gh-linked">
        <div className="gh-linked-head">
          <div>
            <strong>{sync.projectTitle || "Linked board"}</strong>
            <span className="gh-muted">
              {sync.owner ? `${sync.owner} · ` : ""}#{sync.projectNumber}
            </span>
          </div>
          <div className="spacer" />
          {sync.projectUrl && (
            <button className="btn small" onClick={() => BrowserOpenURL(sync.projectUrl)}>
              Open board
            </button>
          )}
          <button className="btn small danger" disabled={busy} onClick={unlink}>
            Unlink
          </button>
        </div>
        {sync.repo && (
          <div className="gh-muted">New issues open in {sync.repo}</div>
        )}
        <StatusMapEditor projectId={projectId} sync={sync} onError={onError} />
      </div>
    );
  }

  return (
    <div className="gh-link">
      <div className="field">
        <label>Owner</label>
        <div className="row">
          <input
            value={owner}
            placeholder="Your account, or an org login"
            onChange={(e) => setOwner(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && load(owner.trim())}
          />
          <button className="btn small" onClick={() => load(owner.trim())}>
            Load boards
          </button>
        </div>
      </div>

      <div className="field">
        <label>Board</label>
        {loading ? (
          <span className="gh-muted">Loading boards…</span>
        ) : (
          <select value={selected} onChange={(e) => setSelected(e.target.value)}>
            <option value="">Pick a Projects board…</option>
            {boards.map((b) => (
              <option key={b.id} value={b.id}>
                #{b.number} {b.title}
                {b.closed ? " (closed)" : ""}
              </option>
            ))}
          </select>
        )}
        {!loading && boards.length === 0 && (
          <span className="gh-muted">
            No boards found for that owner. Projects v2 boards live under the
            account or organization, not the repository.
          </span>
        )}
      </div>

      <div className="field">
        <label>Repository for new issues</label>
        <input
          value={repo}
          placeholder="owner/name"
          onChange={(e) => setRepo(e.target.value)}
        />
        <label className="gh-check">
          <input
            type="checkbox"
            checked={asIssue}
            onChange={(e) => setAsIssue(e.target.checked)}
          />
          Create real issues instead of drafts
        </label>
      </div>

      <button className="btn primary" disabled={busy || !selected} onClick={link}>
        {busy ? "Linking…" : "Link and sync"}
      </button>
    </div>
  );
}
