import { useState } from "react";
import { COLUMNS, TYPES, uid } from "./columns";
import { enrichTask, aiConfigured } from "../../ai";

const PRIORITIES = ["", "low", "medium", "high"];

// Edit a task or story: fields, checklists, child tasks, plus a
// one-click AI fill that drafts everything from the title.
export default function TaskDetailModal({
  task,
  tasks = [],
  sprints = [],
  projectName = "",
  onSave,
  onDelete,
  onOpen,
  onAddChild,
  onClose,
  onError,
}) {
  const [draft, setDraft] = useState({ ...task });
  const [subTitle, setSubTitle] = useState("");
  const [accTitle, setAccTitle] = useState("");
  const [labelText, setLabelText] = useState("");
  const [childTitle, setChildTitle] = useState("");
  const [aiBusy, setAiBusy] = useState(false);

  const set = (patch) => setDraft((d) => ({ ...d, ...patch }));
  const isStory = draft.type === "story";
  const parent = draft.parentId
    ? tasks.find((t) => t.id === draft.parentId)
    : null;
  const children = tasks.filter((t) => t.parentId === task.id);

  // --- checklist helpers (subtasks + acceptance criteria) ---
  const addItem = (key, text, clear) => {
    const t = text.trim();
    if (!t) return;
    clear("");
    set({ [key]: [...(draft[key] || []), { id: uid(), title: t, done: false }] });
  };
  const toggleItem = (key, id) =>
    set({
      [key]: draft[key].map((s) => (s.id === id ? { ...s, done: !s.done } : s)),
    });
  const removeItem = (key, id) =>
    set({ [key]: draft[key].filter((s) => s.id !== id) });

  const addLabel = () => {
    const l = labelText.trim();
    if (!l || (draft.labels || []).includes(l)) return;
    setLabelText("");
    set({ labels: [...(draft.labels || []), l] });
  };
  const removeLabel = (l) =>
    set({ labels: draft.labels.filter((x) => x !== l) });

  const toChecklist = (arr) =>
    (arr || []).map((t) =>
      typeof t === "string" ? { id: uid(), title: t, done: false } : t
    );

  const fillWithAI = async () => {
    if (!draft.title.trim()) return;
    if (!aiConfigured()) {
      onError && onError("Pick an Ollama model in Preferences → AI first.");
      return;
    }
    setAiBusy(true);
    try {
      const r = await enrichTask(draft.title, draft.type, projectName);
      set({
        description: r.description || draft.description,
        acceptance: [...(draft.acceptance || []), ...toChecklist(r.acceptance)],
        subtasks: [...(draft.subtasks || []), ...toChecklist(r.subtasks)],
        priority: r.priority || draft.priority,
        labels: Array.from(
          new Set([...(draft.labels || []), ...(r.labels || [])])
        ),
      });
    } catch (e) {
      onError && onError(String(e));
    } finally {
      setAiBusy(false);
    }
  };

  const addChild = () => {
    const t = childTitle.trim();
    if (!t || !onAddChild) return;
    setChildTitle("");
    onAddChild(task.id, t);
  };

  const save = () => {
    if (!draft.title.trim()) return;
    onSave({ ...draft, done: draft.status === "done", updatedAt: Date.now() });
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal kb-detail" onClick={(e) => e.stopPropagation()}>
        <div className="kb-detail-head">
          <h2>{isStory ? "Story" : "Task"} details</h2>
          <button
            className="btn ai small"
            onClick={fillWithAI}
            disabled={aiBusy || !draft.title.trim()}
          >
            {aiBusy ? "Thinking…" : "✨ Fill with AI"}
          </button>
        </div>

        {parent && (
          <div className="kb-parent-note">
            Part of story: <strong>{parent.title}</strong>
          </div>
        )}

        <div className="field">
          <label>Title</label>
          <input
            value={draft.title}
            onChange={(e) => set({ title: e.target.value })}
          />
        </div>

        <div className="kb-detail-row">
          <div className="field">
            <label>Type</label>
            <select
              value={draft.type || "task"}
              onChange={(e) => set({ type: e.target.value })}
            >
              {TYPES.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.label}
                </option>
              ))}
            </select>
          </div>
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
          <label>Sprint</label>
          <select
            value={draft.sprintId || ""}
            onChange={(e) => set({ sprintId: e.target.value })}
          >
            <option value="">Backlog</option>
            {sprints.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </select>
        </div>

        {isStory && (
          <div className="kb-detail-row">
            <div className="field">
              <label>Story points</label>
              <input
                type="number"
                min="0"
                value={draft.storyPoints || 0}
                onChange={(e) =>
                  set({ storyPoints: parseInt(e.target.value, 10) || 0 })
                }
              />
            </div>
            <div className="field">
              <label>Assignee</label>
              <input
                value={draft.assignee || ""}
                placeholder="Name..."
                onChange={(e) => set({ assignee: e.target.value })}
              />
            </div>
          </div>
        )}

        <div className="field">
          <label>Description</label>
          <textarea
            rows={4}
            value={draft.description || ""}
            placeholder={
              isStory
                ? "As a <role>, I want <goal>, so that <benefit>..."
                : "Notes, acceptance criteria, links..."
            }
            onChange={(e) => set({ description: e.target.value })}
          />
        </div>

        {isStory && (
          <div className="field">
            <label>Acceptance criteria</label>
            {(draft.acceptance || []).map((s) => (
              <div className={`task-row ${s.done ? "done" : ""}`} key={s.id}>
                <button
                  className="task-check"
                  onClick={() => toggleItem("acceptance", s.id)}
                >
                  {s.done ? "✓" : ""}
                </button>
                <span className="task-title">{s.title}</span>
                <button
                  className="link-btn"
                  onClick={() => removeItem("acceptance", s.id)}
                >
                  Remove
                </button>
              </div>
            ))}
            <div className="row">
              <input
                value={accTitle}
                placeholder="Add criterion..."
                onChange={(e) => setAccTitle(e.target.value)}
                onKeyDown={(e) =>
                  e.key === "Enter" && addItem("acceptance", accTitle, setAccTitle)
                }
              />
              <button
                className="btn small"
                onClick={() => addItem("acceptance", accTitle, setAccTitle)}
              >
                Add
              </button>
            </div>
          </div>
        )}

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
              <button
                className="task-check"
                onClick={() => toggleItem("subtasks", s.id)}
              >
                {s.done ? "✓" : ""}
              </button>
              <span className="task-title">{s.title}</span>
              <button
                className="link-btn"
                onClick={() => removeItem("subtasks", s.id)}
              >
                Remove
              </button>
            </div>
          ))}
          <div className="row">
            <input
              value={subTitle}
              placeholder="Add subtask..."
              onChange={(e) => setSubTitle(e.target.value)}
              onKeyDown={(e) =>
                e.key === "Enter" && addItem("subtasks", subTitle, setSubTitle)
              }
            />
            <button
              className="btn small"
              onClick={() => addItem("subtasks", subTitle, setSubTitle)}
            >
              Add
            </button>
          </div>
        </div>

        {isStory && (
          <div className="field">
            <label>Child tasks ({children.length})</label>
            {children.map((c) => (
              <div className={`task-row ${c.status === "done" ? "done" : ""}`} key={c.id}>
                <span className={`kb-pill status-${c.status}`}>{c.status}</span>
                <span
                  className="task-title link"
                  onClick={() => onOpen && onOpen(c)}
                >
                  {c.title}
                </span>
              </div>
            ))}
            <div className="row">
              <input
                value={childTitle}
                placeholder="Add child task..."
                onChange={(e) => setChildTitle(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && addChild()}
              />
              <button className="btn small" onClick={addChild}>
                Add
              </button>
            </div>
          </div>
        )}

        <div className="modal-actions kb-detail-actions">
          <button className="btn danger" onClick={() => onDelete(task.id)}>
            Delete {isStory ? "story" : "task"}
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
