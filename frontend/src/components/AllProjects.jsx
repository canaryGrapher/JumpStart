import { useMemo, useState } from "react";
import ProjectIcon from "./ProjectIcon";
import Icon, { ICONS } from "./Icon";

// Every project, as a searchable grid: title, description, task-completion
// progress, and a favorite toggle — the same star Sidebar uses, so
// favoriting here and from the sidebar always agree.
export default function AllProjects({ projects, onOpen, onToggleFavorite }) {
  const [query, setQuery] = useState("");

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    const matched = !q
      ? projects
      : projects.filter(
          (p) =>
            (p.name || "").toLowerCase().includes(q) ||
            (p.description || "").toLowerCase().includes(q)
        );
    return [...matched].sort((a, b) => (a.name || "").localeCompare(b.name || ""));
  }, [projects, query]);

  return (
    <>
      <div className="all-projects-search">
        <input
          className="all-projects-search-input"
          value={query}
          placeholder="Search projects…"
          onChange={(e) => setQuery(e.target.value)}
          autoFocus
        />
      </div>

      <div className="project-grid">
        {filtered.map((p) => {
          const tasks = p.tasks || [];
          const done = tasks.filter((t) => t.done).length;
          const pct = tasks.length ? Math.round((done / tasks.length) * 100) : null;
          const favorite = !!p.favorite;

          return (
            <div className="project-card" key={p.id} onClick={() => onOpen(p.id)}>
              <div className="project-card-top">
                <ProjectIcon project={p} className="main-header-icon" />
                <span className="project-card-name">{p.name}</span>
                <button
                  className={`project-card-fav ${favorite ? "on" : ""}`}
                  title={favorite ? "Remove from favorites" : "Add to favorites"}
                  aria-label={favorite ? "Remove from favorites" : "Add to favorites"}
                  aria-pressed={favorite}
                  onClick={(e) => {
                    e.stopPropagation();
                    onToggleFavorite(p);
                  }}
                >
                  <Icon d={ICONS.star} filled={favorite} />
                </button>
              </div>

              <p className={`project-card-desc ${p.description ? "" : "empty"}`}>
                {p.description || "No description yet."}
              </p>

              <div className="project-card-foot">
                {pct === null ? (
                  <span className="project-card-tasks-empty">No tasks tracked</span>
                ) : (
                  <>
                    <div className="meter">
                      <div style={{ width: `${pct}%` }} />
                    </div>
                    <span className="project-card-pct">
                      {done}/{tasks.length} tasks · {pct}%
                    </span>
                  </>
                )}
              </div>
            </div>
          );
        })}

        {filtered.length === 0 && (
          <div className="sub">{projects.length === 0 ? "No projects yet." : "No matches."}</div>
        )}
      </div>
    </>
  );
}
