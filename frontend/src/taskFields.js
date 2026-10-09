// Canonical task field list shared by CSV import/export, the Import-config
// "Copy prompt", and the AI enrich/chat prompts. Keep in sync with
// internal/taskcsv.Header and model.Task.

export const CSV_HEADER = [
  "id",
  "title",
  "type",
  "status",
  "priority",
  "description",
  "assignee",
  "labels",
  "storyPoints",
  "parentId",
  "sprintId",
  "sprint",
  "done",
  "subtasks",
  "acceptance",
  "createdAt",
  "updatedAt",
  "milestone",
  "issueType",
  "parentKey",
  "reviewers",
  "linkedPrs",
  "dueDate",
];

// Human-readable field notes for prompts / import guides.
export const TASK_FIELD_DOCS = [
  { key: "id", note: "optional; omit on create — JumpStart assigns one" },
  { key: "title", note: "required" },
  { key: "type", note: "story | task | bug (default task)" },
  { key: "status", note: "backlog | todo | inprogress | testing | done" },
  { key: "priority", note: "low | medium | high" },
  { key: "description", note: "prose; for stories prefer As a / I want / so that" },
  { key: "assignee", note: "GitHub login(s), comma-separated" },
  { key: "labels", note: "CSV: comma-separated in one cell; JSON: string array" },
  { key: "storyPoints", note: "integer 1–13" },
  { key: "parentId", note: "id of parent story (tasks/bugs under a story)" },
  { key: "sprintId", note: "sprint id; prefer sprint name when seeding" },
  { key: "sprint", note: "sprint display name; unknown names create the sprint" },
  { key: "done", note: "boolean; keep in sync with status === done" },
  {
    key: "subtasks",
    note: 'implementation checklist — CSV: Title|Next (prefix [x] when done); JSON: [{title, done}]',
  },
  {
    key: "acceptance",
    note: "testable acceptance criteria (any type) — same shapes as subtasks",
  },
  { key: "createdAt", note: "unix ms; optional on import" },
  { key: "updatedAt", note: "unix ms; optional on import" },
  { key: "milestone", note: "GitHub milestone title (optional)" },
  { key: "issueType", note: "GitHub issue type (optional)" },
  { key: "parentKey", note: "GitHub parent issue ref (optional)" },
  { key: "reviewers", note: "CSV: comma-separated; JSON: string array" },
  { key: "linkedPrs", note: "CSV: comma-separated URLs; JSON: string array" },
  { key: "dueDate", note: "YYYY-MM-DD; leave empty for no due date" },
];

// Escape one CSV cell.
const csvCell = (value) => {
  const s = value == null ? "" : String(value);
  if (/[",\n\r]/.test(s)) return `"${s.replace(/"/g, '""')}"`;
  return s;
};

// Build a CSV document from row objects keyed by CSV_HEADER.
export const rowsToCSV = (rows) => {
  const lines = [CSV_HEADER.join(",")];
  for (const row of rows) {
    lines.push(CSV_HEADER.map((h) => csvCell(row[h])).join(","));
  }
  return lines.join("\n");
};

export const SAMPLE_TASK_ROWS = [
  {
    title: "Set up CI pipeline",
    type: "task",
    status: "todo",
    priority: "high",
    description: "Wire GitHub Actions for build + test",
    labels: "ci,infra",
    storyPoints: 3,
    sprint: "Sprint 1",
    done: false,
    subtasks: "Install runners|Add workflow",
    acceptance: "Must pass on main",
  },
  {
    title: "Fix login redirect",
    type: "bug",
    status: "inprogress",
    priority: "medium",
    description: "Session cookie drops on refresh",
    assignee: "alice",
    labels: "bug",
    storyPoints: 2,
    sprint: "Sprint 1",
    done: false,
    acceptance: "Redirects to dashboard after login",
  },
  {
    title: "User onboarding story",
    type: "story",
    status: "backlog",
    priority: "low",
    description: "As a new user, I want a guided first run, so that I create a project in under 10 minutes.",
    labels: "ux",
    storyPoints: 5,
    done: false,
    subtasks: "Welcome modal|Sample project",
    acceptance: "User completes first project in <10m|Onboarding can be skipped",
  },
  {
    // Empty template row so users see every column.
    title: "",
    type: "task",
    status: "todo",
    done: false,
  },
];

export const buildSampleCSV = () => rowsToCSV(SAMPLE_TASK_ROWS);

// Compact schema block embedded in the Import-config copy prompt.
export const taskSchemaForPrompt = () => {
  const lines = TASK_FIELD_DOCS.map((f) => `      "${f.key}": … // ${f.note}`);
  return `{
${lines.join("\n")}
    }`;
};
