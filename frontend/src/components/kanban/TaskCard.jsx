import { useState } from "react";

const TYPE_ICON = { story: "▣", task: "○", bug: "▲" };

// One draggable card on the board. Story cards can expand to show and
// tick off their child tasks.
export default function TaskCard({
  task,
  kids = [],
  onOpen,
  onToggleChild,
  onDragStart,
  dragging,
}) {
  const [open, setOpen] = useState(false);
  const subs = task.subtasks || [];
  const subsDone = subs.filter((s) => s.done).length;
  const kidsDone = kids.filter((k) => k.status === "done").length;
  const isStory = task.type === "story";

  return (
    <div
      className={`kb-card type-${task.type || "task"} ${dragging ? "dragging" : ""}`}
      draggable
      onDragStart={(e) => {
        e.dataTransfer.setData("text/plain", task.id);
        e.dataTransfer.effectAllowed = "move";
        onDragStart(task.id);
      }}
    >
      <div className="kb-card-main" onClick={() => onOpen(task)}>
        <div className="kb-card-title">
          <span className={`kb-type-badge t-${task.type || "task"}`}>
            {TYPE_ICON[task.type] || "○"}
          </span>
          {task.title}
        </div>
        {task.description && (
          <div className="kb-card-desc">{task.description}</div>
        )}
        <div className="kb-card-meta">
          {task.priority && (
            <span className={`kb-pill prio-${task.priority}`}>
              {task.priority}
            </span>
          )}
          {isStory && task.storyPoints > 0 && (
            <span className="kb-pill pts">{task.storyPoints} pts</span>
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

      {isStory && kids.length > 0 && (
        <div className="kb-kids">
          <button
            className="kb-kids-toggle"
            onClick={(e) => {
              e.stopPropagation();
              setOpen((o) => !o);
            }}
          >
            {open ? "▾" : "▸"} {kidsDone}/{kids.length} tasks
          </button>
          {open && (
            <div className="kb-kid-list">
              {kids.map((k) => (
                <div className="kb-kid" key={k.id}>
                  <button
                    className={`kb-kid-check ${k.status === "done" ? "done" : ""}`}
                    onClick={(e) => {
                      e.stopPropagation();
                      onToggleChild(k);
                    }}
                  >
                    {k.status === "done" ? "✓" : ""}
                  </button>
                  <span
                    className={`kb-kid-title ${k.status === "done" ? "done" : ""}`}
                    onClick={(e) => {
                      e.stopPropagation();
                      onOpen(k);
                    }}
                  >
                    {k.title}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
