// Kanban column definitions, task types, and migration helpers.

// `empty` shows when the column has no cards but the board has some.
// `emptyFirst` (Backlog only) shows when the whole board is empty, where a
// "drag things here" hint would be a dead end.
export const COLUMNS = [
  {
    id: "backlog",
    label: "Backlog",
    empty: "Drag items here to park them",
    emptyFirst: "Nothing here yet. Add a story, or ask AI to draft a few.",
  },
  { id: "todo", label: "To Do", empty: "Drag from Backlog when it's ready to start" },
  { id: "inprogress", label: "In Progress", empty: "Drag here when you pick it up" },
  { id: "testing", label: "Testing", empty: "Drag here when it needs QA or verification" },
  { id: "done", label: "Done", empty: "Finished work lands here" },
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
