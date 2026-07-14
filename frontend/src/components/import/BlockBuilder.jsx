import { emptyProject, emptyProc } from "./configModel";

// Interactive block builder: edit projects, their processes (with env
// rows) and seed tasks through form fields. Emits the full projects
// array up on every change.
export default function BlockBuilder({ projects, onChange }) {
  const setProject = (i, patch) =>
    onChange(projects.map((p, idx) => (idx === i ? { ...p, ...patch } : p)));

  const removeProject = (i) => onChange(projects.filter((_, idx) => idx !== i));

  const addProject = () => onChange([...projects, emptyProject()]);

  return (
    <div className="blocks">
      {projects.length === 0 && (
        <div className="blocks-empty">
          No projects yet. Add one below, or paste JSON in the JSON tab.
        </div>
      )}

      {projects.map((p, i) => (
        <ProjectBlock
          key={i}
          project={p}
          onChange={(patch) => setProject(i, patch)}
          onRemove={() => removeProject(i)}
        />
      ))}

      <button className="btn add-block" onClick={addProject}>
        + Add project
      </button>
    </div>
  );
}

function ProjectBlock({ project, onChange, onRemove }) {
  const procs = project.processes || [];
  const tasks = project.tasks || [];

  const setProc = (i, patch) =>
    onChange({ processes: procs.map((x, idx) => (idx === i ? { ...x, ...patch } : x)) });
  const addProc = () => onChange({ processes: [...procs, emptyProc()] });
  const removeProc = (i) => onChange({ processes: procs.filter((_, idx) => idx !== i) });

  const setTask = (i, patch) =>
    onChange({ tasks: tasks.map((x, idx) => (idx === i ? { ...x, ...patch } : x)) });
  const addTask = () => onChange({ tasks: [...tasks, { title: "", done: false }] });
  const removeTask = (i) => onChange({ tasks: tasks.filter((_, idx) => idx !== i) });

  return (
    <div className="block project-block">
      <div className="block-head">
        <span className="block-tag">Project</span>
        <input
          className="block-title-input"
          placeholder="Project name"
          value={project.name}
          onChange={(e) => onChange({ name: e.target.value })}
        />
        <button className="icon-btn" title="Remove project" onClick={onRemove}>
          ×
        </button>
      </div>

      <label className="block-field">
        <span>Root folder</span>
        <input
          placeholder="/absolute/path/to/repo"
          value={project.root}
          onChange={(e) => onChange({ root: e.target.value })}
        />
      </label>

      <label className="check-row">
        <input
          type="checkbox"
          checked={project.tasksEnabled !== false}
          onChange={(e) => onChange({ tasksEnabled: e.target.checked })}
        />
        Enable task tracker
      </label>

      <div className="sub-blocks">
        <div className="sub-head">Processes</div>
        {procs.map((proc, i) => (
          <ProcessBlock
            key={i}
            proc={proc}
            onChange={(patch) => setProc(i, patch)}
            onRemove={() => removeProc(i)}
          />
        ))}
        <button className="btn small" onClick={addProc}>
          + Add process
        </button>
      </div>

      <div className="sub-blocks">
        <div className="sub-head">Tasks</div>
        {tasks.map((t, i) => (
          <div className="task-row" key={i}>
            <input
              type="checkbox"
              checked={!!t.done}
              onChange={(e) => setTask(i, { done: e.target.checked })}
            />
            <input
              placeholder="Task title"
              value={t.title}
              onChange={(e) => setTask(i, { title: e.target.value })}
            />
            <button className="icon-btn" title="Remove task" onClick={() => removeTask(i)}>
              ×
            </button>
          </div>
        ))}
        <button className="btn small" onClick={addTask}>
          + Add task
        </button>
      </div>
    </div>
  );
}

function ProcessBlock({ proc, onChange, onRemove }) {
  const rows = proc._envRows || [];
  const setRow = (i, patch) =>
    onChange({ _envRows: rows.map((r, idx) => (idx === i ? { ...r, ...patch } : r)) });
  const addRow = () => onChange({ _envRows: [...rows, { k: "", v: "" }] });
  const removeRow = (i) => onChange({ _envRows: rows.filter((_, idx) => idx !== i) });

  return (
    <div className="block proc-sub-block">
      <div className="block-head">
        <span className="block-tag alt">Process</span>
        <input
          className="block-title-input"
          placeholder="Display name (e.g. frontend)"
          value={proc.name}
          onChange={(e) => onChange({ name: e.target.value })}
        />
        <button className="icon-btn" title="Remove process" onClick={onRemove}>
          ×
        </button>
      </div>

      <label className="block-field">
        <span>Working directory</span>
        <input
          placeholder="/absolute/path"
          value={proc.dir}
          onChange={(e) => onChange({ dir: e.target.value })}
        />
      </label>

      <label className="block-field">
        <span>Start command</span>
        <input
          placeholder="npm run dev"
          value={proc.command}
          onChange={(e) => onChange({ command: e.target.value })}
        />
      </label>

      <div className="env-block">
        <span className="env-label">Env vars</span>
        {rows.map((r, i) => (
          <div className="env-row" key={i}>
            <input
              placeholder="KEY"
              value={r.k}
              onChange={(e) => setRow(i, { k: e.target.value })}
            />
            <input
              placeholder="value"
              value={r.v}
              onChange={(e) => setRow(i, { v: e.target.value })}
            />
            <button className="icon-btn" title="Remove var" onClick={() => removeRow(i)}>
              ×
            </button>
          </div>
        ))}
        <button className="btn small" onClick={addRow}>
          + Add env var
        </button>
      </div>
    </div>
  );
}
