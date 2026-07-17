import { useMemo, useState } from "react";
import Icon, { ICONS } from "./Icon";

const initial = (name) => (name || "?").trim().charAt(0).toUpperCase();

const plural = (n) => `${n} subprocess${n === 1 ? "" : "es"}`;

export default function Sidebar({
  projects,
  view,
  selectedId,
  onNavigate,
  onSelect,
  onAdd,
  onOpenPrefs,
}) {
  const [query, setQuery] = useState("");

  const visible = useMemo(() => {
    const sorted = [...projects].sort((a, b) =>
      (a.name || "").localeCompare(b.name || "")
    );
    const q = query.trim().toLowerCase();
    if (!q) return sorted;
    return sorted.filter((p) => (p.name || "").toLowerCase().includes(q));
  }, [projects, query]);

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

      <div className="side-section">Projects</div>

      <div className="side-search">
        <input
          value={query}
          placeholder="Search projects…"
          onChange={(e) => setQuery(e.target.value)}
        />
      </div>

      <nav className="side-group side-projects">
        {visible.map((p) => (
          <button
            key={p.id}
            className={`side-row project ${p.id === selectedId ? "active" : ""}`}
            onClick={() => onSelect(p.id)}
          >
            <span className="avatar">{initial(p.name)}</span>
            <span className="side-text">
              <span className="side-name">{p.name}</span>
              <span className="side-sub">{plural((p.processes || []).length)}</span>
              {p.description && <span className="side-desc">{p.description}</span>}
            </span>
          </button>
        ))}
        {visible.length === 0 && (
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
