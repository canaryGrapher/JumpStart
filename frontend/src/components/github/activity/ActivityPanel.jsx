import { useEffect, useState } from "react";
import { GitHubListIssues, GitHubListPullRequests } from "../../../api";
import ActivityRow from "./ActivityRow";
import NewIssueModal from "./NewIssueModal";
import NewPullRequestModal from "./NewPullRequestModal";

const STATE_FILTERS = ["OPEN", "CLOSED"];

// Issues and pull requests from the project's linked repository, right
// alongside the board they're tracked next to. Collapsed by default so
// the Kanban board stays the main event; expanding it lazy-loads
// whichever tab is showing.
export default function ActivityPanel({ projectId, sync, onError }) {
  const [expanded, setExpanded] = useState(false);
  const [tab, setTab] = useState("issues");
  const [issues, setIssues] = useState(null);
  const [prs, setPrs] = useState(null);
  const [onlyOpen, setOnlyOpen] = useState(true);
  const [loading, setLoading] = useState(false);
  const [newIssueOpen, setNewIssueOpen] = useState(false);
  const [newPrOpen, setNewPrOpen] = useState(false);

  const states = onlyOpen ? ["OPEN"] : STATE_FILTERS.concat("MERGED");

  const load = (which) => {
    setLoading(true);
    const call = which === "prs" ? GitHubListPullRequests(projectId, states, 30) : GitHubListIssues(projectId, states, 30);
    call
      .then((rows) => (which === "prs" ? setPrs(rows || []) : setIssues(rows || [])))
      .catch((e) => onError && onError(String(e)))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    if (!expanded) return;
    load(tab);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [expanded, tab, onlyOpen]);

  if (!sync?.enabled) return null;

  const rows = tab === "prs" ? prs : issues;

  return (
    <div className="gh-activity">
      <button className="gh-activity-head" onClick={() => setExpanded((v) => !v)}>
        <span className={`gh-chevron ${expanded ? "open" : ""}`}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M9 6l6 6-6 6" />
          </svg>
        </span>
        <span className="gh-activity-title-lg">Issues &amp; Pull Requests</span>
        <span className="gh-muted">{sync.repo}</span>
      </button>

      {expanded && (
        <div className="gh-activity-body">
          <div className="gh-activity-tabs">
            <button className={tab === "issues" ? "on" : ""} onClick={() => setTab("issues")}>
              Issues
            </button>
            <button className={tab === "prs" ? "on" : ""} onClick={() => setTab("prs")}>
              Pull requests
            </button>

            <div className="spacer" />

            <label className="gh-check gh-open-toggle">
              <input type="checkbox" checked={onlyOpen} onChange={(e) => setOnlyOpen(e.target.checked)} />
              Open only
            </label>

            <button className="icon-btn" title="Refresh" onClick={() => load(tab)}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M23 4v6h-6M1 20v-6h6" />
                <path d="M3.5 9a9 9 0 0114.7-3.4L23 10M1 14l4.8 4.4A9 9 0 0020.5 15" />
              </svg>
            </button>

            <button className="btn small primary" onClick={() => (tab === "prs" ? setNewPrOpen(true) : setNewIssueOpen(true))}>
              {tab === "prs" ? "New pull request" : "New issue"}
            </button>
          </div>

          <div className="gh-activity-list">
            {loading && <span className="gh-muted gh-activity-loading">Loading…</span>}
            {!loading && rows && rows.length === 0 && (
              <span className="gh-muted gh-activity-loading">
                No {onlyOpen ? "open " : ""}
                {tab === "prs" ? "pull requests" : "issues"}.
              </span>
            )}
            {!loading &&
              rows &&
              rows.map((item) => (
                <ActivityRow key={item.id} item={item} isDraft={item.isDraft} branches={tab === "prs"} />
              ))}
          </div>
        </div>
      )}

      {newIssueOpen && (
        <NewIssueModal
          projectId={projectId}
          onClose={() => setNewIssueOpen(false)}
          onError={onError}
          onCreated={() => {
            setNewIssueOpen(false);
            setTab("issues");
            setExpanded(true);
            load("issues");
          }}
        />
      )}

      {newPrOpen && (
        <NewPullRequestModal
          projectId={projectId}
          onClose={() => setNewPrOpen(false)}
          onError={onError}
          onCreated={() => {
            setNewPrOpen(false);
            setTab("prs");
            setExpanded(true);
            load("prs");
          }}
        />
      )}
    </div>
  );
}
