import {
  DEFAULT_QUARTERS,
  MONTHS,
  daysInMonth,
  formatMD,
  parseMD,
} from "../dueDates";

// Month + day pickers for one year-agnostic MM-DD date.
function MonthDay({ value, onChange, label }) {
  const { month, day } = parseMD(value);
  const clampDay = (m, d) => Math.min(d, daysInMonth(m));
  return (
    <span className="quarter-md" role="group" aria-label={label}>
      <select
        aria-label={`${label} month`}
        value={month}
        onChange={(e) => {
          const m = Number(e.target.value);
          onChange(formatMD({ month: m, day: clampDay(m, day) }));
        }}
      >
        {MONTHS.map((name, i) => (
          <option key={name} value={i + 1}>
            {name}
          </option>
        ))}
      </select>
      <select
        aria-label={`${label} day`}
        value={day}
        onChange={(e) => onChange(formatMD({ month, day: Number(e.target.value) }))}
      >
        {Array.from({ length: daysInMonth(month) }, (_, i) => i + 1).map((d) => (
          <option key={d} value={d}>
            {d}
          </option>
        ))}
      </select>
    </span>
  );
}

// Edits the four quarters as start/end month-day pairs. The end may be
// earlier in the year than the start (a quarter that wraps New Year).
export default function QuarterEditor({ value, onChange }) {
  const quarters =
    Array.isArray(value) && value.length === 4 ? value : DEFAULT_QUARTERS;
  const update = (i, patch) =>
    onChange(
      quarters.map((q, idx) =>
        idx === i ? { ...q, name: q.name || `Q${i + 1}`, ...patch } : q
      )
    );
  return (
    <div className="quarter-editor">
      {quarters.map((q, i) => (
        <div className="quarter-row" key={i}>
          <span className="quarter-name">{q.name || `Q${i + 1}`}</span>
          <MonthDay
            label={`Q${i + 1} start`}
            value={q.start}
            onChange={(start) => update(i, { start })}
          />
          <span className="quarter-to">to</span>
          <MonthDay
            label={`Q${i + 1} end`}
            value={q.end}
            onChange={(end) => update(i, { end })}
          />
        </div>
      ))}
    </div>
  );
}
