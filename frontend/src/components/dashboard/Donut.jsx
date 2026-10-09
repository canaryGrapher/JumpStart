// Ring chart used by the dashboard panels and task widgets.
export default function Donut({ segments, size = 132, thickness = 18, center }) {
  const total = segments.reduce((s, x) => s + x.value, 0) || 1;
  const r = (size - thickness) / 2;
  const c = 2 * Math.PI * r;
  let offset = 0;
  return (
    <svg className="dash-donut" width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
      <circle
        cx={size / 2}
        cy={size / 2}
        r={r}
        fill="none"
        stroke="var(--card-soft)"
        strokeWidth={thickness}
      />
      {segments.map((seg) => {
        const len = (seg.value / total) * c;
        const el = (
          <circle
            key={seg.id}
            cx={size / 2}
            cy={size / 2}
            r={r}
            fill="none"
            stroke={seg.color}
            strokeWidth={thickness}
            strokeDasharray={`${len} ${c - len}`}
            strokeDashoffset={-offset}
            strokeLinecap="butt"
            transform={`rotate(-90 ${size / 2} ${size / 2})`}
          />
        );
        offset += len;
        return el;
      })}
      {center && (
        <foreignObject x={thickness} y={thickness} width={size - thickness * 2} height={size - thickness * 2}>
          <div className="dash-donut-center">{center}</div>
        </foreignObject>
      )}
    </svg>
  );
}
