import { useEffect, useState } from "react";
import { GitFileDiff } from "../../api";

const lineClass = (line) => {
  if (line.startsWith("+++") || line.startsWith("---")) return "meta";
  if (line.startsWith("@@")) return "hunk";
  if (line.startsWith("diff ") || line.startsWith("index ")) return "file";
  if (line.startsWith("+")) return "add";
  if (line.startsWith("-")) return "del";
  return "";
};

// The right-hand diff viewer. selection is { path, staged } for whichever
// file was last clicked in either file list; staged/unstaged context
// travels with it so the header can say which side is being reviewed
// (staging vs. working tree) without the user losing their place.
export default function FileDiffPane({ projectRoot, selection }) {
  const [patch, setPatch] = useState("");
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState("");

  useEffect(() => {
    if (!selection) {
      setPatch("");
      setErr("");
      return;
    }
    let alive = true;
    setLoading(true);
    setErr("");
    GitFileDiff(projectRoot, selection.path, selection.staged)
      .then((p) => alive && setPatch(p || ""))
      .catch((e) => alive && setErr(String(e)))
      .finally(() => alive && setLoading(false));
    return () => {
      alive = false;
    };
  }, [projectRoot, selection]);

  if (!selection) {
    return (
      <div className="git-diff-pane git-diff-pane-empty">
        <span className="row-hint">Select a file to review its changes.</span>
      </div>
    );
  }

  const lines = patch ? patch.split("\n") : [];
  const isEmpty = lines.length === 0 || (lines.length === 1 && lines[0] === "");

  return (
    <div className="git-diff-pane">
      <div className="git-diff-pane-head">
        <span className="git-diff-pane-path" title={selection.path}>
          {selection.path}
        </span>
        <span className={`git-diff-pane-tag ${selection.staged ? "staged" : "unstaged"}`}>
          {selection.staged ? "Staged" : "Unstaged"}
        </span>
      </div>
      {loading ? (
        <div className="git-diff-pane-status">Loading diff…</div>
      ) : err ? (
        <div className="error">{err}</div>
      ) : isEmpty ? (
        <div className="git-diff-pane-status">No line changes to show for this file.</div>
      ) : (
        <pre className="git-diff-patch">
          {lines.map((l, i) => (
            <div key={i} className={`git-diff-line ${lineClass(l)}`}>
              {l || " "}
            </div>
          ))}
        </pre>
      )}
    </div>
  );
}
