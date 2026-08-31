import FileRow from "./FileRow";

// A titled section (Unstaged Changes / Staged Changes) with a count, a
// bulk action (Stage All / Unstage All), and its file rows. Staged and
// unstaged sections render this same component with different props so
// the two stay visually consistent while remaining unambiguous about
// which side a file is on.
export default function FileList({
  title,
  files,
  staged,
  busy,
  selectedPath,
  selectedStaged,
  onSelect,
  onToggle,
  onBulkAction,
  bulkLabel,
  emptyLabel,
}) {
  return (
    <div className={`git-file-section ${staged ? "staged" : "unstaged"}`}>
      <div className="git-file-section-head">
        <span className="git-file-section-title">
          {title}
          <span className="git-file-count">{files.length}</span>
        </span>
        {files.length > 0 && (
          <button type="button" className="btn tiny" disabled={busy} onClick={onBulkAction}>
            {bulkLabel}
          </button>
        )}
      </div>
      {files.length === 0 ? (
        <div className="git-file-empty">{emptyLabel}</div>
      ) : (
        <div className="git-file-list">
          {files.map((f) => (
            <FileRow
              key={`${staged ? "s" : "u"}:${f.path}`}
              file={f}
              staged={staged}
              busy={busy}
              selected={selectedPath === f.path && selectedStaged === staged}
              onSelect={onSelect}
              onToggle={onToggle}
            />
          ))}
        </div>
      )}
    </div>
  );
}
