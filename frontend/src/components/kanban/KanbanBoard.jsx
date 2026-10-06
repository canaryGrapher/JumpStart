import { useCallback, useLayoutEffect, useMemo, useRef, useState } from "react";
import ConfirmDialog from "../ConfirmDialog";
import { DEFAULT_COLUMNS, TYPES, withStatus } from "./columns";
import TaskCard from "./TaskCard";
import TaskContextMenu from "./TaskContextMenu";
import SprintBar from "./SprintBar";

// Board with drag-and-drop between columns. Only top-level cards
// (stories and standalone tasks) are dragged; a story's children are
// shown nested inside its card. The board shows one sprint at a time,
// or every task when the sprint filter is "__all__".
export default function KanbanBoard({
  tasks,
  columns = DEFAULT_COLUMNS,
  sprints = [],
  sprintFilter = "",
  onSprintFilter,
  onOpenRoadmap,
  onQuickAddSprint,
  onOpenAI,
  onChange,
  onOpen,
  onDelete,
  onAdd,
  onAddColumn,
}) {
  const [dragId, setDragId] = useState(null);
  const [overCol, setOverCol] = useState(null);
  const [adding, setAdding] = useState(null); // column id with open input
  const [title, setTitle] = useState("");
  const [addType, setAddType] = useState("story");
  const [query, setQuery] = useState("");
  const [menu, setMenu] = useState(null); // { x, y, task }
  const [pendingDelete, setPendingDelete] = useState(null);
  const headRef = useRef(null);

  // .kb-board (below) needs to know how tall this sticky group — the
  // sprint bar plus the search input — actually is, so it can size
  // itself to exactly the viewport space left under the tabs. See
  // .kb-head / .kb-board in _kanban.scss.
  useLayoutEffect(() => {
    const el = headRef.current;
    if (!el) return;
    const publish = () => {
      document.documentElement.style.setProperty("--kb-head-h", `${el.offsetHeight}px`);
    };
    publish();
    const ro = new ResizeObserver(publish);
    ro.observe(el);
    return () => {
      ro.disconnect();
      document.documentElement.style.removeProperty("--kb-head-h");
    };
  }, []);

  const childrenOf = useMemo(() => {
    const map = {};
    for (const t of tasks) {
      if (t.parentId) (map[t.parentId] ||= []).push(t);
    }
    return map;
  }, [tasks]);

  const q = query.trim().toLowerCase();
  const matches = (t) =>
    !q ||
    (t.title || "").toLowerCase().includes(q) ||
    (t.description || "").toLowerCase().includes(q);

  const inSprint = (t) =>
    sprintFilter === "__all__" || (t.sprintId || "") === sprintFilter;

  const topLevel = tasks.filter((t) => !t.parentId && matches(t) && inSprint(t));

  // Empty columns say something different depending on why they're empty:
  // a search miss, a brand-new board, or just no cards at this stage yet.
  const emptyCopy = (col) => {
    if (q) return `No tasks match “${query.trim()}”. Try a shorter search.`;
    if (tasks.length === 0) return col.emptyFirst || col.empty;
    return col.empty;
  };

  // Reassign the dragged card (and its children) to another sprint.
  const dropOnSprint = (sprintId) => {
    const id = dragId;
    setDragId(null);
    if (!id) return;
    onChange(
      tasks.map((t) =>
        t.id === id || t.parentId === id
          ? { ...t, sprintId, updatedAt: Date.now() }
          : t
      )
    );
  };

  const drop = (colId) => {
    setOverCol(null);
    if (!dragId) return;
    const id = dragId;
    setDragId(null);
    const task = tasks.find((t) => t.id === id);
    if (!task || task.status === colId) return;
    onChange(tasks.map((t) => (t.id === id ? withStatus(t, colId) : t)));
  };

  // Enter and blur both submit, and closing the input on Enter triggers a
  // blur whose closure still holds the old title. The ref makes the
  // second call a no-op so a card is never added twice.
  const submitting = useRef(false);

  const submitAdd = (colId) => {
    if (submitting.current) return;
    submitting.current = true;
    setTimeout(() => {
      submitting.current = false;
    }, 0);

    const t = title.trim();
    setTitle("");
    setAdding(null);
    if (t)
      onAdd(t, {
        type: addType,
        status: colId,
        sprintId: sprintFilter === "__all__" ? "" : sprintFilter,
      });
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

  const openContextMenu = (e, task) => {
    setMenu({ x: e.clientX, y: e.clientY, task });
  };

  const closeMenu = useCallback(() => setMenu(null), []);
  const askDelete = (task) => setPendingDelete(task);

  return (
    <>
      <div className="kb-head" ref={headRef}>
        <SprintBar
          sprints={sprints}
          tasks={tasks}
          selected={sprintFilter}
          onSelect={onSprintFilter}
          onDropTask={dropOnSprint}
          onOpenRoadmap={onOpenRoadmap}
          onQuickAdd={onQuickAddSprint}
          onOpenAI={onOpenAI}
        />
        <div className="kb-search">
          <input
            className="kb-search-input"
            value={query}
            placeholder="Search tasks by title or description…"
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
      </div>
      <div className="kb-board">
      {columns.map((col) => {
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
                  placeholder={
                    col.id === "backlog" && addType === "story"
                      ? "What needs to happen?"
                      : `${TYPES.find((ty) => ty.id === addType)?.label || "Task"} title`
                  }
                  onChange={(e) => setTitle(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") submitAdd(col.id);
                    if (e.key === "Escape") setAdding(null);
                  }}
                  onBlur={() => submitAdd(col.id)}
                />
              </div>
            ) : (
              <button
                className="kb-add-btn"
                onClick={() => {
                  // Keep the type in sync with the label the user just clicked.
                  setAddType(col.id === "backlog" ? "story" : "task");
                  setAdding(col.id);
                }}
              >
                {col.id === "backlog" ? "+ Add story" : "+ Add task"}
              </button>
            )}

            <div className="kb-col-body">
              {items.map((t) => (
                <TaskCard
                  key={t.id}
                  task={t}
                  kids={childrenOf[t.id] || []}
                  dragging={dragId === t.id}
                  onOpen={onOpen}
                  onContextMenu={openContextMenu}
                  onToggleChild={toggleChild}
                  onDragStart={setDragId}
                />
              ))}
              {items.length === 0 && (
                <div className="kb-empty">{emptyCopy(col)}</div>
              )}
            </div>
          </div>
        );
      })}
      {onAddColumn && (
        <div className="kb-col kb-col-add">
          <button
            type="button"
            className="kb-add-column-btn"
            onClick={onAddColumn}
            title="Add a column and optionally map it to a GitHub status"
          >
            + Add column
          </button>
        </div>
      )}
      </div>

      {menu && (
        <TaskContextMenu
          x={menu.x}
          y={menu.y}
          task={menu.task}
          onOpen={onOpen}
          onEdit={onOpen}
          onDelete={askDelete}
          onClose={closeMenu}
        />
      )}

      {pendingDelete && (
        <ConfirmDialog
          title={`Delete ${pendingDelete.type === "story" ? "story" : "task"}?`}
          body={
            pendingDelete.type === "story"
              ? `Delete “${pendingDelete.title || "Untitled"}” and its child tasks? This cannot be undone.`
              : `Delete “${pendingDelete.title || "Untitled"}”? This cannot be undone.`
          }
          confirmLabel="Delete"
          danger
          onConfirm={() => {
            onDelete && onDelete(pendingDelete.id);
            setPendingDelete(null);
          }}
          onCancel={() => setPendingDelete(null)}
        />
      )}
    </>
  );
}
