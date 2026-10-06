// Summarize GitHub conflict field reasons for tooltips and list rows.
export function conflictReasonSummary(github, max = 2) {
  const fields = github?.conflictFields || [];
  if (!fields.length) {
    return github?.conflict
      ? "Local edits differ from GitHub"
      : "";
  }
  const parts = fields.slice(0, max).map((f) => {
    const label = f.label || f.field || "Field";
    return `${label} differs`;
  });
  const more = fields.length > max ? ` (+${fields.length - max} more)` : "";
  return parts.join("; ") + more;
}

export function formatConflictLine(f) {
  const label = f.label || f.field || "Field";
  const local = f.local || "(empty)";
  const remote = f.remote || "(empty)";
  return `${label}: yours “${local}” vs GitHub “${remote}”`;
}
