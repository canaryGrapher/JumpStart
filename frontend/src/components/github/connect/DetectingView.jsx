import Checklist from "./Checklist";

// While GitHubDetectRepo is in flight (and for a beat after), every row
// reads as "pending". Once the result lands, each row resolves to
// whatever it actually found — the point is to narrate what the app is
// checking, not to fake individual round trips for each line.
export default function DetectingView({ detection, loading }) {
  const st = (ok) => (loading || !detection ? "pending" : ok ? "done" : "warn");

  const items = [
    { key: "repo", label: "Checking for a Git repository", state: st(detection?.isGitRepo) },
    { key: "remote", label: "Looking for a configured remote", state: st(detection?.hasRemote) },
    { key: "github", label: "Confirming the remote points to GitHub", state: st(detection?.isGitHub) },
    {
      key: "exists",
      label: "Looking up the repository on GitHub",
      state: loading || !detection ? "pending" : detection.isGitHub && detection.repoExists ? "done" : "warn",
    },
  ];

  return (
    <div className="gh-detecting">
      <div className="gh-scan-badge">
        <span className="gh-scan-ring" />
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
          <path d="M9 19c-4.3 1.4-4.3-2.5-6-3m12 5v-3.5c0-1 .1-1.4-.5-2 2.8-.3 5.5-1.4 5.5-6a4.6 4.6 0 00-1.3-3.2 4.2 4.2 0 00-.1-3.2s-1-.3-3.3 1.3a11.5 11.5 0 00-6 0C6.9 2.6 5.9 2.9 5.9 2.9a4.2 4.2 0 00-.1 3.2A4.6 4.6 0 004.5 9.3c0 4.6 2.7 5.7 5.5 6-.6.6-.6 1.2-.5 2V21" />
        </svg>
      </div>
      <h3>Looking at this project</h3>
      <p className="gh-muted">Figuring out how it relates to GitHub, so you don't have to.</p>
      <Checklist items={items} />
    </div>
  );
}
