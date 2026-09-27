import Icon, { ICONS } from "../Icon";
import ProjectIcon from "../ProjectIcon";

const plural = (n) => `${n} subprocess${n === 1 ? "" : "es"}`;

// One project entry in the sidebar. The star is a separate button layered on
// the row rather than a nested <button>, which is invalid HTML and would make
// the whole row unclickable in some browsers.
export default function ProjectRow({ project, active, onSelect, onToggleFavorite }) {
  const favorite = !!project.favorite;

  return (
    <div className={`side-row-wrap ${favorite ? "is-favorite" : ""}`}>
      <button
        className={`side-row project ${active ? "active" : ""}`}
        title={project.description || project.name}
        aria-current={active ? "page" : undefined}
        onClick={() => onSelect(project.id)}
      >
        <ProjectIcon project={project} className="avatar" />
        <span className="side-text">
          <span className="side-name">{project.name}</span>
          <span className="side-sub">{plural((project.processes || []).length)}</span>
        </span>
      </button>

      <button
        className={`side-star ${favorite ? "on" : ""}`}
        title={favorite ? "Remove from favorites" : "Add to favorites"}
        aria-label={favorite ? "Remove from favorites" : "Add to favorites"}
        aria-pressed={favorite}
        onClick={() => onToggleFavorite(project)}
      >
        <Icon d={ICONS.star} filled={favorite} />
      </button>
    </div>
  );
}
