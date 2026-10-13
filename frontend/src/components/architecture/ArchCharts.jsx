function BarChart({ title, points = [] }) {
  const max = Math.max(1, ...points.map((p) => p.value || 0));
  return (
    <div className="arch-chart">
      {title ? <h4>{title}</h4> : null}
      <div className="arch-bars">
        {points.map((p) => {
          const pct = Math.max(2, Math.round(((p.value || 0) / max) * 100));
          return (
            <div key={p.label} className="arch-bar" title={`${p.label}: ${p.value}`}>
              <span className="arch-bar-label">{p.label}</span>
              <div className="arch-bar-track">
                <div className="arch-bar-fill" style={{ width: `${pct}%` }} />
              </div>
              <span className="arch-bar-val">{p.value}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function PieChart({ title, points = [] }) {
  const total = points.reduce((s, p) => s + (p.value || 0), 0) || 1;
  const colors = ["#3b82f6", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6", "#14b8a6", "#ec4899"];
  let acc = 0;
  const slices = points.map((p, i) => {
    const start = (acc / total) * 360;
    acc += p.value || 0;
    const end = (acc / total) * 360;
    return { ...p, start, end, color: colors[i % colors.length] };
  });

  // Conic gradient pie
  const gradient = slices
    .map((s) => `${s.color} ${s.start}deg ${s.end}deg`)
    .join(", ");

  return (
    <div className="arch-chart arch-pie-wrap">
      {title ? <h4>{title}</h4> : null}
      <div className="arch-pie-row">
        <div className="arch-pie" style={{ background: `conic-gradient(${gradient})` }} title={title} />
        <ul className="arch-pie-legend">
          {slices.map((s) => (
            <li key={s.label} title={`${s.label}: ${s.value}`}>
              <span className="swatch" style={{ background: s.color }} />
              <span>{s.label}</span>
              <span className="dim">{s.value}</span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}

export function ArchCharts({ charts = [] }) {
  if (!charts.length) return null;
  return (
    <div className="arch-charts">
      {charts.map((c, i) =>
        c.type === "pie" ? (
          <PieChart key={i} title={c.title} points={c.points} />
        ) : (
          <BarChart key={i} title={c.title} points={c.points} />
        )
      )}
    </div>
  );
}

export function ArchTables({ tables = [] }) {
  if (!tables.length) return null;
  return (
    <div className="arch-tables">
      {tables.map((t, i) => (
        <div key={i} className="arch-table-block">
          {t.title ? <h4>{t.title}</h4> : null}
          <div className="arch-table-wrap">
            <table className="arch-table">
              {t.headers?.length ? (
                <thead>
                  <tr>
                    {t.headers.map((h, hi) => (
                      <th key={hi}>{h}</th>
                    ))}
                  </tr>
                </thead>
              ) : null}
              <tbody>
                {(t.rows || []).map((row, ri) => (
                  <tr key={ri}>
                    {row.map((cell, ci) => (
                      <td key={ci} title={cell}>
                        {cell}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ))}
    </div>
  );
}
