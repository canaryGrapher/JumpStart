import { useState } from "react";
import { COLUMNS, withStatus } from "./columns";
import TaskCard from "./TaskCard";

// Board with drag-and-drop between columns.
export default function KanbanBoard({ tasks, onChange, onOpen, onAdd }) {
  const [dragId, setDragId] = useState(null);
  const [overCol, setOverCol] = useState(null);
  const [adding, setAdding] = useState(null); // column id with open input
  const [title, setTitle] = useState("");

  const drop = (colId) => {
    setOverCol(null);
    if (!dragId) return;
    const id = dragId;
    setDragId(null);
    const task = tasks.find((t) => t.id === id);
    if (!task || task.status === colId) return;
    onChange(tasks.map((t) => (t.id === id ? withStatus(t, colId) : t)));
  };

  const submitAdd = (colId) => {
    const t = title.trim();
    setTitle("");
    setAdding(null);
    if (t) onAdd(t, colId);
  };

  return (
    <div className="kb-board">
      {COLUMNS.map((col) => {
        const items = tasks.filter((t) => t.status === col.id);
        return (
          <div
            key={col.id}
            className={`kb-col ${overCol === col.id ? "drag-over" : ""}`}
            onDragOver={(e) => {
              e.preventDefault();
              e.dataTransfer.dropEffect = "move";
              setOverCol(col.id);
            }}
            onDragLeave={(e) => {
              if (!e.currentTarget.contains(e.relatedTarget)) setOverCol(null);
            }}
            onDrop={(e) => {
              e.preventDefault();
              drop(col.id);
            }}
          >
            <div className="kb-col-head">
              <span className="kb-col-title">{col.label}</span>
              <span className="kb-count">{items.length}</span>
            </div>

            <div className="kb-col-body">
              {items.map((t) => (
                <TaskCard
                  key={t.id}
                  task={t}
                  dragging={dragId === t.id}
                  onOpen={onOpen}
                  onDragStart={setDragId}
                />
              ))}
              {items.length === 0 && (
                <div className="kb-empty">Drop tasks here</div>
              )}
            </div>

            {adding === col.id ? (
              <input
                className="kb-add-input"
                autoFocus
                value={title}
                placeholder="Task title..."
                onChange={(e) => setTitle(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") submitAdd(col.id);
                  if (e.key === "Escape") setAdding(null);
                }}
                onBlur={() => submitAdd(col.id)}
              />
            ) : (
              <button className="kb-add-btn" onClick={() => setAdding(col.id)}>
                + Add task
              </button>
            )}
          </div>
        );
      })}
    </div>
  );
}
