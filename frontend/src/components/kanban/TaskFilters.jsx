import { useEffect, useMemo, useState } from "react";
import { ResolveDueRange } from "../../api";
import {
  DUE_PRESETS,
  EMPTY_FILTERS,
  activeFilterCount,
  formatDue,
} from "../../dueDates";

// Multi-select as toggle chips: no dependency, keyboard-friendly, and every
// option stays visible.
function ChipGroup({ label, options, selected, onChange }) {
  if (!options.length) return null;
  const toggle = (value) =>
    onChange(
      selected.includes(value) ? selected.filter((v) => v !== value) : [...selected, value]
    );
  return (
    <div className="kb-filter-group">
      <span className="kb-filter-label">{label}</span>
      <div className="kb-filter-chips">
        {options.map((o) => (
          <button
            key={o.value}
            type="button"
            className={`kb-chip ${selected.includes(o.value) ? "on" : ""}`}
            aria-pressed={selected.includes(o.value)}
            onClick={() => toggle(o.value)}
          >
            {o.label}
          </button>
        ))}
      </div>
    </div>
  );
}

const TRI = [
  { value: "", label: "Any" },
  { value: "has", label: "Has some" },
  { value: "none", label: "None" },
];

// Resolve the selected due-date preset to concrete dates using the backend,
// so quarter boundaries match what the AI and MCP use. Re-resolves when the
// window regains focus, since quarter dates can be edited in Settings.
export function useDueRange(filters, projectQuarters) {
  const [range, setRange] = useState(null);
  const [error, setError] = useState("");
  const [tick, setTick] = useState(0);
  const quartersKey = JSON.stringify(projectQuarters || []);

  useEffect(() => {
    const bump = () => setTick((t) => t + 1);
    window.addEventListener("focus", bump);
    return () => window.removeEventListener("focus", bump);
  }, []);

  useEffect(() => {
    let live = true;
    setError("");
    if (filters.duePreset) {
      ResolveDueRange(filters.duePreset, projectQuarters || [])
        .then((r) => live && setRange({ from: r.from, to: r.to }))
        .catch((e) => {
          if (!live) return;
          setRange(null);
          setError(String(e));
        });
    } else if (filters.dueFrom || filters.dueTo) {
      setRange({ from: filters.dueFrom, to: filters.dueTo });
    } else {
      setRange(null);
    }
    return () => {
      live = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters.duePreset, filters.dueFrom, filters.dueTo, quartersKey, tick]);

  return { range, error };
}

// Filter panel for the board. `filters` is controlled by the board.
export default function TaskFilters({
  tasks,
  columns,
  sprints,
  filters,
  onChange,
  range,
  rangeError,
}) {
  const [custom, setCustom] = useState(!!(filters.dueFrom || filters.dueTo));
  const set = (patch) => onChange({ ...filters, ...patch });

  const assigneeOptions = useMemo(() => {
    const seen = new Map();
    for (const t of tasks) {
      for (const a of String(t.assignee || "").split(",")) {
        const name = a.trim();
        if (name && !seen.has(name.toLowerCase())) seen.set(name.toLowerCase(), name);
      }
    }
    const list = [...seen.values()].sort((a, b) => a.localeCompare(b));
    return [
      { value: "__none__", label: "Unassigned" },
      ...list.map((n) => ({ value: n, label: `@${n}` })),
    ];
  }, [tasks]);

  const labelOptions = useMemo(() => {
    const seen = new Map();
    for (const t of tasks) {
      for (const l of t.labels || []) {
        if (l && !seen.has(l.toLowerCase())) seen.set(l.toLowerCase(), l);
      }
    }
    return [...seen.values()]
      .sort((a, b) => a.localeCompare(b))
      .map((l) => ({ value: l, label: l }));
  }, [tasks]);

  const presetValue = filters.duePreset || (custom ? "__custom__" : "");

  const onPreset = (value) => {
    if (value === "__custom__") {
      setCustom(true);
      set({ duePreset: "" });
    } else {
      setCustom(false);
      set({ duePreset: value, dueFrom: "", dueTo: "" });
    }
  };

  return (
    <div className="kb-filters">
      <div className="kb-filter-group">
        <span className="kb-filter-label">Due date</span>
        <div className="kb-filter-due">
          <select value={presetValue} onChange={(e) => onPreset(e.target.value)}>
            <option value="">Any time</option>
            {DUE_PRESETS.map((p) => (
              <option key={p.id} value={p.id}>
                {p.label}
              </option>
            ))}
            <option value="__custom__">Custom range…</option>
          </select>
          {custom && (
            <>
              <input
                type="date"
                aria-label="Due from"
                value={filters.dueFrom}
                onChange={(e) => set({ dueFrom: e.target.value, duePreset: "" })}
              />
              <span className="kb-filter-to">to</span>
              <input
                type="date"
                aria-label="Due to"
                value={filters.dueTo}
                onChange={(e) => set({ dueTo: e.target.value, duePreset: "" })}
              />
            </>
          )}
          {range && (range.from || range.to) && (
            <span className="kb-filter-range">
              {range.from ? formatDue(range.from) : "…"} – {range.to ? formatDue(range.to) : "…"}
            </span>
          )}
          {rangeError && <span className="kb-filter-error">{rangeError}</span>}
        </div>
        <label className="kb-filter-check">
          <input
            type="checkbox"
            checked={filters.overdue}
            onChange={(e) => set({ overdue: e.target.checked })}
          />
          Past due only
        </label>
        <label className="kb-filter-check">
          <input
            type="checkbox"
            checked={filters.noDueDate}
            onChange={(e) => set({ noDueDate: e.target.checked })}
          />
          No due date
        </label>
      </div>

      <ChipGroup
        label="Status"
        options={columns.map((c) => ({ value: c.id, label: c.label }))}
        selected={filters.statuses}
        onChange={(statuses) => set({ statuses })}
      />
      <ChipGroup
        label="Priority"
        options={[
          { value: "high", label: "High" },
          { value: "medium", label: "Medium" },
          { value: "low", label: "Low" },
          { value: "none", label: "None" },
        ]}
        selected={filters.priorities}
        onChange={(priorities) => set({ priorities })}
      />
      <ChipGroup
        label="Type"
        options={[
          { value: "story", label: "Story" },
          { value: "task", label: "Task" },
          { value: "bug", label: "Bug" },
        ]}
        selected={filters.types}
        onChange={(types) => set({ types })}
      />
      <ChipGroup
        label="Sprint"
        options={[
          { value: "", label: "Backlog" },
          ...sprints.map((s) => ({ value: s.id, label: s.name || "Untitled sprint" })),
        ]}
        selected={filters.sprints}
        onChange={(sprintsSel) => set({ sprints: sprintsSel })}
      />
      <ChipGroup
        label="Assignee"
        options={assigneeOptions}
        selected={filters.assignees}
        onChange={(assignees) => set({ assignees })}
      />
      <ChipGroup
        label="Labels"
        options={labelOptions}
        selected={filters.labels}
        onChange={(labels) => set({ labels })}
      />

      <div className="kb-filter-group">
        <span className="kb-filter-label">Acceptance criteria</span>
        <div className="kb-filter-chips">
          {TRI.map((o) => (
            <button
              key={o.value}
              type="button"
              className={`kb-chip ${filters.acceptance === o.value ? "on" : ""}`}
              onClick={() => set({ acceptance: o.value })}
            >
              {o.label}
            </button>
          ))}
        </div>
      </div>
      <div className="kb-filter-group">
        <span className="kb-filter-label">Subtasks</span>
        <div className="kb-filter-chips">
          {TRI.map((o) => (
            <button
              key={o.value}
              type="button"
              className={`kb-chip ${filters.subtasks === o.value ? "on" : ""}`}
              onClick={() => set({ subtasks: o.value })}
            >
              {o.label}
            </button>
          ))}
        </div>
      </div>

      {activeFilterCount(filters) > 0 && (
        <button
          type="button"
          className="link-btn kb-filter-clear"
          onClick={() => {
            setCustom(false);
            onChange({ ...EMPTY_FILTERS });
          }}
        >
          Clear all filters
        </button>
      )}
    </div>
  );
}
