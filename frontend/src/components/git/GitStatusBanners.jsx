// Banners for the git states that need an explanation rather than just a
// disabled button: detached HEAD, unresolved merge conflicts, and the
// last push/publish failure (auth, rejection, or anything else) with the
// raw error message underneath the friendly gloss.
export default function GitStatusBanners({ status, hasConflicts, pushError }) {
  const banners = [];

  if (status.detachedHead) {
    banners.push({
      kind: "warn",
      text: "You're in a detached HEAD state — create a branch to save this work before switching away.",
    });
  }

  if (hasConflicts || status.conflicted) {
    banners.push({
      kind: "err",
      text: "Merge conflicts need to be resolved before you can commit.",
    });
  }

  if (pushError) {
    const lower = pushError.toLowerCase();
    if (lower.includes("authentication") || lower.includes("401") || lower.includes("403")) {
      banners.push({ kind: "err", text: "Authentication failed — check your Git token in Preferences." });
    } else if (lower.includes("rejected") || lower.includes("non-fast-forward")) {
      banners.push({ kind: "err", text: "Push was rejected — pull the latest changes first, then try again." });
    } else {
      banners.push({ kind: "err", text: pushError });
    }
  }

  if (banners.length === 0) return null;

  return (
    <div className="git-banners">
      {banners.map((b, i) => (
        <div key={i} className={`git-banner ${b.kind}`}>
          {b.text}
        </div>
      ))}
    </div>
  );
}
