import { useState } from "react";
import { PickDirectory, DetectProcesses, GenerateProjectDescription } from "../api";
import { getAISettings } from "../ai";
import ProcForm from "./ProcForm";
import Switch from "./Switch";

const uid = () => crypto.randomUUID();

const emptyProc = () => ({
  id: uid(),
  name: "",
  dir: "",
  command: "",
  env: {},
});

const isBlankProc = (p) =>
  !p.name.trim() && !p.dir.trim() && !p.command.trim();

// "my-cool_app" -> "My Cool App"
const titleFromDir = (dir) =>
  (dir.split("/").filter(Boolean).pop() || "")
    .replace(/[-_]+/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase());

export default function ProjectModal({ initial, onSave, onClose }) {
  const [project, setProject] = useState(
    initial || {
      id: uid(),
      name: "",
      root: "",
      processes: [],
      tasks: [],
      tasksEnabled: true,
    }
  );

  const set = (patch) => setProject((p) => ({ ...p, ...patch }));

  const setProc = (i, patch) =>
    setProject((p) => {
      const processes = [...p.processes];
      processes[i] = { ...processes[i], ...patch };
      return { ...p, processes };
    });

  const removeProc = (i) =>
    setProject((p) => ({
      ...p,
      processes: p.processes.filter((_, idx) => idx !== i),
    }));

  const pickDir = async (setter) => {
    const dir = await PickDirectory();
    if (dir) setter(dir);
  };

  // procId -> { dir, files } for pending env import prompts
  const [envPrompts, setEnvPrompts] = useState({});
  const [detecting, setDetecting] = useState(false);
  const [detectMsg, setDetectMsg] = useState(null); // { text, kind }
  const [generatingDesc, setGeneratingDesc] = useState(false);
  const [descMsg, setDescMsg] = useState(null); // { text, kind }

  const generateDescription = async () => {
    if (!project.root.trim()) {
      setDescMsg({ text: "Choose a project folder first.", kind: "err" });
      return;
    }
    setGeneratingDesc(true);
    setDescMsg(null);
    try {
      const { host, model } = getAISettings();
      const text = await GenerateProjectDescription(host, model, project.root, project.name);
      if (text) set({ description: text });
      setDescMsg(text ? null : { text: "The model returned nothing.", kind: "err" });
    } catch (e) {
      setDescMsg({ text: String(e), kind: "err" });
    } finally {
      setGeneratingDesc(false);
    }
  };

  const autoDetect = async (root = project.root) => {
    if (!root.trim()) {
      setDetectMsg({ text: "Choose a project folder first.", kind: "err" });
      return;
    }
    setDetecting(true);
    setDetectMsg(null);
    try {
      const found = await DetectProcesses(root);
      if (!found || found.length === 0) {
        setDetectMsg({ text: "No runnable subprocesses found.", kind: "" });
        return;
      }
      const prompts = {};
      const processes = found.map((d) => {
        const id = uid();
        if (d.envFiles && d.envFiles.length > 0) {
          prompts[id] = { dir: d.dir, files: d.envFiles };
        }
        return { id, name: d.name, dir: d.dir, command: d.command, env: {} };
      });
      setProject((p) => ({
        ...p,
        processes: [...p.processes.filter((x) => !isBlankProc(x)), ...processes],
      }));
      setEnvPrompts(prompts);
      setDetectMsg({
        text: `Found ${found.length} subprocess${found.length > 1 ? "es" : ""}.`,
        kind: "ok",
      });
    } catch (e) {
      setDetectMsg({ text: String(e), kind: "err" });
    } finally {
      setDetecting(false);
    }
  };

  // Choosing the root folder fills in the name (if empty) and, on a fresh
  // project, kicks off detection right away.
  const pickRoot = async () => {
    const dir = await PickDirectory();
    if (!dir) return;
    const patch = { root: dir };
    if (!project.name.trim()) patch.name = titleFromDir(dir);
    set(patch);
    if (project.processes.every(isBlankProc)) autoDetect(dir);
  };

  const clearPrompt = (procId) =>
    setEnvPrompts(({ [procId]: _, ...rest }) => rest);

  const importEnv = (i, procId, env) => {
    setProc(i, { env: { ...(project.processes[i].env || {}), ...env } });
    clearPrompt(procId);
  };

  const valid =
    project.name.trim() &&
    project.processes.every(
      (p) => p.name.trim() && p.dir.trim() && p.command.trim()
    );

  const save = () => valid && onSave(project);

  // macOS sheet keys: Esc cancels, Return triggers the default button.
  const onKeyDown = (e) => {
    if (e.key === "Escape") onClose();
    else if (e.key === "Enter" && e.target.tagName === "INPUT") save();
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()} onKeyDown={onKeyDown}>
        <header className="sheet-head">
          <h2>{initial ? "Edit Project" : "New Project"}</h2>
          <p>
            {initial
              ? "Change the project's details and subprocesses."
              : "Choose a folder and JumpStart will find what to run."}
          </p>
        </header>

        <div className="form-group">
          <div className="form-row">
            <label htmlFor="pj-name">Name</label>
            <div className="control">
              <input
                id="pj-name"
                autoFocus
                value={project.name}
                placeholder="Project Alpha"
                onChange={(e) => set({ name: e.target.value })}
              />
            </div>
          </div>

          <div className="form-row">
            <label htmlFor="pj-root">Folder</label>
            <div className="control">
              <input
                id="pj-root"
                value={project.root}
                placeholder="/Users/you/code/project-alpha"
                onChange={(e) => set({ root: e.target.value })}
              />
              <button className="btn" onClick={pickRoot}>
                Choose…
              </button>
            </div>
          </div>

          <div className="form-row">
            <label>Task Board</label>
            <div className="control">
              <span className="row-hint">Show a task tracker tab for this project</span>
              <Switch
                checked={!!project.tasksEnabled}
                onChange={(v) => set({ tasksEnabled: v })}
              />
            </div>
          </div>
        </div>

        <div className="sheet-section">
          <h3>Description</h3>
          <div className="actions">
            <button
              className="btn small ai"
              disabled={generatingDesc || !project.root.trim()}
              onClick={generateDescription}
            >
              {generatingDesc ? "Generating…" : "✨ Generate with AI"}
            </button>
          </div>
        </div>
        {descMsg && <p className={`detect-note ${descMsg.kind}`}>{descMsg.text}</p>}
        <div className="field">
          <textarea
            rows={4}
            placeholder="What does this project do?"
            value={project.description || ""}
            onChange={(e) => set({ description: e.target.value })}
          />
        </div>

        <div className="sheet-section">
          <h3>Subprocesses</h3>
          <div className="actions">
            <button
              className="btn small"
              disabled={detecting || !project.root.trim()}
              onClick={() => autoDetect()}
            >
              {detecting ? "Scanning…" : "Auto-Detect"}
            </button>
            <button
              className="btn small"
              onClick={() => set({ processes: [...project.processes, emptyProc()] })}
            >
              Add
            </button>
          </div>
        </div>

        {detectMsg && (
          <p className={`detect-note ${detectMsg.kind}`}>{detectMsg.text}</p>
        )}

        {project.processes.length === 0 && (
          <div className="sheet-empty">
            No subprocesses yet. Use Auto-Detect to scan the project folder,
            or Add to set one up manually.
          </div>
        )}

        {project.processes.map((proc, i) => (
          <ProcForm
            key={proc.id}
            proc={proc}
            index={i}
            rootHint={project.root}
            prompt={envPrompts[proc.id]}
            onChange={(patch) => setProc(i, patch)}
            onRemove={() => removeProc(i)}
            onPickDir={() => pickDir((d) => setProc(i, { dir: d }))}
            onImportEnv={(env) => importEnv(i, proc.id, env)}
            onDismissPrompt={() => clearPrompt(proc.id)}
          />
        ))}

        <div className="modal-actions">
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn primary" disabled={!valid} onClick={save}>
            {initial ? "Save" : "Add Project"}
          </button>
        </div>
      </div>
    </div>
  );
}
