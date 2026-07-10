// One draggable card on the kanban board.
export default function TaskCard({ task, onOpen, onDragStart, dragging }) {
  const subs = task.subtasks || [];
  const subsDone = subs.filter((s) => s.done).length;

  return (
    <div
      className={`kb-card ${dragging ? "dragging" : ""}`}
      draggable
      onDragStart={(e) => {
        e.dataTransfer.setData("text/plain", task.id);
        e.dataTransfer.effectAllowed = "move";
        onDragStart(task.id);
      }}
      onClick={() => onOpen(task)}
    >
      <div className="kb-card-title">{task.title}</div>
      {task.description && (
        <div className="kb-card-desc">{task.description}</div>
      )}
      <div className="kb-card-meta">
        {task.priority && (
          <span className={`kb-pill prio-${task.priority}`}>{task.priority}</span>
        )}
        {(task.labels || []).map((l) => (
          <span className="kb-pill label" key={l}>
            {l}
          </span>
        ))}
        {subs.length > 0 && (
          <span className="kb-pill subs">
            ☑ {subsDone}/{subs.length}
          </span>
        )}
      </div>
    </div>
  );
}
