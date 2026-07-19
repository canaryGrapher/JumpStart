import { BACKLOG_ID, sprintStats } from "./sprints";

// Row of sprint pills above the board. Picking one filters the board;
// dropping a dragged card on a pill reassigns it to that sprint.
export default function SprintBar({
  sprints,
  tasks,
  selected,
  onSelect,
  onDropTask,
  onOpenRoadmap,
  onQuickAdd,
}) {
  const pills = [
    { id: BACKLOG_ID, name: "Backlog", status: "" },
    ...sprints,
    { id: "__all__", name: "All tasks", status: "" },
  ];

  return (
    <div className="sprint-bar">
      <div className="sprint-pills">
        {pills.map((s) => {
          const isAll = s.id === "__all__";
          const stats = isAll
            ? { done: tasks.filter((t) => t.status === "done").length, total: tasks.length }
            : sprintStats(tasks, s.id);
          return (
            <button
              key={s.id || "backlog"}
              className={`sprint-pill ${selected === s.id ? "active" : ""} ${
                s.status ? `state-${s.status}` : ""
              }`}
              onClick={() => onSelect(s.id)}
              onDragOver={(e) => {
                if (isAll) return;
                e.preventDefault();
                e.dataTransfer.dropEffect = "move";
              }}
              onDrop={(e) => {
                if (isAll) return;
                e.preventDefault();
                onDropTask && onDropTask(s.id);
              }}
            >
              <span className="sprint-pill-name">{s.name}</span>
              <span className="sprint-pill-count">
                {stats.done}/{stats.total}
              </span>
            </button>
          );
        })}
      </div>

      <div className="sprint-bar-actions">
        <button className="btn small" onClick={onQuickAdd}>
          + Sprint
        </button>
        <button className="btn small" onClick={onOpenRoadmap}>
          Roadmap
        </button>
      </div>
    </div>
  );
}
