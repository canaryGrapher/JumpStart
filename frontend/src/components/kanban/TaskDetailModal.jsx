import { useEffect, useState } from "react";
import { COLUMNS, TYPES, uid } from "./columns";
import { enrichTask, aiConfigured } from "../../ai";
import { track } from "../../analytics";
import { GitHubListAssignableUsers } from "../../api";
import GitHubFields from "../github/GitHubFields";
import AssigneeSelect from "./AssigneeSelect";

const PRIORITIES = ["", "low", "medium", "high"];

// Edit a task or story: fields, checklists, child tasks, plus a
// one-click AI fill that expands what the user has already written.
export default function TaskDetailModal({
  task,
  tasks = [],
  sprints = [],
  projectId = "",
  sync = null,
  onSave,
  onDelete,
  onOpen,
  onAddChild,
  onClose,
  onError,
  onGitHubChange,
}) {
  const [draft, setDraft] = useState({ ...task });
  const [subTitle, setSubTitle] = useState("");
  const [accTitle, setAccTitle] = useState("");
  const [labelText, setLabelText] = useState("");
  const [childTitle, setChildTitle] = useState("");
  const [aiBusy, setAiBusy] = useState(false);
  // Whether AI fill ran in this editing session, and whether the user then
  // kept any of it. Generation counts only tell us the feature runs;
  // acceptance rate is what tells us it works.
  const [aiFilled, setAiFilled] = useState(false);
  // AI-proposed subtasks stay here until the user accepts or dismisses
  // them, so Populate with AI never silently grows the real checklist.
  const [suggestedSubtasks, setSuggestedSubtasks] = useState([]);
  const [assignees, setAssignees] = useState([]);

  const set = (patch) => setDraft((d) => ({ ...d, ...patch }));
  const isStory = draft.type === "story";
  const parent = draft.parentId
    ? tasks.find((t) => t.id === draft.parentId)
    : null;
  const children = tasks.filter((t) => t.parentId === task.id);

  // Sync can attach a GitHub link after the modal opened. Fold it into
  // the draft so Save cannot wipe the link and create a duplicate card.
  useEffect(() => {
    if (!task?.github?.itemId) return;
    setDraft((d) => {
      if (d.github?.itemId) return d;
      return { ...d, github: task.github };
    });
  }, [task?.github?.itemId, task?.github]);

  useEffect(() => {
    let live = true;
    if (!projectId || !sync?.repo) {
      setAssignees([]);
      return undefined;
    }
    GitHubListAssignableUsers(projectId)
      .then((list) => live && setAssignees(list || []))
      .catch(() => live && setAssignees([]));
    return () => {
      live = false;
    };
  }, [projectId, sync?.repo]);

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
  const renameItem = (key, id, title) =>
    set({
      [key]: draft[key].map((s) => (s.id === id ? { ...s, title } : s)),
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

  const acceptSuggestion = (id) => {
    const item = suggestedSubtasks.find((s) => s.id === id);
    if (!item) return;
    setSuggestedSubtasks((list) => list.filter((s) => s.id !== id));
    set({ subtasks: [...(draft.subtasks || []), { ...item, done: false }] });
  };
  const dismissSuggestion = (id) =>
    setSuggestedSubtasks((list) => list.filter((s) => s.id !== id));
  const acceptAllSuggestions = () => {
    if (!suggestedSubtasks.length) return;
    set({
      subtasks: [
        ...(draft.subtasks || []),
        ...suggestedSubtasks.map((s) => ({ ...s, done: false })),
      ],
    });
    setSuggestedSubtasks([]);
  };
  const dismissAllSuggestions = () => setSuggestedSubtasks([]);

  const fillWithAI = async () => {
    if (!draft.title.trim()) return;
    if (!aiConfigured()) {
      onError && onError("Pick an Ollama model in Preferences → AI first.");
      return;
    }
    setAiBusy(true);
    try {
      // Send the body too: the model expands on what the user wrote
      // instead of guessing a scope from the title alone.
      const r = await enrichTask(
        draft.title,
        draft.description || "",
        draft.type,
        projectId
      );
      set({
        description: r.description || draft.description,
        acceptance: [...(draft.acceptance || []), ...toChecklist(r.acceptance)],
        priority: r.priority || draft.priority,
        labels: Array.from(
          new Set([...(draft.labels || []), ...(r.labels || [])])
        ),
        storyPoints:
          r.storyPoints > 0 ? r.storyPoints : draft.storyPoints || 0,
      });
      // Subtasks are suggestions only — the user accepts or dismisses.
      const existing = new Set(
        (draft.subtasks || []).map((s) => s.title.trim().toLowerCase())
      );
      const suggestions = toChecklist(r.subtasks).filter(
        (s) => s.title.trim() && !existing.has(s.title.trim().toLowerCase())
      );
      setSuggestedSubtasks(suggestions);
      setAiFilled(true);
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
    // Saving after an AI fill is the closest thing to an explicit "keep"
    // for description / acceptance / labels. Suggested subtasks still need
    // an Accept click; only those already on the task count here.
    if (aiFilled) {
      track("ai_suggestion_accepted", {
        surface: "task_enrich",
        kind: draft.type === "story" ? "story" : "task",
        acceptance_count: (draft.acceptance || []).length,
        subtask_count: (draft.subtasks || []).length,
      });
    }
    // Prefer the live board copy's GitHub link when the draft never saw it
    // (race between first sync and Save).
    const live = tasks.find((t) => t.id === task.id);
    const github = draft.github?.itemId
      ? draft.github
      : live?.github?.itemId
        ? live.github
        : draft.github || task.github;
    onSave({
      ...draft,
      github,
      done: draft.status === "done",
      updatedAt: Date.now(),
    });
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
            title={
              draft.description?.trim()
                ? "Expand this title and description into criteria and subtasks"
                : "Draft criteria and subtasks from the title — add a description first for a better result"
            }
          >
            {aiBusy ? "Thinking…" : "✨ Populate with AI"}
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
          </div>
        )}

        <div className="field">
          <label>Assignees</label>
          <AssigneeSelect
            value={draft.assignee || ""}
            users={assignees}
            onChange={(assignee) => set({ assignee })}
            placeholder={
              sync?.repo ? "Select from the team…" : "Add people…"
            }
          />
        </div>

        <div className="field">
          <label>Description</label>
          <textarea
            rows={4}
            value={draft.description || ""}
            placeholder={
              isStory
                ? "As a <role>, I want <goal>, so that <benefit>…"
                : "Notes, links, context…"
            }
            onChange={(e) => set({ description: e.target.value })}
          />
        </div>

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
              <input
                className="task-title"
                value={s.title}
                onChange={(e) =>
                  renameItem("acceptance", s.id, e.target.value)
                }
              />
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
              placeholder="Add criterion…"
              onChange={(e) => setAccTitle(e.target.value)}
              onKeyDown={(e) =>
                e.key === "Enter" &&
                addItem("acceptance", accTitle, setAccTitle)
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
              placeholder="Add label…"
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
              <input
                className="task-title"
                value={s.title}
                onChange={(e) => renameItem("subtasks", s.id, e.target.value)}
              />
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
              placeholder="Add subtask…"
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

        {suggestedSubtasks.length > 0 && (
          <div className="field kb-suggested">
            <div className="kb-suggested-head">
              <label>Suggested subtasks</label>
              <div className="kb-suggested-actions">
                <button
                  type="button"
                  className="link-btn"
                  onClick={acceptAllSuggestions}
                >
                  Accept all
                </button>
                <button
                  type="button"
                  className="link-btn"
                  onClick={dismissAllSuggestions}
                >
                  Dismiss all
                </button>
              </div>
            </div>
            <p className="kb-suggested-hint">
              From AI — accept to add, or dismiss to drop.
            </p>
            {suggestedSubtasks.map((s) => (
              <div className="task-row suggested" key={s.id}>
                <span className="task-title">{s.title}</span>
                <button
                  type="button"
                  className="btn tiny"
                  onClick={() => acceptSuggestion(s.id)}
                >
                  Accept
                </button>
                <button
                  type="button"
                  className="link-btn"
                  onClick={() => dismissSuggestion(s.id)}
                >
                  Remove
                </button>
              </div>
            ))}
          </div>
        )}

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
                placeholder="Add child task…"
                onChange={(e) => setChildTitle(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && addChild()}
              />
              <button className="btn small" onClick={addChild}>
                Add
              </button>
            </div>
          </div>
        )}

        <GitHubFields
          task={task}
          sync={sync}
          projectId={projectId}
          onError={onError}
          onGitHubChange={(gh) => {
            setDraft((d) => ({ ...d, github: gh }));
            onGitHubChange && onGitHubChange(task.id, gh);
          }}
        />

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
