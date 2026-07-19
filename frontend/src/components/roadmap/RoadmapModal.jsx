import { useState } from "react";
import { blankSprint, moveSprint, resequence, sprintStats } from "../kanban/sprints";
import RoadmapRow from "./RoadmapRow";

// Roadmap builder: drag sprints to sequence them, edit dates and goals
// inline. Order in this list is the order sprints run in.
export default function RoadmapModal({ sprints, tasks, onSave, onClose }) {
  const [draft, setDraft] = useState(sprints);
  const [editing, setEditing] = useState(null);
  const [dragIdx, setDragIdx] = useState(null);
  const [overIdx, setOverIdx] = useState(null);

  const patch = (id, p) =>
    setDraft((d) => d.map((s) => (s.id === id ? { ...s, ...p } : s)));

  const add = () => {
    const s = blankSprint("", draft.length);
    setDraft([...draft, s]);
    setEditing(s.id);
  };

  // Deleting a sprint leaves its tasks; they fall back to the backlog.
  const remove = (id) => {
    setDraft(resequence(draft.filter((s) => s.id !== id)));
    setEditing(null);
  };

  const drop = (to) => {
    if (dragIdx !== null) setDraft(moveSprint(draft, dragIdx, to));
    setDragIdx(null);
    setOverIdx(null);
  };

  const unassigned = tasks.filter((t) => !t.sprintId).length;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal rm-modal" onClick={(e) => e.stopPropagation()}>
        <div className="rm-head">
          <h2>Roadmap</h2>
          <span className="rm-hint">
            Drag to sequence · {draft.length} sprints · {unassigned} in backlog
          </span>
        </div>

        <div className="rm-list">
          {draft.map((s, i) => (
            <RoadmapRow
              key={s.id}
              sprint={s}
              index={i}
              stats={sprintStats(tasks, s.id)}
              editing={editing === s.id}
              dragging={dragIdx === i}
              dropBefore={overIdx === i && dragIdx !== null && dragIdx !== i}
              onEdit={setEditing}
              onPatch={(p) => patch(s.id, p)}
              onDelete={() => remove(s.id)}
              onDragStart={setDragIdx}
              onDragOver={setOverIdx}
              onDrop={drop}
            />
          ))}
          {draft.length === 0 && (
            <div className="rm-empty">
              No sprints yet. Add one to start sequencing your roadmap.
            </div>
          )}
        </div>

        <button className="rm-add" onClick={add}>
          + Add sprint
        </button>

        <div className="modal-actions">
          <div className="spacer" />
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn primary" onClick={() => onSave(resequence(draft))}>
            Save roadmap
          </button>
        </div>
      </div>
    </div>
  );
}
