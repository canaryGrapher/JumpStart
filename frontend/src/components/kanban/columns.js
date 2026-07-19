// Kanban column definitions, task types, and migration helpers.

export const COLUMNS = [
  { id: "backlog", label: "Backlog" },
  { id: "todo", label: "To Do" },
  { id: "inprogress", label: "In Progress" },
  { id: "done", label: "Done" },
];

export const TYPES = [
  { id: "story", label: "Story" },
  { id: "task", label: "Task" },
  { id: "bug", label: "Bug" },
];

export const uid = () =>
  (crypto.randomUUID && crypto.randomUUID()) ||
  `${Date.now()}-${Math.random().toString(16).slice(2)}`;

// Older tasks only have a `done` flag; give them the newer fields.
export const migrate = (t) => ({
  ...t,
  status: t.status || (t.done ? "done" : "todo"),
  type: t.type || "task",
  parentId: t.parentId || "",
  sprintId: t.sprintId || "",
  subtasks: t.subtasks || [],
  acceptance: t.acceptance || [],
  labels: t.labels || [],
});

export const withStatus = (t, status) => ({
  ...t,
  status,
  done: status === "done",
  updatedAt: Date.now(),
});

// A fresh empty item of the given type/status.
export const blankTask = (
  title,
  { type = "task", status = "todo", parentId = "", sprintId = "" } = {}
) => ({
  id: uid(),
  title,
  type,
  parentId,
  sprintId,
  status,
  done: status === "done",
  description: "",
  priority: "",
  labels: [],
  subtasks: [],
  acceptance: [],
  storyPoints: 0,
  assignee: "",
  createdAt: Date.now(),
});
