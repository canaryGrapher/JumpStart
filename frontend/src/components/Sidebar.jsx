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
  return (
    <aside className="sidebar">
      <div className="titlebar-drag" />

      <nav className="side-group">
        <button
          className={`side-row ${view === "dashboard" ? "active" : ""}`}
          onClick={() => onNavigate("dashboard")}
        >
          <Icon d={ICONS.dashboard} />
          <span>Dashboard</span>
        </button>
        <button
          className={`side-row ${view === "ports" ? "active" : ""}`}
          onClick={() => onNavigate("ports")}
        >
          <Icon d={ICONS.ports} />
          <span>Ports</span>
        </button>
      </nav>

      <div className="side-section">Projects</div>

      <nav className="side-group side-projects">
        {projects.map((p) => (
          <button
            key={p.id}
            className={`side-row project ${p.id === selectedId ? "active" : ""}`}
            onClick={() => onSelect(p.id)}
          >
            <span className="avatar">{initial(p.name)}</span>
            <span className="side-text">
              <span className="side-name">{p.name}</span>
              <span className="side-sub">{plural((p.processes || []).length)}</span>
            </span>
          </button>
        ))}
        {projects.length === 0 && <div className="side-empty">No projects yet</div>}
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
