import { useState } from "react";
import { PickDirectory, DetectProcesses } from "../api";
import EnvEditor from "./EnvEditor";
import EnvImportPrompt from "./EnvImportPrompt";

const uid = () => crypto.randomUUID();

const emptyProc = () => ({
  id: uid(),
  name: "",
  dir: "",
  command: "",
  env: {},
});

export default function ProjectModal({ initial, onSave, onClose }) {
  const [project, setProject] = useState(
    initial || {
      id: uid(),
      name: "",
      root: "",
      processes: [emptyProc()],
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
  const [detectMsg, setDetectMsg] = useState("");

  const autoDetect = async () => {
    if (!project.root.trim()) {
      setDetectMsg("Set a root folder first.");
      return;
    }
    setDetecting(true);
    setDetectMsg("");
    try {
      const found = await DetectProcesses(project.root);
      if (!found || found.length === 0) {
        setDetectMsg("No runnable subprocesses found.");
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
      const keep = project.processes.filter(
        (p) => p.name.trim() || p.dir.trim() || p.command.trim()
      );
      set({ processes: [...keep, ...processes] });
      setEnvPrompts(prompts);
      setDetectMsg(`Detected ${found.length} subprocess${found.length > 1 ? "es" : ""}.`);
    } catch (e) {
      setDetectMsg(String(e));
    } finally {
      setDetecting(false);
    }
  };

  const clearPrompt = (procId) =>
    setEnvPrompts(({ [procId]: _, ...rest }) => rest);

  const importEnv = (i, procId, env) => {
    setProc(i, { env: { ...(project.processes[i].env || {}), ...env } });
    clearPrompt(procId);
  };

  const valid =
    project.name.trim() &&
    project.processes.every((p) => p.name.trim() && p.dir.trim() && p.command.trim());

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2>{initial ? "Edit project" : "New project"}</h2>

        <div className="field">
          <label>Project name</label>
          <input
            value={project.name}
            placeholder="Project Alpha"
            onChange={(e) => set({ name: e.target.value })}
          />
        </div>

        <div className="field">
          <label>Root folder</label>
          <div className="row">
            <input
              value={project.root}
              placeholder="/Users/you/code/project-alpha"
              onChange={(e) => set({ root: e.target.value })}
            />
            <button className="btn" onClick={() => pickDir((d) => set({ root: d }))}>
              Browse
            </button>
            <button className="btn" disabled={detecting} onClick={autoDetect}>
              {detecting ? "Detecting..." : "Auto-detect"}
            </button>
          </div>
          {detectMsg && <span className="hint">{detectMsg}</span>}
        </div>

        <label className="check-row">
          <input
            type="checkbox"
            checked={!!project.tasksEnabled}
            onChange={(e) => set({ tasksEnabled: e.target.checked })}
          />
          Enable project management (task tracker tab)
        </label>

        <label style={{ fontSize: 12, fontWeight: 600, color: "var(--cf-text-dim)" }}>
          Subprocesses
        </label>
        {project.processes.map((proc, i) => (
          <div className="proc-block" key={proc.id}>
            <div className="proc-block-top">
              <strong>Subprocess {i + 1}</strong>
              <button className="link-btn" onClick={() => removeProc(i)}>
                Remove
              </button>
            </div>
            <div className="field">
              <label>Name</label>
              <input
                value={proc.name}
                placeholder="frontend (Next.js)"
                onChange={(e) => setProc(i, { name: e.target.value })}
              />
            </div>
            <div className="field">
              <label>Working directory</label>
              <div className="row">
                <input
                  value={proc.dir}
                  placeholder={project.root ? project.root + "/frontend" : "/path/to/frontend"}
                  onChange={(e) => setProc(i, { dir: e.target.value })}
                />
                <button
                  className="btn"
                  onClick={() => pickDir((d) => setProc(i, { dir: d }))}
                >
                  Browse
                </button>
              </div>
            </div>
            <div className="field">
              <label>Start command</label>
              <input
                value={proc.command}
                placeholder="npm run dev"
                onChange={(e) => setProc(i, { command: e.target.value })}
              />
            </div>
            <div className="field">
              <label>Environment variables</label>
              {envPrompts[proc.id] && (
                <EnvImportPrompt
                  dir={envPrompts[proc.id].dir}
                  files={envPrompts[proc.id].files}
                  onImport={(env) => importEnv(i, proc.id, env)}
                  onDismiss={() => clearPrompt(proc.id)}
                />
              )}
              <EnvEditor
                env={proc.env || {}}
                onChange={(env) => setProc(i, { env })}
              />
            </div>
          </div>
        ))}

        <button
          className="link-btn"
          onClick={() => set({ processes: [...project.processes, emptyProc()] })}
        >
          + Add subprocess
        </button>

        <div className="modal-actions">
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button
            className="btn primary"
            disabled={!valid}
            onClick={() => onSave(project)}
          >
            Save
          </button>
        </div>
      </div>
    </div>
  );
}
