import { statusMeta } from "./gitStatusMeta";

// One row in the Unstaged/Staged file lists: a status badge, the path,
// and the single action available for that side (Stage or Unstage).
// Clicking the row (rather than the action button) selects the file for
// the diff pane, so review and staging never fight over the same click.
export default function FileRow({ file, staged, selected, busy, onSelect, onToggle }) {
  const label = staged ? file.stagedLabel : file.unstagedLabel;
  const meta = statusMeta(label);

  return (
    <div className={`git-file-row ${selected ? "selected" : ""} ${file.conflicted ? "conflict" : ""}`}>
      <button
        type="button"
        className="git-file-main"
        onClick={() => onSelect(file.path, staged)}
        title={file.oldPath ? `${file.oldPath} → ${file.path}` : file.path}
      >
        <span className={`git-file-badge ${meta.className}`} title={meta.label}>
          {meta.icon}
        </span>
        <span className="git-file-path">{file.path}</span>
        {file.conflicted && <span className="git-file-conflict-tag">Conflict</span>}
      </button>
      <button
        type="button"
        className={`btn tiny ${staged ? "" : "primary"}`}
        disabled={busy}
        onClick={(e) => {
          e.stopPropagation();
          onToggle(file.path);
        }}
      >
        {staged ? "Unstage" : "Stage"}
      </button>
    </div>
  );
}
