import { useMemo, useState } from "react";
import { COLUMNS, TYPES, withStatus } from "./columns";
import TaskCard from "./TaskCard";

// Board with drag-and-drop between columns. Only top-level cards
// (stories and standalone tasks) are dragged; a story's children are
// shown nested inside its card.
export default function KanbanBoard({ tasks, onChange, onOpen, onAdd }) {
  const [dragId, setDragId] = useState(null);
  const [overCol, setOverCol] = useState(null);
  const [adding, setAdding] = useState(null); // column id with open input
  const [title, setTitle] = useState("");
  const [addType, setAddType] = useState("story");

  const childrenOf = useMemo(() => {
    const map = {};
    for (const t of tasks) {
      if (t.parentId) (map[t.parentId] ||= []).push(t);
    }
    return map;
  }, [tasks]);

  const topLevel = tasks.filter((t) => !t.parentId);

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
    if (t) onAdd(t, { type: addType, status: colId });
  };

  // Toggle a child task done/undone from inside its story card.
  const toggleChild = (child) =>
    onChange(
      tasks.map((t) =>
        t.id === child.id
          ? withStatus(t, t.status === "done" ? "todo" : "done")
          : t
      )
    );

  return (
    <div className="kb-board">
      {COLUMNS.map((col) => {
        const items = topLevel.filter((t) => t.status === col.id);
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
                  kids={childrenOf[t.id] || []}
                  dragging={dragId === t.id}
                  onOpen={onOpen}
                  onToggleChild={toggleChild}
                  onDragStart={setDragId}
                />
              ))}
              {items.length === 0 && (
                <div className="kb-empty">Drop items here</div>
              )}
            </div>

            {adding === col.id ? (
              <div className="kb-add">
                <div className="kb-type-toggle">
                  {TYPES.map((ty) => (
                    <button
                      key={ty.id}
                      className={addType === ty.id ? "active" : ""}
                      onClick={() => setAddType(ty.id)}
                    >
                      {ty.label}
                    </button>
                  ))}
                </div>
                <input
                  className="kb-add-input"
                  autoFocus
                  value={title}
                  placeholder={`${addType} title...`}
                  onChange={(e) => setTitle(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") submitAdd(col.id);
                    if (e.key === "Escape") setAdding(null);
                  }}
                  onBlur={() => submitAdd(col.id)}
                />
              </div>
            ) : (
              <button className="kb-add-btn" onClick={() => setAdding(col.id)}>
                + Add item
              </button>
            )}
          </div>
        );
      })}
    </div>
  );
}
