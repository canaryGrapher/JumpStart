import useBuildInfo, { LABELS, PRODUCTION } from "../buildInfo";

// Persistent build indicator shown in the topbar beside the title.
// Hidden entirely on production builds; dev/beta/pre-release show the label
// plus the running version so screenshots and bug reports are unambiguous.
export default function BuildBadge() {
  const { version, kind } = useBuildInfo();
  if (!kind || kind === PRODUCTION) return null;

  return (
    <span className="build-badge">
      {LABELS[kind]}
      <span className="build-badge-dot" aria-hidden="true">
        ·
      </span>
      <span className="build-badge-version">v{version}</span>
    </span>
  );
}
