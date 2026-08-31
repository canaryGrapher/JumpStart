// Small square/circular avatar for a project: the user's uploaded icon
// (project.icon, a data: URI) when present, otherwise the first letter of
// the project name. Shared by the sidebar row and the main header so both
// stay in sync automatically when an icon is added, changed, or removed.
export default function ProjectIcon({ project, className = "" }) {
  const initial = (project?.name || "?").trim().charAt(0).toUpperCase();
  const icon = project?.icon;

  return (
    <span className={`project-icon ${className}`}>
      {icon ? <img src={icon} alt="" /> : initial}
    </span>
  );
}
