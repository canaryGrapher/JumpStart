// Sprint definitions and helpers. Sprints are an ordered sequence
// (the roadmap); tasks join one via task.sprintId. An empty sprintId
// means the task sits in the project backlog.

import { uid } from "./columns";

export const BACKLOG_ID = "";

export const SPRINT_STATES = [
  { id: "planned", label: "Planned" },
  { id: "active", label: "Active" },
  { id: "completed", label: "Completed" },
];

export const blankSprint = (name, order = 0) => ({
  id: uid(),
  name: name || `Sprint ${order + 1}`,
  goal: "",
  status: "planned",
  startDate: "",
  endDate: "",
  order,
  createdAt: Date.now(),
});

// Older projects have no sprints; give each one the newer fields and
// keep the roadmap sequence consistent with array position.
export const migrateSprints = (sprints = []) =>
  [...sprints]
    .sort((a, b) => (a.order ?? 0) - (b.order ?? 0))
    .map((s, i) => ({
      goal: "",
      status: "planned",
      startDate: "",
      endDate: "",
      createdAt: Date.now(),
      ...s,
      order: i,
    }));

// Renumber after a reorder so `order` always matches position.
export const resequence = (sprints) =>
  sprints.map((s, i) => ({ ...s, order: i }));

// Move the sprint at `from` to index `to`, returning a new sequence.
export const moveSprint = (sprints, from, to) => {
  if (from === to || from < 0 || to < 0 || from >= sprints.length) return sprints;
  const next = [...sprints];
  const [item] = next.splice(from, 1);
  next.splice(Math.min(to, next.length), 0, item);
  return resequence(next);
};

export const tasksInSprint = (tasks, sprintId) =>
  tasks.filter((t) => (t.sprintId || "") === sprintId);

// Completion stats for one sprint, counting every task assigned to it.
export const sprintStats = (tasks, sprintId) => {
  const items = tasksInSprint(tasks, sprintId);
  const done = items.filter((t) => t.status === "done").length;
  const points = items.reduce((sum, t) => sum + (t.storyPoints || 0), 0);
  return {
    total: items.length,
    done,
    points,
    pct: items.length ? Math.round((done / items.length) * 100) : 0,
  };
};

// The sprint a board should open on: the active one, else the first
// incomplete one, else the backlog.
export const defaultSprintId = (sprints) => {
  if (!sprints.length) return BACKLOG_ID;
  const active = sprints.find((s) => s.status === "active");
  if (active) return active.id;
  const pending = sprints.find((s) => s.status !== "completed");
  return pending ? pending.id : BACKLOG_ID;
};

export const formatRange = (s) => {
  const fmt = (d) =>
    d ? new Date(`${d}T00:00:00`).toLocaleDateString(undefined, {
      month: "short",
      day: "numeric",
    }) : "";
  const a = fmt(s.startDate);
  const b = fmt(s.endDate);
  if (a && b) return `${a} – ${b}`;
  return a || b || "No dates set";
};
