// Due-date helpers shared by the card, the detail modal, and the filters.
// Dates are stored as local calendar dates, "YYYY-MM-DD". Range presets are
// resolved by the Go backend (ResolveDueRange) so quarter dates have one
// source of truth shared with the MCP server and the AI.

const pad = (n) => String(n).padStart(2, "0");

export const toDateStr = (d) =>
  `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;

export const todayStr = () => toDateStr(new Date());

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;
export const isValidDate = (s) => {
  if (!DATE_RE.test(s || "")) return false;
  const [y, m, d] = s.split("-").map(Number);
  const dt = new Date(y, m - 1, d);
  return dt.getFullYear() === y && dt.getMonth() === m - 1 && dt.getDate() === d;
};

const isDone = (t) => t.done || t.status === "done";

// A task is past due when its due date is before today and it is not done.
// Due today is not past due.
export const isPastDue = (task, today = todayStr()) =>
  !!task.dueDate && isValidDate(task.dueDate) && !isDone(task) && task.dueDate < today;

const dayMs = 24 * 60 * 60 * 1000;

// Whole days from today to the due date (negative when past).
export const daysUntil = (due, today = todayStr()) => {
  const [y1, m1, d1] = due.split("-").map(Number);
  const [y2, m2, d2] = today.split("-").map(Number);
  return Math.round((Date.UTC(y1, m1 - 1, d1) - Date.UTC(y2, m2 - 1, d2)) / dayMs);
};

// Short card label: "Oct 9", with the year only when it is not this year.
export const formatDue = (due, today = todayStr()) => {
  if (!isValidDate(due)) return due || "";
  const [y, m, d] = due.split("-").map(Number);
  const opts = { month: "short", day: "numeric" };
  if (y !== Number(today.slice(0, 4))) opts.year = "numeric";
  return new Date(y, m - 1, d).toLocaleDateString(undefined, opts);
};

// Tooltip / detail text: "Past due by 3 days", "Due today", "Due in 5 days".
export const dueSummary = (task, today = todayStr()) => {
  if (!task.dueDate) return "";
  const n = daysUntil(task.dueDate, today);
  if (isDone(task)) return `Was due ${formatDue(task.dueDate, today)}`;
  if (n < 0) return `Past due by ${-n} day${-n === 1 ? "" : "s"}`;
  if (n === 0) return "Due today";
  return `Due in ${n} day${n === 1 ? "" : "s"}`;
};

// Named ranges offered in the filter, in display order. `preset` is the
// value ResolveDueRange understands.
export const DUE_PRESETS = [
  { id: "last_week", label: "Last week" },
  { id: "this_week", label: "This week" },
  { id: "next_week", label: "Next week" },
  { id: "this_month", label: "This month" },
  { id: "next_month", label: "Next month" },
  { id: "q1", label: "Q1" },
  { id: "q2", label: "Q2" },
  { id: "q3", label: "Q3" },
  { id: "q4", label: "Q4" },
  { id: "this_year", label: "This year" },
];

export const MONTHS = [
  "Jan", "Feb", "Mar", "Apr", "May", "Jun",
  "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
];

export const daysInMonth = (month) =>
  [31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31][month - 1] || 31;

// "MM-DD" <-> { month, day }
export const parseMD = (s) => {
  const m = /^(\d{2})-(\d{2})$/.exec(s || "");
  return m ? { month: Number(m[1]), day: Number(m[2]) } : { month: 1, day: 1 };
};
export const formatMD = ({ month, day }) => `${pad(month)}-${pad(day)}`;
export const describeMD = (s) => {
  const { month, day } = parseMD(s);
  return `${MONTHS[month - 1]} ${day}`;
};

export const DEFAULT_QUARTERS = [
  { name: "Q1", start: "01-01", end: "03-31" },
  { name: "Q2", start: "04-01", end: "06-30" },
  { name: "Q3", start: "07-01", end: "09-30" },
  { name: "Q4", start: "10-01", end: "12-31" },
];

// Filter state for the board. Empty arrays / strings mean "no constraint".
export const EMPTY_FILTERS = {
  duePreset: "",
  dueFrom: "",
  dueTo: "",
  noDueDate: false,
  overdue: false,
  statuses: [],
  priorities: [],
  types: [],
  sprints: [], // sprint ids; "" means the backlog
  assignees: [],
  labels: [],
  acceptance: "", // "" | "has" | "none"
  subtasks: "", // "" | "has" | "none"
};

export const activeFilterCount = (f) =>
  [
    f.duePreset || f.dueFrom || f.dueTo,
    f.noDueDate,
    f.overdue,
    f.statuses.length,
    f.priorities.length,
    f.types.length,
    f.sprints.length,
    f.assignees.length,
    f.labels.length,
    f.acceptance,
    f.subtasks,
  ].filter(Boolean).length;

const splitAssignees = (a) =>
  String(a || "")
    .split(",")
    .map((x) => x.trim().toLowerCase())
    .filter(Boolean);

// Does one task satisfy the filters? `range` is the resolved due-date range
// ({from, to}, either may be empty) or null when no range is selected.
export const matchesFilters = (task, f, range, today = todayStr()) => {
  if (f.statuses.length && !f.statuses.includes(task.status)) return false;
  if (f.priorities.length && !f.priorities.includes(task.priority || "none")) return false;
  if (f.types.length && !f.types.includes(task.type || "task")) return false;
  if (f.sprints.length && !f.sprints.includes(task.sprintId || "")) return false;
  if (f.assignees.length) {
    const have = splitAssignees(task.assignee);
    const want = f.assignees.map((a) => (a === "__none__" ? "__none__" : a.toLowerCase()));
    const hit = want.some((w) => (w === "__none__" ? have.length === 0 : have.includes(w)));
    if (!hit) return false;
  }
  if (f.labels.length) {
    const have = (task.labels || []).map((l) => l.toLowerCase());
    if (!f.labels.some((l) => have.includes(l.toLowerCase()))) return false;
  }
  if (f.acceptance === "has" && !(task.acceptance || []).length) return false;
  if (f.acceptance === "none" && (task.acceptance || []).length) return false;
  if (f.subtasks === "has" && !(task.subtasks || []).length) return false;
  if (f.subtasks === "none" && (task.subtasks || []).length) return false;
  if (f.noDueDate && task.dueDate) return false;
  if (f.overdue && !isPastDue(task, today)) return false;
  if (range) {
    if (!task.dueDate || !isValidDate(task.dueDate)) return false;
    if (range.from && task.dueDate < range.from) return false;
    if (range.to && task.dueDate > range.to) return false;
  }
  return true;
};
