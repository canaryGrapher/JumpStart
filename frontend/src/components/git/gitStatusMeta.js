// Maps a git status label (from FileChange.stagedLabel / unstagedLabel)
// to the short badge letter, full label, and CSS class the file rows and
// diff pane use. Centralised here so the icon story stays consistent
// wherever a file's status shows up.
const META = {
  modified: { icon: "M", label: "Modified", className: "mod" },
  added: { icon: "A", label: "Added", className: "add" },
  deleted: { icon: "D", label: "Deleted", className: "del" },
  renamed: { icon: "R", label: "Renamed", className: "ren" },
  copied: { icon: "C", label: "Copied", className: "ren" },
  "type-changed": { icon: "T", label: "Type changed", className: "mod" },
  untracked: { icon: "U", label: "Untracked", className: "unt" },
  conflicted: { icon: "!", label: "Conflicted", className: "conflict" },
};

export const statusMeta = (label) => META[label] || { icon: "?", label: label || "Changed", className: "mod" };
