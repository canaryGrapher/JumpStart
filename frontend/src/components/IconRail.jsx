const Icon = ({ d }) => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    {d.split("|").map((p, i) => (
      <path key={i} d={p} />
    ))}
  </svg>
);

const ICONS = {
  dashboard: "M3 3h8v8H3z|M13 3h8v5h-8z|M13 10h8v11h-8z|M3 13h8v8H3z",
  ports: "M4 17l6-6-6-6|M12 19h8",
};

export default function IconRail({ view, onNavigate }) {
  return (
    <aside className="rail">
      <div className="logo">🚀</div>
      <button
        className={`rail-btn ${view === "dashboard" ? "active" : ""}`}
        title="Dashboard"
        onClick={() => onNavigate("dashboard")}
      >
        <Icon d={ICONS.dashboard} />
      </button>
      <button
        className={`rail-btn ${view === "ports" ? "active" : ""}`}
        title="Ports"
        onClick={() => onNavigate("ports")}
      >
        <Icon d={ICONS.ports} />
      </button>
      <div className="spacer" />
    </aside>
  );
}
