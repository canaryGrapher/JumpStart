// Builds an isolated HOME for the e2e run: a ~/.jumpstart with one synthetic
// project, MCP enabled on a test port, and a project root holding sample
// files. Nothing here touches the real ~/.jumpstart.
//
//   node scripts/seed.mjs <homeDir> [--legacy]
//
// --legacy writes tasks in the pre-upgrade shape (no due date, links or
// attachments) to prove old data loads unchanged.
import fs from "node:fs";
import path from "node:path";

export const MCP_PORT = 18788;
export const MCP_TOKEN = "e2e-test-token-0123456789abcdef";
export const PROJECT_ID = "e2e-proj";

const PNG_1x1 =
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==";

const pad = (n) => String(n).padStart(2, "0");
export const fmt = (d) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
const addDays = (d, n) => new Date(d.getFullYear(), d.getMonth(), d.getDate() + n);

export function fixtureDates(now = new Date()) {
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const monday = addDays(today, -((today.getDay() + 6) % 7));
  const y = today.getFullYear();
  return {
    today: fmt(today),
    overdue: fmt(addDays(today, -3)),
    longAgo: fmt(addDays(today, -10)),
    lastWeek: fmt(addDays(monday, -7 + 2)),
    nextWeek: fmt(addDays(monday, 7 + 2)),
    monthEnd: fmt(new Date(y, today.getMonth() + 1, 0)),
    nextMonth: fmt(new Date(y, today.getMonth() + 1, 15)),
    q1: `${y}-02-10`,
    q2: `${y}-05-10`,
    q3: `${y}-08-10`,
    q4: `${y}-11-10`,
    lastYear: `${y - 1}-12-31`,
    yearEnd: `${y}-12-31`,
  };
}

export function seed(home, { legacy = false } = {}) {
  const d = fixtureDates();
  const dir = path.join(home, ".jumpstart");
  const root = path.join(home, "project-root");
  // Reset everything a previous test run may have left behind.
  fs.rmSync(path.join(dir, "attachments"), { recursive: true, force: true });
  fs.rmSync(path.join(dir, "calendar.json"), { force: true });
  fs.rmSync(root, { recursive: true, force: true }); // files/links/folders tests create here
  fs.rmSync(path.join(home, "outside-secret.txt"), { force: true });
  fs.mkdirSync(dir, { recursive: true });
  fs.mkdirSync(root, { recursive: true });
  fs.writeFileSync(path.join(root, "notes.md"), "Release is planned for Friday. Contact ops@example.com.\n");
  fs.writeFileSync(path.join(root, "shot.png"), Buffer.from(PNG_1x1, "base64"));
  fs.writeFileSync(path.join(root, "doc.pdf"), "%PDF-1.4\n%fake\n");

  const t = (id, title, extra = {}) => ({
    id,
    title,
    type: "task",
    status: "todo",
    done: extra.status === "done",
    sprintId: "s1",
    createdAt: 1790000000000,
    ...extra,
  });
  const tasks = [
    t("t-overdue", "Overdue bug", {
      type: "bug", priority: "high", assignee: "Alex, Sam", labels: ["stripe", "urgent"],
      dueDate: d.overdue, acceptance: [{ id: "a1", title: "Retries succeed", done: false }],
    }),
    t("t-today", "Due today task", {
      status: "inprogress", priority: "medium", assignee: "sam", dueDate: d.today,
      subtasks: [{ id: "s-1", title: "Write it", done: true }, { id: "s-2", title: "Ship it", done: false }],
    }),
    t("t-done-late", "Done but late", { status: "done", priority: "low", dueDate: d.longAgo, labels: ["stripe"] }),
    t("t-lastweek", "Last week item", { dueDate: d.lastWeek, assignee: "Ada" }),
    t("t-nextweek", "Next week item", { dueDate: d.nextWeek }),
    t("t-monthend", "Month end item", { dueDate: d.monthEnd }),
    t("t-nextmonth", "Next month item", { dueDate: d.nextMonth }),
    t("t-q1", "Q1 item", { dueDate: d.q1 }),
    t("t-q2", "Q2 item", { dueDate: d.q2 }),
    t("t-q3", "Q3 item", { dueDate: d.q3 }),
    t("t-q4", "Q4 item", { dueDate: d.q4 }),
    t("t-lastyear", "Last year item", { dueDate: d.lastYear }),
    t("t-nodate", "No date item", { status: "backlog", sprintId: "" }),
    t("story-1", "Checkout revamp", { type: "story", status: "inprogress", priority: "high" }),
    t("child-1", "Add payment form", { parentId: "story-1", dueDate: d.overdue }),
    t("child-2", "Write form tests", { parentId: "story-1" }),
  ];
  const outTasks = legacy
    ? tasks.map(({ dueDate, links, attachments, ...rest }) => rest)
    : tasks;

  const project = {
    id: PROJECT_ID,
    name: "E2E Board",
    root,
    processes: [],
    tasks: outTasks,
    sprints: [
      { id: "s1", name: "Sprint 1", status: "active", order: 0, createdAt: 1790000000000 },
      { id: "s2", name: "Sprint 2", status: "planned", order: 1, createdAt: 1790000000000 },
    ],
    tasksEnabled: true,
  };
  fs.writeFileSync(path.join(dir, "config.json"), JSON.stringify([project], null, 2));
  fs.writeFileSync(
    path.join(dir, "mcp.json"),
    JSON.stringify({ enabled: true, port: MCP_PORT, token: MCP_TOKEN }, null, 2),
    { mode: 0o600 }
  );
  fs.writeFileSync(path.join(home, "fixture.json"), JSON.stringify({ dates: d, root, projectId: PROJECT_ID }, null, 2));
  return { dir, root, dates: d };
}

if (process.argv[1] && process.argv[1].endsWith("seed.mjs")) {
  const home = process.argv[2];
  if (!home) throw new Error("usage: seed.mjs <homeDir> [--legacy]");
  seed(home, { legacy: process.argv.includes("--legacy") });
  console.log(`seeded ${home}`);
}
