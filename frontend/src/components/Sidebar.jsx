import { useEffect, useState } from "react";

const initial = (name) => (name || "?").trim().charAt(0).toUpperCase();

const Chevron = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <path d="M15 18l-6-6 6-6" />
  </svg>
);

export default function Sidebar({ projects, selectedId, onSelect, onAdd }) {
  const [collapsed, setCollapsed] = useState(
    () => localStorage.getItem("sidebarCollapsed") === "1"
  );

  useEffect(() => {
    localStorage.setItem("sidebarCollapsed", collapsed ? "1" : "0");
  }, [collapsed]);

  return (
    <aside className={`sidebar ${collapsed ? "collapsed" : ""}`}>
      <div className="titlebar-drag" />
      <div className="sidebar-head">
        {!collapsed && <h2>Projects</h2>}
        <button
          className="collapse-btn"
          title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          onClick={() => setCollapsed((c) => !c)}
        >
          <Chevron />
        </button>
      </div>
      {!collapsed && <div className="sub">Run and manage your local apps</div>}
      <nav>
        {projects.map((p) => (
          <button
            key={p.id}
            className={`side-item ${p.id === selectedId ? "active" : ""}`}
            title={collapsed ? p.name : undefined}
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
      <button className="add-btn" title="Add project" onClick={onAdd}>
        {collapsed ? "+" : "+ Add project"}
      </button>
    </aside>
  );
}
