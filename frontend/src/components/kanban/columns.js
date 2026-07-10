// Kanban column definitions and task migration helpers.

export const COLUMNS = [
  { id: "todo", label: "To Do" },
  { id: "inprogress", label: "In Progress" },
  { id: "done", label: "Done" },
];

// Older tasks only have a `done` flag; give them a kanban status.
export const migrate = (t) => ({
  ...t,
  status: t.status || (t.done ? "done" : "todo"),
  subtasks: t.subtasks || [],
  labels: t.labels || [],
});

export const withStatus = (t, status) => ({
  ...t,
  status,
  done: status === "done",
  updatedAt: Date.now(),
});
