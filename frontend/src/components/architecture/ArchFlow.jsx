import { useState } from "react";

export default function ArchFlow({ steps = [] }) {
  const [hover, setHover] = useState(null);
  if (!steps.length) return null;

  return (
    <div className="arch-flow">
      <ol className="arch-flow-list">
        {steps.map((s, i) => (
          <li key={s.id || i}>
            <button
              type="button"
              className={`arch-flow-step ${hover === s.id ? "on" : ""}`}
              onMouseEnter={() => setHover(s.id)}
              onMouseLeave={() => setHover(null)}
              title={s.description || s.label}
            >
              <span className="arch-flow-num">{i + 1}</span>
              <span className="arch-flow-body">
                <strong>{s.label}</strong>
                {s.description ? <span className="dim">{s.description}</span> : null}
              </span>
            </button>
            {i < steps.length - 1 ? <div className="arch-flow-arrow" aria-hidden>↓</div> : null}
          </li>
        ))}
      </ol>
    </div>
  );
}
