import { useMemo, useState } from "react";
import Icon, { ICONS } from "./Icon";
import ProjectRow from "./sidebar/ProjectRow";

export default function Sidebar({
  projects,
  view,
  selectedId,
  onNavigate,
  onSelect,
  onAdd,
  onOpenPrefs,
  onToggleFavorite,
}) {
  const [query, setQuery] = useState("");

  // Favorites float into their own group above the rest; both groups stay
  // alphabetical and share the same search filter.
  const { favorites, others } = useMemo(() => {
    const q = query.trim().toLowerCase();
    const matched = projects.filter(
      (p) => !q || (p.name || "").toLowerCase().includes(q)
    );
    const byName = (a, b) => (a.name || "").localeCompare(b.name || "");
    return {
      favorites: matched.filter((p) => p.favorite).sort(byName),
      others: matched.filter((p) => !p.favorite).sort(byName),
    };
  }, [projects, query]);

  const renderRow = (p) => (
    <ProjectRow
      key={p.id}
      project={p}
      active={p.id === selectedId}
      onSelect={onSelect}
      onToggleFavorite={onToggleFavorite}
    />
  );

  const empty = favorites.length === 0 && others.length === 0;

  return (
    <aside className="sidebar">
      <div className="titlebar-drag" />

      <nav className="side-cards">
        <button
          className={`side-card ${view === "dashboard" ? "active" : ""}`}
          onClick={() => onNavigate("dashboard")}
        >
          <Icon d={ICONS.dashboard} />
          <span>Dashboard</span>
        </button>
        <button
          className={`side-card ${view === "ports" ? "active" : ""}`}
          onClick={() => onNavigate("ports")}
        >
          <Icon d={ICONS.ports} />
          <span>Ports</span>
        </button>
      </nav>

      <div className="side-search">
        <input
          value={query}
          placeholder="Search projects…"
          onChange={(e) => setQuery(e.target.value)}
        />
      </div>

      <nav className="side-group side-projects">
        {favorites.length > 0 && (
          <>
            <div className="side-section with-icon">
              <Icon d={ICONS.star} filled />
              <span>Favorites</span>
            </div>
            {favorites.map(renderRow)}
          </>
        )}

        {others.length > 0 && (
          <>
            <div className="side-section">
              {favorites.length > 0 ? "All projects" : "Projects"}
            </div>
            {others.map(renderRow)}
          </>
        )}

        {empty && (
          <div className="side-empty">
            {projects.length === 0 ? "No projects yet" : "No matches"}
          </div>
        )}
      </nav>

      <div className="sidebar-foot">
        <button className="side-row" onClick={onAdd}>
          <Icon d={ICONS.plus} />
          <span>Add Project</span>
        </button>
        <button className="icon-btn" title="Preferences" onClick={onOpenPrefs}>
          <Icon d={ICONS.gear} />
        </button>
      </div>
    </aside>
  );
}
