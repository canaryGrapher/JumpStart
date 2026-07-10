import { useState } from "react";
import { COLUMNS } from "./columns";

const uid = () => crypto.randomUUID();
const PRIORITIES = ["", "low", "medium", "high"];

// Edit a task's details: description, subtasks, priority, labels, status.
export default function TaskDetailModal({ task, onSave, onDelete, onClose }) {
  const [draft, setDraft] = useState({ ...task });
  const [subTitle, setSubTitle] = useState("");
  const [labelText, setLabelText] = useState("");

  const set = (patch) => setDraft((d) => ({ ...d, ...patch }));

  const addSub = () => {
    const t = subTitle.trim();
    if (!t) return;
    setSubTitle("");
    set({ subtasks: [...(draft.subtasks || []), { id: uid(), title: t, done: false }] });
  };

  const toggleSub = (id) =>
    set({
      subtasks: draft.subtasks.map((s) => (s.id === id ? { ...s, done: !s.done } : s)),
    });

  const removeSub = (id) =>
    set({ subtasks: draft.subtasks.filter((s) => s.id !== id) });

  const addLabel = () => {
    const l = labelText.trim();
    if (!l || (draft.labels || []).includes(l)) return;
    setLabelText("");
    set({ labels: [...(draft.labels || []), l] });
  };

  const removeLabel = (l) =>
    set({ labels: draft.labels.filter((x) => x !== l) });

  const save = () => {
    if (!draft.title.trim()) return;
    onSave({ ...draft, done: draft.status === "done", updatedAt: Date.now() });
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal kb-detail" onClick={(e) => e.stopPropagation()}>
        <h2>Task details</h2>

        <div className="field">
          <label>Title</label>
          <input
            value={draft.title}
            onChange={(e) => set({ title: e.target.value })}
          />
        </div>

        <div className="kb-detail-row">
          <div className="field">
            <label>Status</label>
            <select
              value={draft.status}
              onChange={(e) => set({ status: e.target.value })}
            >
              {COLUMNS.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.label}
                </option>
              ))}
            </select>
          </div>
          <div className="field">
            <label>Priority</label>
            <select
              value={draft.priority || ""}
              onChange={(e) => set({ priority: e.target.value })}
            >
              {PRIORITIES.map((p) => (
                <option key={p} value={p}>
                  {p || "none"}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="field">
          <label>Description</label>
          <textarea
            rows={4}
            value={draft.description || ""}
            placeholder="Notes, acceptance criteria, links..."
            onChange={(e) => set({ description: e.target.value })}
          />
        </div>

        <div className="field">
          <label>Labels</label>
          <div className="kb-labels">
            {(draft.labels || []).map((l) => (
              <span className="kb-pill label" key={l}>
                {l}
                <button onClick={() => removeLabel(l)}>×</button>
              </span>
            ))}
          </div>
          <div className="row">
            <input
              value={labelText}
              placeholder="Add label..."
              onChange={(e) => setLabelText(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && addLabel()}
            />
            <button className="btn small" onClick={addLabel}>
              Add
            </button>
          </div>
        </div>

        <div className="field">
          <label>Subtasks</label>
          {(draft.subtasks || []).map((s) => (
            <div className={`task-row ${s.done ? "done" : ""}`} key={s.id}>
              <button className="task-check" onClick={() => toggleSub(s.id)}>
                {s.done ? "✓" : ""}
              </button>
              <span className="task-title">{s.title}</span>
              <button className="link-btn" onClick={() => removeSub(s.id)}>
                Remove
              </button>
            </div>
          ))}
          <div className="row">
            <input
              value={subTitle}
              placeholder="Add subtask..."
              onChange={(e) => setSubTitle(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && addSub()}
            />
            <button className="btn small" onClick={addSub}>
              Add
            </button>
          </div>
        </div>

        <div className="modal-actions kb-detail-actions">
          <button className="btn danger" onClick={() => onDelete(task.id)}>
            Delete task
          </button>
          <div className="spacer" />
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button
            className="btn primary"
            disabled={!draft.title.trim()}
            onClick={save}
          >
            Save
          </button>
        </div>
      </div>
    </div>
  );
}
