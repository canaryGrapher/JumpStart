import { useMemo, useState } from "react";

// Simple layered / circular layout for architecture nodes + edges.
// No external graph library — SVG only, matching the git graph approach.

function layoutNodes(nodes) {
  const n = nodes.length;
  if (n === 0) return new Map();
  const cx = 280;
  const cy = 180;
  const rx = Math.min(220, 40 + n * 14);
  const ry = Math.min(140, 30 + n * 10);
  const map = new Map();
  nodes.forEach((node, i) => {
    const a = (Math.PI * 2 * i) / n - Math.PI / 2;
    map.set(node.id, {
      x: cx + rx * Math.cos(a),
      y: cy + ry * Math.sin(a),
      ...node,
    });
  });
  return map;
}

export default function ArchGraph({ nodes = [], edges = [] }) {
  const [hover, setHover] = useState(null);
  const positioned = useMemo(() => layoutNodes(nodes), [nodes]);
  const list = [...positioned.values()];

  if (!nodes.length) {
    return <p className="arch-empty">No graph nodes in this section.</p>;
  }

  const tip = hover
    ? hover.kind === "edge"
      ? `${hover.label || "link"}: ${hover.from} → ${hover.to}${hover.description ? ` — ${hover.description}` : ""}`
      : `${hover.label}${hover.kind ? ` (${hover.kind})` : ""}${hover.description ? ` — ${hover.description}` : ""}`
    : null;

  return (
    <div className="arch-graph">
      <svg viewBox="0 0 560 360" className="arch-graph-svg" role="img" aria-label="Architecture graph">
        {edges.map((e, i) => {
          const a = positioned.get(e.from);
          const b = positioned.get(e.to);
          if (!a || !b) return null;
          const active = hover?.kind === "edge" && hover.from === e.from && hover.to === e.to;
          return (
            <g key={`${e.from}-${e.to}-${i}`}>
              <line
                x1={a.x}
                y1={a.y}
                x2={b.x}
                y2={b.y}
                className={`arch-edge ${active ? "on" : ""}`}
                onMouseEnter={() => setHover({ kind: "edge", ...e })}
                onMouseLeave={() => setHover(null)}
              />
            </g>
          );
        })}
        {list.map((n) => {
          const active = hover?.kind === "node" && hover.id === n.id;
          return (
            <g
              key={n.id}
              transform={`translate(${n.x},${n.y})`}
              onMouseEnter={() => setHover({ kind: "node", ...n })}
              onMouseLeave={() => setHover(null)}
              className={`arch-node ${active ? "on" : ""}`}
            >
              <circle r="22" />
              <text textAnchor="middle" dy="4" className="arch-node-label">
                {(n.label || n.id).slice(0, 10)}
              </text>
            </g>
          );
        })}
      </svg>
      {tip ? <div className="arch-tooltip">{tip}</div> : <div className="arch-tooltip dim">Hover a node or edge for details</div>}
    </div>
  );
}
