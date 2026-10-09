import { useMemo, useState } from "react";

// Edit mode for the board's columns: drag to reorder, rename, add, and
// delete (choosing where a deleted column's tasks go). Nothing is saved
// until Done; Cancel discards every change.
export default function BoardLayoutEditor({ columns, tasks, statusMap = {}, onSave, onCancel, onError }) {
  const [cols, setCols] = useState(() => columns.map((c) => ({ id: c.id, label: c.label })));
  const [removed, setRemoved] = useState([]); // [{id,label}]
  const [moves, setMoves] = useState({}); // removedId -> destination id
  const [newLabel, setNewLabel] = useState("");
  const [dragIdx, setDragIdx] = useState(null);
  const [saving, setSaving] = useState(false);

  const counts = useMemo(() => {
    const m = {};
    for (const t of tasks) m[t.status] = (m[t.status] || 0) + 1;
    return m;
  }, [tasks]);

  const rename = (i, label) => setCols((cs) => cs.map((c, j) => (j === i ? { ...c, label } : c)));
  const remove = (i) => {
    const c = cols[i];
    setCols((cs) => cs.filter((_, j) => j !== i));
    if (c.id) setRemoved((r) => [...r, c]);
  };
  const restore = (id) => {
    const c = removed.find((r) => r.id === id);
    setRemoved((r) => r.filter((x) => x.id !== id));
    setMoves(({ [id]: _, ...rest }) => rest);
    if (c) setCols((cs) => [...cs, c]);
  };
  const add = () => {
    const label = newLabel.trim();
    if (!label) return;
    setCols((cs) => [...cs, { id: "", label, isNew: true }]);
    setNewLabel("");
  };
  const move = (from, to) => {
    if (from === null || from === to || to < 0) return;
    setCols((cs) => {
      if (to >= cs.length) return cs;
      const next = [...cs];
      const [m] = next.splice(from, 1);
      next.splice(to, 0, m);
      return next;
    });
  };
  const drop = (to) => {
    move(dragIdx, to);
    setDragIdx(null);
  };

  const keptIds = cols.filter((c) => c.id).map((c) => c.id);
  const problems = [
    ...(cols.length === 0 ? ["Keep at least one column."] : []),
    ...cols.filter((c) => !c.label.trim()).map(() => "Every column needs a name."),
    ...removed
      .filter((r) => counts[r.id] && !moves[r.id])
      .map((r) => `Choose where the ${counts[r.id]} task(s) in “${r.label}” should go.`),
  ];

  const done = async () => {
    if (problems.length) return;
    setSaving(true);
    try {
      await onSave(
        cols.map((c, i) => ({ id: c.id, label: c.label.trim(), order: i })),
        moves
      );
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="board-editor" role="region" aria-label="Edit board columns">
      <div className="board-editor-bar">
        <strong>Editing board</strong>
        <span className="board-editor-hint">Drag columns to reorder, rename them, add or delete. Nothing saves until Done.</span>
        <div className="spacer" />
        <button type="button" className="btn small" onClick={onCancel} disabled={saving}>
          Cancel
        </button>
        <button type="button" className="btn small primary" onClick={done} disabled={saving || problems.length > 0}>
          {saving ? "Saving…" : "Done"}
        </button>
      </div>

      <div className="board-editor-cols">
        {cols.map((c, i) => (
          <div
            key={c.id || `new-${i}`}
            className={`board-editor-col ${dragIdx === i ? "dragging" : ""}`}
            draggable
            onDragStart={() => setDragIdx(i)}
            onDragOver={(e) => e.preventDefault()}
            onDrop={(e) => {
              e.preventDefault();
              drop(i);
            }}
            onDragEnd={() => setDragIdx(null)}
            data-col={c.id}
          >
            <span className="board-editor-grip" aria-hidden title="Drag to reorder">
              ⋮⋮
            </span>
            <input
              aria-label={`Name of column ${i + 1}`}
              value={c.label}
              onChange={(e) => rename(i, e.target.value)}
            />
            <span className="board-editor-meta">
              {c.isNew ? "new" : `${counts[c.id] || 0} task${counts[c.id] === 1 ? "" : "s"}`}
              {statusMap[c.id] && <span className="kb-pill gh-ref" title="Synced with a GitHub status">GitHub</span>}
            </span>
            <div className="board-editor-move">
              <button type="button" className="btn tiny" disabled={i === 0} onClick={() => move(i, i - 1)} aria-label="Move left">
                ←
              </button>
              <button type="button" className="btn tiny" disabled={i === cols.length - 1} onClick={() => move(i, i + 1)} aria-label="Move right">
                →
              </button>
              <button type="button" className="link-btn danger" onClick={() => remove(i)}>
                Delete
              </button>
            </div>
          </div>
        ))}
        <div className="board-editor-col add">
          <input
            placeholder="New column name"
            value={newLabel}
            onChange={(e) => setNewLabel(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && add()}
          />
          <button type="button" className="btn small" onClick={add} disabled={!newLabel.trim()}>
            + Add column
          </button>
        </div>
      </div>

      {removed.length > 0 && (
        <div className="board-editor-removed">
          <strong>Deleting</strong>
          {removed.map((r) => (
            <div className="board-editor-removed-row" key={r.id}>
              <span>“{r.label}”</span>
              {counts[r.id] ? (
                <label>
                  Move its {counts[r.id]} task{counts[r.id] === 1 ? "" : "s"} to
                  <select
                    aria-label={`Destination for ${r.label}`}
                    value={moves[r.id] || ""}
                    onChange={(e) => setMoves((m) => ({ ...m, [r.id]: e.target.value }))}
                  >
                    <option value="">Choose a column…</option>
                    {cols.filter((c) => c.id && keptIds.includes(c.id)).map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.label}
                      </option>
                    ))}
                  </select>
                </label>
              ) : (
                <span className="board-editor-hint">No tasks.</span>
              )}
              {statusMap[r.id] && <span className="board-editor-warn">Its GitHub status mapping will be removed.</span>}
              <button type="button" className="link-btn" onClick={() => restore(r.id)}>
                Undo
              </button>
            </div>
          ))}
        </div>
      )}

      {problems.length > 0 && <p className="board-editor-problems">{problems[0]}</p>}
    </div>
  );
}
