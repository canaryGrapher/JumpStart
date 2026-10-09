import { useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";
import { useSavedFilters } from "../kanban/SavedFilters";
import TaskWidget from "./TaskWidget";
import HtmlWidget from "./HtmlWidget";

export const DUE_RANGES = [
  ["overdue", "Past due"],
  ["today", "Due today"],
  ["tomorrow", "Due tomorrow"],
  ["this_week", "Due this week"],
  ["next_week", "Due next week"],
  ["this_month", "Due this month"],
  ["next_month", "Due next month"],
  ["this_quarter", "Due this quarter"],
  ["next_quarter", "Due next quarter"],
  ["this_year", "Due this year"],
  ["no_date", "No due date"],
];

export const BUILTINS = [
  ["stats", "Overview tiles", "Projects, CPU, memory, tasks, processes, ports"],
  ["flow", "Task flow", "Kanban pipeline across every project"],
  ["donut", "Completion mix", "Share of work by column"],
  ["recent", "Recent projects", "Pick up where you left off"],
  ["activity", "Activity by project", "Your most-used workspaces"],
  ["ports", "Live ports", "Listening ports from managed processes"],
  ["import", "Config import", "Add projects from JSON"],
];

const DISPLAYS = [
  ["list", "List"],
  ["count", "Count"],
  ["bar", "Bar chart"],
  ["donut", "Donut"],
  ["table", "Table"],
];
const GROUPS = [
  ["status", "Status"],
  ["priority", "Priority"],
  ["type", "Type"],
  ["assignee", "Assignee"],
  ["label", "Label"],
  ["project", "Project"],
  ["sprint", "Sprint"],
  ["due", "Due date"],
];
const DUE_PRESETS = [
  ["", "Any date"],
  ["today", "Today"],
  ["tomorrow", "Tomorrow"],
  ["this_week", "This week"],
  ["next_week", "Next week"],
  ["this_month", "This month"],
  ["next_month", "Next month"],
  ["this_quarter", "This quarter"],
  ["next_quarter", "Next quarter"],
  ["this_year", "This year"],
];

const HTML_EXAMPLE = `<style>
  .n { font-size: 34px; font-weight: 700; }
  li { cursor: pointer; }
  li:hover { color: var(--accent); }
</style>
<div class="n" id="n">…</div>
<ul id="list"></ul>
<script>
  addEventListener("message", (e) => {
    if (e.data.type !== "jumpstart:data") return;
    const { total, tasks } = e.data.data;
    n.textContent = total + " tasks";
    list.replaceChildren(...tasks.slice(0, 5).map((t) => {
      const li = document.createElement("li");
      li.textContent = t.title + (t.dueDate ? " · " + t.dueDate : "");
      li.onclick = () => jumpstart.open(t.projectId, t.taskId);
      return li;
    }));
  });
</script>`;

const csv = (list) => (list || []).join(", ");
const fromCsv = (s) => s.split(",").map((x) => x.trim()).filter(Boolean);

// A small form for the task query behind custom and code widgets, with a
// JSON escape hatch for every TaskQuery field.
function QueryFields({ query = {}, onChange }) {
  const [raw, setRaw] = useState(null); // string while editing JSON
  const [rawError, setRawError] = useState("");
  const set = (k, v) => {
    const next = { ...query, [k]: v };
    if (v === "" || v === false || (Array.isArray(v) && !v.length)) delete next[k];
    onChange(next);
  };
  const toggle = (k, v) => set(k, (query[k] || []).includes(v) ? query[k].filter((x) => x !== v) : [...(query[k] || []), v]);
  if (raw !== null) {
    return (
      <div className="we-field">
        <label htmlFor="we-query-json">Query (JSON)</label>
        <textarea id="we-query-json" rows={6} value={raw} spellCheck={false} onChange={(e) => setRaw(e.target.value)} />
        {rawError && <span className="we-error">{rawError}</span>}
        <span className="prefs-inline">
          <button type="button" className="btn small" onClick={() => {
            try {
              const q = JSON.parse(raw || "{}");
              if (typeof q !== "object" || Array.isArray(q)) throw new Error("expected an object");
              onChange(q);
              setRaw(null);
              setRawError("");
            } catch (e) {
              setRawError(String(e.message || e));
            }
          }}>Apply JSON</button>
          <button type="button" className="link-btn" onClick={() => setRaw(null)}>Back to form</button>
        </span>
      </div>
    );
  }
  return (
    <fieldset className="we-query">
      <legend>Which tasks</legend>
      <div className="we-row">
        <label htmlFor="we-due">Due</label>
        <select id="we-due" value={query.duePreset || ""} onChange={(e) => set("duePreset", e.target.value)}>
          {DUE_PRESETS.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
        </select>
        <label className="we-check"><input type="checkbox" checked={!!query.overdue} onChange={(e) => set("overdue", e.target.checked)} /> Past due</label>
        <label className="we-check"><input type="checkbox" checked={!!query.noDueDate} onChange={(e) => set("noDueDate", e.target.checked)} /> No due date</label>
      </div>
      <div className="we-row">
        <span className="we-label">Priority</span>
        {["high", "medium", "low"].map((p) => (
          <label className="we-check" key={p}><input type="checkbox" checked={(query.priorities || []).includes(p)} onChange={() => toggle("priorities", p)} /> {p}</label>
        ))}
      </div>
      <div className="we-row">
        <span className="we-label">Type</span>
        {["story", "task", "bug"].map((p) => (
          <label className="we-check" key={p}><input type="checkbox" checked={(query.types || []).includes(p)} onChange={() => toggle("types", p)} /> {p}</label>
        ))}
      </div>
      <div className="we-row">
        <label htmlFor="we-assignees">Assignees</label>
        <input id="we-assignees" placeholder="Sam, Alex" defaultValue={csv(query.assignees)} onBlur={(e) => set("assignees", fromCsv(e.target.value))} />
      </div>
      <div className="we-row">
        <label htmlFor="we-labels">Labels</label>
        <input id="we-labels" placeholder="stripe, blocked" defaultValue={csv(query.labels)} onBlur={(e) => set("labels", fromCsv(e.target.value))} />
      </div>
      <div className="we-row">
        <label htmlFor="we-statuses">Statuses</label>
        <input id="we-statuses" placeholder="todo, inprogress" defaultValue={csv(query.statuses)} onBlur={(e) => set("statuses", fromCsv(e.target.value))} />
      </div>
      <div className="we-row">
        <label htmlFor="we-text">Text contains</label>
        <input id="we-text" defaultValue={query.text || ""} onBlur={(e) => set("text", e.target.value.trim())} />
      </div>
      <button type="button" className="link-btn" onClick={() => setRaw(JSON.stringify(query, null, 2))}>Edit as JSON</button>
    </fieldset>
  );
}

// Add a widget (gallery first) or edit an existing one, with a live
// preview of the draft.
export default function WidgetEditor({ initial, projects, onSave, onClose }) {
  const [draft, setDraft] = useState(initial);
  const [error, setError] = useState("");
  const filters = useSavedFilters(draft?.projectId || "");
  const allFilters = useMemo(
    () => [...filters.project.map((f) => ({ ...f, scope: "project" })), ...filters.appWide.map((f) => ({ ...f, scope: "app" }))],
    [filters.project, filters.appWide]
  );
  const set = (patch) => setDraft((d) => ({ ...d, ...patch }));
  const isTask = draft && ["due", "filter", "custom", "html"].includes(draft.type);
  const needsGroup = draft && ["bar", "donut", "table"].includes(draft.display);

  const pick = (w) => setDraft({ size: "m", ...w });

  const save = () => {
    if (draft.type === "filter" && !draft.filterId) return setError("Pick a saved filter.");
    if (draft.type === "html" && !draft.html?.trim()) return setError("Add the widget's HTML.");
    if (needsGroup && !draft.groupBy) return setError("Charts and tables need “Group by”.");
    const clean = { ...draft };
    if (clean.type === "custom" && !clean.query) clean.query = {};
    onSave(clean);
  };

  useEffect(() => {
    const onKey = (e) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  // Portaled to <body>: the dashboard's animated container would otherwise
  // become the containing block for the fixed overlay.
  return createPortal(
    <div className="modal-overlay" onMouseDown={onClose}>
      <div className="modal we-modal" role="dialog" aria-label={initial ? "Edit widget" : "Add widget"} onMouseDown={(e) => e.stopPropagation()}>
        <h2>{initial ? "Edit widget" : "Add widget"}</h2>
        {!draft ? (
          <div className="we-gallery">
            <section>
              <h3>Due dates</h3>
              <div className="we-cards">
                {DUE_RANGES.map(([r, label]) => (
                  <button type="button" key={r} className="we-card" onClick={() => pick({ type: "due", range: r, size: "s" })}>
                    <strong>{label}</strong>
                    <span>Open tasks across your projects</span>
                  </button>
                ))}
              </div>
            </section>
            <section>
              <h3>Your own</h3>
              <div className="we-cards">
                <button type="button" className="we-card" onClick={() => pick({ type: "filter", display: "list" })}>
                  <strong>Saved filter</strong>
                  <span>Any saved filter as a list or chart</span>
                </button>
                <button type="button" className="we-card" onClick={() => pick({ type: "custom", title: "Custom", display: "bar", groupBy: "status", query: {} })}>
                  <strong>Custom chart or list</strong>
                  <span>Pick tasks, then show a count, list, bar, donut or table</span>
                </button>
                <button type="button" className="we-card" onClick={() => pick({ type: "html", title: "Custom code", html: HTML_EXAMPLE, query: { overdue: true } })}>
                  <strong>Code widget (HTML/JS)</strong>
                  <span>Runs sandboxed: no network or app access</span>
                </button>
              </div>
            </section>
            <section>
              <h3>Built-in panels</h3>
              <div className="we-cards">
                {BUILTINS.map(([t, label, hint]) => (
                  <button type="button" key={t} className="we-card" onClick={() => pick({ type: t, size: t === "ports" || t === "import" || t === "stats" ? "l" : "m" })}>
                    <strong>{label}</strong>
                    <span>{hint}</span>
                  </button>
                ))}
              </div>
            </section>
          </div>
        ) : (
          <div className="we-body">
            <div className="we-form">
              <div className="we-row">
                <label htmlFor="we-title">Title</label>
                <input id="we-title" value={draft.title || ""} placeholder="Default" onChange={(e) => set({ title: e.target.value })} />
              </div>
              <div className="we-row">
                <span className="we-label">Size</span>
                <div className="seg" role="radiogroup" aria-label="Size">
                  {[["s", "Small"], ["m", "Medium"], ["l", "Large"]].map(([v, l]) => (
                    <button type="button" key={v} role="radio" aria-checked={draft.size === v} className={draft.size === v ? "on" : ""} onClick={() => set({ size: v })}>{l}</button>
                  ))}
                </div>
              </div>
              {isTask && (
                <div className="we-row">
                  <label htmlFor="we-project">Project</label>
                  <select id="we-project" value={draft.projectId || ""} onChange={(e) => set({ projectId: e.target.value || undefined })}>
                    <option value="">All projects</option>
                    {projects.filter((p) => p.tasksEnabled || (p.tasks || []).length).map((p) => (
                      <option key={p.id} value={p.id}>{p.name}</option>
                    ))}
                  </select>
                </div>
              )}
              {draft.type === "due" && (
                <div className="we-row">
                  <label htmlFor="we-range">Range</label>
                  <select id="we-range" value={draft.range} onChange={(e) => set({ range: e.target.value })}>
                    {DUE_RANGES.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
                  </select>
                </div>
              )}
              {draft.type === "filter" && (
                <div className="we-row">
                  <label htmlFor="we-filter">Saved filter</label>
                  <select id="we-filter" value={draft.filterId || ""} onChange={(e) => set({ filterId: e.target.value })}>
                    <option value="">Choose…</option>
                    {allFilters.map((f) => (
                      <option key={f.id} value={f.id}>{f.name}{f.scope === "project" ? " (project)" : ""}</option>
                    ))}
                  </select>
                </div>
              )}
              {(draft.type === "custom" || draft.type === "html") && (
                <QueryFields query={draft.query || {}} onChange={(q) => set({ query: q })} />
              )}
              {isTask && draft.type !== "html" && (
                <>
                  <div className="we-row">
                    <label htmlFor="we-display">Show as</label>
                    <select id="we-display" value={draft.display || "list"} onChange={(e) => set({ display: e.target.value, groupBy: draft.groupBy || (e.target.value === "list" ? undefined : "status") })}>
                      {DISPLAYS.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
                    </select>
                    {(draft.display && draft.display !== "list") && (
                      <>
                        <label htmlFor="we-group">Group by</label>
                        <select id="we-group" value={draft.groupBy || ""} onChange={(e) => set({ groupBy: e.target.value || undefined })}>
                          {draft.display === "count" && <option value="">Nothing</option>}
                          {GROUPS.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
                        </select>
                      </>
                    )}
                  </div>
                </>
              )}
              {isTask && (
                <label className="we-check">
                  <input type="checkbox" checked={!!draft.includeDone} onChange={(e) => set({ includeDone: e.target.checked })} /> Include done tasks
                </label>
              )}
              {draft.type === "html" && (
                <div className="we-field">
                  <label htmlFor="we-html">HTML / JS</label>
                  <textarea id="we-html" rows={12} spellCheck={false} value={draft.html || ""} onChange={(e) => set({ html: e.target.value })} />
                  <span className="prefs-hint">
                    Runs in a sandbox with no network, storage or app access. It receives the tasks chosen above as
                    {" "}<code>{"{type: \"jumpstart:data\", data}"}</code> messages and can call <code>jumpstart.open(projectId, taskId)</code>.
                  </span>
                </div>
              )}
              {error && <p className="we-error" role="alert">{error}</p>}
            </div>
            {isTask && (
              <div className="we-preview" aria-label="Preview">
                <span className="we-label">Preview</span>
                <div className={`panel dash-widget-panel`}>
                  {draft.type === "html" ? <HtmlWidget widget={draft} version={0} /> : (draft.type !== "filter" || draft.filterId) ? <TaskWidget widget={draft} version={0} /> : <div className="dash-empty-inline">Pick a saved filter.</div>}
                </div>
              </div>
            )}
          </div>
        )}
        <div className="modal-actions">
          {draft && !initial && <button type="button" className="link-btn" onClick={() => setDraft(null)}>Back</button>}
          <span style={{ flex: 1 }} />
          <button type="button" className="btn" onClick={onClose}>Cancel</button>
          {draft && <button type="button" className="btn primary" onClick={save}>{initial ? "Save widget" : "Add widget"}</button>}
        </div>
      </div>
    </div>,
    document.body
  );
}
