// Shared helpers for the import block builder: factories and
// conversion between the builder's shape and the conf JSON.

export const emptyProc = () => ({
  name: "",
  dir: "",
  command: "",
  _envRows: [], // [{ k, v }] — flattened to an env object on serialize
});

export const emptyProject = () => ({
  name: "",
  root: "",
  tasksEnabled: true,
  processes: [],
  tasks: [],
});

// toBuilder decorates parsed projects with editable env rows.
const toBuilder = (projects) =>
  projects.map((p) => ({
    name: p.name || "",
    root: p.root || "",
    tasksEnabled: p.tasksEnabled !== false,
    processes: (p.processes || []).map((proc) => ({
      name: proc.name || "",
      dir: proc.dir || "",
      command: proc.command || "",
      _envRows: Object.entries(proc.env || {}).map(([k, v]) => ({ k, v: String(v) })),
    })),
    tasks: (p.tasks || []).map((t) => ({ title: t.title || "", done: !!t.done })),
  }));

// fromBuilder strips builder-only fields and rebuilds env objects.
const fromBuilder = (projects) =>
  projects.map((p) => {
    const proc = (p.processes || []).map((x) => {
      const env = {};
      for (const { k, v } of x._envRows || []) {
        if (k.trim()) env[k.trim()] = v;
      }
      const out = { name: x.name, dir: x.dir, command: x.command };
      if (Object.keys(env).length) out.env = env;
      return out;
    });
    const out = { name: p.name, root: p.root, tasksEnabled: p.tasksEnabled };
    if (proc.length) out.processes = proc;
    if ((p.tasks || []).length) out.tasks = p.tasks.map((t) => ({ title: t.title, done: !!t.done }));
    return out;
  });

// serialize turns builder projects into pretty conf JSON text.
export const serialize = (projects) =>
  JSON.stringify({ projects: fromBuilder(projects) }, null, 2);

// parse reads conf text (bare array or { projects: [] }) into builder
// projects. Returns { projects, error }.
export const parse = (text) => {
  if (!text.trim()) return { projects: [], error: null };
  let data;
  try {
    data = JSON.parse(text);
  } catch (e) {
    return { projects: null, error: "Invalid JSON: " + e.message };
  }
  const arr = Array.isArray(data) ? data : data.projects;
  if (!Array.isArray(arr)) {
    return { projects: null, error: 'Expected an array or a { "projects": [...] } object.' };
  }
  return { projects: toBuilder(arr), error: null };
};
