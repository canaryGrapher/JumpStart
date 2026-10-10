import { useEffect, useMemo, useRef, useState } from "react";
import Icon, { ICONS } from "./Icon";
import ProjectRow from "./sidebar/ProjectRow";

export default function Sidebar({
  projects,
  view,
  selectedId,
  collapsed = false,
  onExpand,
  onNavigate,
  onSelect,
  onAdd,
  onOpenPrefs,
  onToggleFavorite,
}) {
  const [query, setQuery] = useState("");
  const searchRef = useRef(null);
  const focusSearchAfterExpand = useRef(false);

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

  useEffect(() => {
    if (collapsed) {
      setQuery("");
      return;
    }
    if (focusSearchAfterExpand.current) {
      focusSearchAfterExpand.current = false;
      searchRef.current?.focus();
    }
  }, [collapsed]);

  const expandAndFocusSearch = () => {
    focusSearchAfterExpand.current = true;
    onExpand?.();
  };

  const renderRow = (p) => (
    <ProjectRow
      key={p.id}
      project={p}
      active={p.id === selectedId}
      collapsed={collapsed}
      onSelect={onSelect}
      onToggleFavorite={onToggleFavorite}
    />
  );

  const empty = favorites.length === 0 && others.length === 0;

  return (
    <aside className={`sidebar ${collapsed ? "is-collapsed" : ""}`} aria-label="Sidebar">
      <div className="titlebar-drag" />

      {collapsed ? (
        <button
          className="side-search-btn"
          type="button"
          title="Search"
          aria-label="Search projects"
          onClick={expandAndFocusSearch}
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <circle cx="11" cy="11" r="7" />
            <path d="M20 20l-3.5-3.5" />
          </svg>
        </button>
      ) : (
        <div className="side-search">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <circle cx="11" cy="11" r="7" />
            <path d="M20 20l-3.5-3.5" />
          </svg>
          <input
            ref={searchRef}
            value={query}
            placeholder="Search"
            aria-label="Search projects"
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => e.key === "Escape" && setQuery("")}
          />
        </div>
      )}

      <nav className="side-group side-nav">
        <button
          className={`side-row ${view === "dashboard" ? "active" : ""}`}
          title="Dashboard"
          aria-label="Dashboard"
          aria-current={view === "dashboard" ? "page" : undefined}
          onClick={() => onNavigate("dashboard")}
        >
          <Icon d={ICONS.dashboard} />
          {!collapsed && <span>Dashboard</span>}
        </button>
        <button
          className={`side-row ${view === "ports" ? "active" : ""}`}
          title="Ports"
          aria-label="Ports"
          aria-current={view === "ports" ? "page" : undefined}
          onClick={() => onNavigate("ports")}
        >
          <Icon d={ICONS.ports} />
          {!collapsed && <span>Ports</span>}
        </button>
      </nav>

      <nav className="side-group side-projects">
        {favorites.length > 0 && (
          <>
            {!collapsed && (
              <div className="side-section with-icon">
                <Icon d={ICONS.star} filled />
                <span>Favorites</span>
              </div>
            )}
            {favorites.map(renderRow)}
            {collapsed && others.length > 0 && <div className="side-rail-sep" aria-hidden="true" />}
          </>
        )}

        {others.length > 0 && (
          <>
            {!collapsed && (
              <div className="side-section">
                {favorites.length > 0 ? "All projects" : "Projects"}
              </div>
            )}
            {others.map(renderRow)}
          </>
        )}

        {!collapsed && empty && (
          <div className="side-empty">
            {projects.length === 0 ? "No projects yet" : "No matches"}
          </div>
        )}
      </nav>

      <div className="sidebar-foot">
        <button className="side-row" title="Add Project" aria-label="Add Project" onClick={onAdd}>
          <Icon d={ICONS.plus} />
          {!collapsed && <span>Add Project</span>}
        </button>
        <button className="icon-btn" title="Settings" aria-label="Settings" onClick={onOpenPrefs}>
          <Icon d={ICONS.gear} />
        </button>
      </div>
    </aside>
  );
}
