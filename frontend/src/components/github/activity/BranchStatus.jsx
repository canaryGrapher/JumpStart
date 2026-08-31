// Small chip strip under a branch picker: whether the selected branch
// exists locally, on GitHub, or both, and — when it's both — whether
// the two copies agree. This is the detail a plain <select> can't carry
// in its own option text.
const STATUS_LABEL = {
  "in-sync": "In sync",
  "local-ahead": "Local has unpushed changes",
  "remote-ahead": "GitHub has changes you don't have locally",
  diverged: "Diverged from GitHub",
};

export default function BranchStatus({ branch }) {
  if (!branch) return null;
  return (
    <div className="gh-branch-status">
      <span className={`gh-branch-chip ${branch.local ? "on" : "off"}`}>Local</span>
      <span className={`gh-branch-chip ${branch.remote ? "on" : "off"}`}>GitHub</span>
      {branch.local && branch.remote && branch.status && (
        <span className={`gh-branch-sync ${branch.status}`}>{STATUS_LABEL[branch.status] || branch.status}</span>
      )}
      {branch.local && !branch.remote && (
        <span className="gh-muted">Not pushed to GitHub yet</span>
      )}
      {!branch.local && branch.remote && (
        <span className="gh-muted">Not checked out locally</span>
      )}
    </div>
  );
}
