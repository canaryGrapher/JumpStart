import { BrowserOpenURL } from "../../../api";

const ago = (iso) => {
  if (!iso) return "";
  const secs = Math.round((Date.now() - new Date(iso).getTime()) / 1000);
  if (secs < 60) return "just now";
  if (secs < 3600) return `${Math.round(secs / 60)}m ago`;
  if (secs < 86400) return `${Math.round(secs / 3600)}h ago`;
  return `${Math.round(secs / 86400)}d ago`;
};

// One row shared by the Issues and Pull Requests lists. `state` drives
// the status dot's color; PRs pass a couple of extra bits (draft,
// branches) that issues just leave undefined.
export default function ActivityRow({ item, isDraft, branches }) {
  const state = isDraft ? "draft" : (item.state || "").toLowerCase();
  return (
    <button className="gh-activity-row" onClick={() => BrowserOpenURL(item.url)}>
      <span className={`gh-dot-state ${state}`} />
      <span className="gh-activity-title">{item.title}</span>
      <span className="gh-num">#{item.number}</span>
      {branches && (
        <span className="gh-muted gh-activity-branches">
          {item.headRefName} → {item.baseRefName}
        </span>
      )}
      <div className="spacer" />
      {item.author && <span className="gh-muted">{item.author}</span>}
      {item.comments > 0 && (
        <span className="gh-muted gh-activity-comments">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M21 11.5a8.4 8.4 0 01-.9 3.8 8.5 8.5 0 01-7.6 4.7 8.4 8.4 0 01-3.8-.9L3 21l1.9-5.7a8.4 8.4 0 01-.9-3.8 8.5 8.5 0 014.7-7.6 8.4 8.4 0 013.8-.9h.5a8.48 8.48 0 018 8v.5z" />
          </svg>
          {item.comments}
        </span>
      )}
      <span className="gh-muted gh-activity-time">{ago(item.updatedAt)}</span>
    </button>
  );
}
