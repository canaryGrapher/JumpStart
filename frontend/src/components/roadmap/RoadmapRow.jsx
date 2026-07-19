import { SPRINT_STATES, formatRange } from "../kanban/sprints";

// One sprint in the roadmap sequence: draggable handle, inline fields,
// and a progress meter for the tasks assigned to it.
export default function RoadmapRow({
  sprint,
  index,
  stats,
  editing,
  dragging,
  dropBefore,
  onEdit,
  onPatch,
  onDelete,
  onDragStart,
  onDragOver,
  onDrop,
}) {
  return (
    <div
      className={`rm-row ${dragging ? "dragging" : ""} ${
        dropBefore ? "drop-before" : ""
      } state-${sprint.status || "planned"}`}
      draggable
      onDragStart={() => onDragStart(index)}
      onDragEnd={() => onDragStart(null)}
      onDragOver={(e) => {
        e.preventDefault();
        onDragOver(index);
      }}
      onDrop={(e) => {
        e.preventDefault();
        onDrop(index);
      }}
    >
      <div className="rm-seq">
        <span className="rm-handle">⋮⋮</span>
        <span className="rm-num">{index + 1}</span>
      </div>

      <div className="rm-body">
        {editing ? (
          <div className="rm-edit">
            <input
              className="rm-name-input"
              autoFocus
              value={sprint.name}
              placeholder="Sprint name"
              onChange={(e) => onPatch({ name: e.target.value })}
            />
            <input
              value={sprint.goal || ""}
              placeholder="Sprint goal..."
              onChange={(e) => onPatch({ goal: e.target.value })}
            />
            <div className="rm-edit-row">
              <input
                type="date"
                value={sprint.startDate || ""}
                onChange={(e) => onPatch({ startDate: e.target.value })}
              />
              <input
                type="date"
                value={sprint.endDate || ""}
                onChange={(e) => onPatch({ endDate: e.target.value })}
              />
              <select
                value={sprint.status || "planned"}
                onChange={(e) => onPatch({ status: e.target.value })}
              >
                {SPRINT_STATES.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.label}
                  </option>
                ))}
              </select>
            </div>
            <div className="rm-edit-actions">
              <button className="btn danger small" onClick={onDelete}>
                Delete
              </button>
              <button className="btn small" onClick={() => onEdit(null)}>
                Done
              </button>
            </div>
          </div>
        ) : (
          <div className="rm-view" onClick={() => onEdit(sprint.id)}>
            <div className="rm-title-line">
              <strong>{sprint.name}</strong>
              <span className={`kb-pill state-${sprint.status || "planned"}`}>
                {sprint.status || "planned"}
              </span>
            </div>
            {sprint.goal && <div className="rm-goal">{sprint.goal}</div>}
            <div className="rm-meta">
              <span>{formatRange(sprint)}</span>
              <span>
                {stats.done}/{stats.total} tasks
                {stats.points ? ` · ${stats.points} pts` : ""}
              </span>
            </div>
            <div className="meter">
              <div style={{ width: `${stats.pct}%` }} />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
