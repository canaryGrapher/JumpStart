const initial = (name) => (name || "?").trim().charAt(0).toUpperCase();

export default function Sidebar({ projects, selectedId, onSelect, onAdd }) {
  return (
    <aside className="sidebar">
      <h2>Projects</h2>
      <div className="sub">Run and manage your local apps</div>
      <nav>
        {projects.map((p) => (
          <button
            key={p.id}
            className={`side-item ${p.id === selectedId ? "active" : ""}`}
            onClick={() => onSelect(p.id)}
          >
            <span className="avatar">{initial(p.name)}</span>
            <span className="side-text">
              <span>{p.name}</span>
              <span className="side-sub">
                {(p.processes || []).length} subprocess{(p.processes || []).length === 1 ? "" : "es"}
              </span>
            </span>
          </button>
        ))}
      </nav>
      <button className="add-btn" onClick={onAdd}>
        + Add project
      </button>
    </aside>
  );
}
