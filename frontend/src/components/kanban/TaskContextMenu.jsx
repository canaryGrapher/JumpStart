import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";

const MENU_PAD = 8;

// Portaled right-click menu for a kanban card. Positioned at the cursor
// and clamped so it stays inside the viewport.
export default function TaskContextMenu({ x, y, task, onOpen, onEdit, onDelete, onClose }) {
  const ref = useRef(null);
  const [pos, setPos] = useState({ top: y, left: x });

  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const rect = el.getBoundingClientRect();
    const left = Math.min(x, window.innerWidth - rect.width - MENU_PAD);
    const top = Math.min(y, window.innerHeight - rect.height - MENU_PAD);
    setPos({
      top: Math.max(MENU_PAD, top),
      left: Math.max(MENU_PAD, left),
    });
  }, [x, y, task?.id]);

  useEffect(() => {
    const onDoc = (e) => {
      if (ref.current?.contains(e.target)) return;
      onClose();
    };
    const onKey = (e) => {
      if (e.key === "Escape") onClose();
    };
    const onScroll = () => onClose();
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    window.addEventListener("scroll", onScroll, true);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
      window.removeEventListener("scroll", onScroll, true);
    };
  }, [onClose]);

  if (!task) return null;

  const run = (fn) => {
    fn && fn(task);
    onClose();
  };

  return createPortal(
    <div
      className="task-context-menu"
      ref={ref}
      role="menu"
      style={{ top: pos.top, left: pos.left }}
      onContextMenu={(e) => e.preventDefault()}
    >
      <button type="button" className="task-context-item" role="menuitem" onClick={() => run(onOpen)}>
        Open
      </button>
      <button type="button" className="task-context-item" role="menuitem" onClick={() => run(onEdit)}>
        Edit
      </button>
      <div className="task-context-sep" />
      <button
        type="button"
        className="task-context-item danger"
        role="menuitem"
        onClick={() => run(onDelete)}
      >
        Delete
      </button>
    </div>,
    document.body
  );
}
