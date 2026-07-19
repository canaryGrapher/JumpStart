import { useEffect, useState } from "react";
import { GitDiff } from "../../api";

const MODES = [
  { key: "working", label: "Working" },
  { key: "staging", label: "Staged" },
  { key: "worktree", label: "All changes" },
  { key: "remote", label: "vs Remote" },
  { key: "stash", label: "Stash" },
];

const lineClass = (line) => {
  if (line.startsWith("+++") || line.startsWith("---")) return "meta";
  if (line.startsWith("@@")) return "hunk";
  if (line.startsWith("diff ") || line.startsWith("index ")) return "file";
  if (line.startsWith("+")) return "add";
  if (line.startsWith("-")) return "del";
  return "";
};

export default function DiffModal({ projectRoot, onClose }) {
  const [mode, setMode] = useState("worktree");
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");

  useEffect(() => {
    let alive = true;
    setLoading(true);
    setErr("");
    GitDiff(projectRoot, mode)
      .then((r) => alive && setResult(r))
      .catch((e) => alive && setErr(String(e)))
      .finally(() => alive && setLoading(false));
    return () => {
      alive = false;
    };
  }, [projectRoot, mode]);

  const files = (result && result.files) || [];
  const lines = result && result.patch ? result.patch.split("\n") : [];

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal git-diff-modal" onClick={(e) => e.stopPropagation()}>
        <div className="git-diff-head">
          <h2>Changes</h2>
          <button className="btn small" onClick={onClose}>
            Close
          </button>
        </div>

        <div className="git-diff-tabs">
          {MODES.map((m) => (
            <button
              key={m.key}
              className={`git-diff-tab ${mode === m.key ? "active" : ""}`}
              onClick={() => setMode(m.key)}
            >
              {m.label}
            </button>
          ))}
        </div>

        <div className="git-diff-body">
          {loading ? (
            <div className="sub">Loading diff…</div>
          ) : err ? (
            <div className="error">{err}</div>
          ) : !result || result.empty ? (
            <div className="sub git-diff-empty">No changes for this comparison.</div>
          ) : (
            <>
              <div className="git-diff-summary">
                <span className="git-diff-label">{result.label}</span>
                {files.length > 0 && (
                  <ul className="git-diff-files">
                    {files.map((f) => (
                      <li key={f.path}>
                        <span className="git-diff-fname">{f.path}</span>
                        {(f.additions > 0 || f.deletions > 0) && (
                          <span className="git-diff-stat">
                            <span className="add">+{f.additions}</span>{" "}
                            <span className="del">-{f.deletions}</span>
                          </span>
                        )}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
              <pre className="git-diff-patch">
                {lines.map((l, i) => (
                  <div key={i} className={`git-diff-line ${lineClass(l)}`}>
                    {l || " "}
                  </div>
                ))}
              </pre>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
