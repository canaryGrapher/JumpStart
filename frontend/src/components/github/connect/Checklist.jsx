// A step in the detection checklist. `state` drives both the icon and
// the stagger animation: "pending" items are still queued, "done"/"warn"
// items have resolved and settle into place a little after the one
// before them, which is what sells the "figuring it out" feel.
const ICONS = {
  done: "M20 6L9 17l-5-5",
  warn: "M12 9v4|M12 17h.01",
  pending: "",
};

export function ChecklistItem({ state, label, index }) {
  return (
    <li
      className={`gh-check-item ${state}`}
      style={{ animationDelay: `${index * 90}ms` }}
    >
      <span className="gh-check-icon">
        {state === "pending" ? (
          <span className="gh-check-spinner" />
        ) : (
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round">
            {ICONS[state].split("|").map((d, i) => (
              <path key={i} d={d} />
            ))}
          </svg>
        )}
      </span>
      <span className="gh-check-label">{label}</span>
    </li>
  );
}

export default function Checklist({ items }) {
  return (
    <ul className="gh-checklist">
      {items.map((item, i) => (
        <ChecklistItem key={item.key} index={i} state={item.state} label={item.label} />
      ))}
    </ul>
  );
}
