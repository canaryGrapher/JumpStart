// Shared mock data mirroring the live JumpStart dashboard screenshot.
export const FAVORITES = [
  { n: "AnyLM", s: "2 subprocesses", c: "#3b82f6", g: "💬" },
  { n: "College Connect", s: "3 subprocesses", c: "#22c55e", g: "C" },
  { n: "CronCompose", s: "4 subprocesses", c: "#0ea5e9", g: "≋" },
  { n: "DualNature Pro", s: "2 subprocesses", c: "#8b5cf6", g: "◈" },
  { n: "FocusLens", s: "2 subprocesses", c: "#a855f7", g: "◎" },
  { n: "JumpStart", s: "2 subprocesses", c: "#14b8a6", g: "🚀", sel: true },
];

export const ALL_PROJECTS = [
  { n: "AttendEase", s: "2 subprocesses", c: "#f59e0b", g: "A" },
  { n: "Carousel Studio", s: "1 subprocess", c: "#ec4899", g: "◈" },
  { n: "Consent Manager", s: "5 subprocesses", c: "#6366f1", g: "C" },
  { n: "Defiance Smart Chain", s: "3 subprocesses", c: "#84cc16", g: "D" },
  { n: "PeepalRP", s: "2 subprocesses", c: "#f97316", g: "P" },
  { n: "Postggresively", s: "3 subprocesses", c: "#336791", g: "🐘" },
];

export const RECENT = [
  {
    n: "JumpStart",
    s: "2 subprocesses",
    d: "JumpStart is a macOS control panel that helps manage…",
    m: "12m ago",
    c: "#14b8a6",
    g: "🚀",
  },
  {
    n: "FocusLens",
    s: "2 subprocesses",
    d: "FocusLens tracks deep work sessions and streaks…",
    m: "14m ago",
    c: "#a855f7",
    g: "◎",
  },
  {
    n: "College Connect",
    s: "3 subprocesses",
    d: "Campus social graph for students and clubs…",
    m: "1h ago",
    c: "#22c55e",
    g: "C",
  },
  {
    n: "AnyLM",
    s: "2 subprocesses",
    d: "AnyLM is a local-first workspace for large language…",
    m: "3h ago",
    c: "#3b82f6",
    g: "💬",
  },
  {
    n: "Postggresively",
    s: "3 subprocesses",
    d: "Postgres tooling for migrations and query plans…",
    m: "1d ago",
    c: "#336791",
    g: "🐘",
  },
];

export const MOST_USED = [
  {
    n: "Postggresively",
    s: "3 subprocesses",
    d: "Postgres tooling for migrations and query plans…",
    m: "57 starts",
    c: "#336791",
    g: "🐘",
  },
  {
    n: "College Connect",
    s: "3 subprocesses",
    d: "Campus social graph for students and clubs…",
    m: "56 starts",
    c: "#22c55e",
    g: "C",
  },
  {
    n: "FocusLens",
    s: "2 subprocesses",
    d: "FocusLens tracks deep work sessions and streaks…",
    m: "31 starts",
    c: "#a855f7",
    g: "◎",
  },
  {
    n: "CronCompose",
    s: "4 subprocesses",
    d: "CronCompose schedules and visualises cron jobs…",
    m: "22 starts",
    c: "#0ea5e9",
    g: "≋",
  },
  {
    n: "JumpStart",
    s: "2 subprocesses",
    d: "JumpStart is a macOS control panel that helps manage…",
    m: "18 starts",
    c: "#14b8a6",
    g: "🚀",
  },
];

export const PORTS = [
  { port: ":5173", project: "JumpStart", sub: "landing (Vite)", pid: "61011" },
  { port: ":5174", project: "JumpStart", sub: "frontend (Vite)", pid: "61042" },
  { port: ":34115", project: "FocusLens", sub: "api (Go)", pid: "58201" },
  { port: ":5432", project: "Postggresively", sub: "postgres", pid: "4412" },
];

export const STATS = [
  {
    id: "stat1",
    label: "Projects",
    value: "24",
    pill: "1 running",
    pillKind: "good",
    sub: "subprocesses currently managed",
  },
  {
    id: "stat2",
    label: "System CPU",
    value: "4%",
    pill: "12 cores",
    pillKind: "neutral",
    meter: 4,
  },
  {
    id: "stat3",
    label: "System memory",
    value: "74%",
    pill: "18/24 GB",
    pillKind: "neutral",
    meter: 74,
  },
  {
    id: "stat4",
    label: "Features built",
    value: "225/461",
    pill: "49%",
    pillKind: "good",
    meter: 49,
  },
];
