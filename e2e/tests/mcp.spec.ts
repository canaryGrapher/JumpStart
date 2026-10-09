import { test, expect } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";
import {
  reseed, Mcp, MCP_PORT, MCP_TOKEN, PROJECT_ID, DATA_DIR, readTask, readProject,
  expectedRange, inRange, openBoard, openTask, card, type Dates,
} from "./helpers";

const P = PROJECT_ID;
let dates: Dates;
let root: string;
let mcp: Mcp;

test.beforeEach(async () => {
  const s = reseed();
  dates = s.dates;
  root = s.root;
  fs.rmSync(path.join(DATA_DIR, "attachments"), { recursive: true, force: true });
  mcp = await Mcp.connect();
});

test.describe("connection and tool surface", () => {
  test("requests without the right bearer token are rejected", async () => {
    for (const auth of [undefined, "Bearer wrong-token", "Bearer "]) {
      const res = await fetch(`http://127.0.0.1:${MCP_PORT}/mcp`, {
        method: "POST",
        headers: { "content-type": "application/json", accept: "application/json, text/event-stream", ...(auth ? { authorization: auth } : {}) },
        body: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} }),
      });
      expect(res.status, String(auth)).toBe(401);
    }
  });

  test("lists the new tools with the new parameters", async () => {
    const tools = await mcp.tools();
    const byName = Object.fromEntries(tools.map((t) => [t.name, t]));
    for (const name of [
      "list_tasks", "get_task", "upsert_task", "delete_task", "append_task_note",
      "add_task_attachment", "read_task_attachment", "remove_task_attachment", "get_quarters", "set_quarters",
    ]) expect(byName[name], name).toBeTruthy();

    const upsert = Object.keys(byName.upsert_task.inputSchema.properties);
    for (const f of ["dueDate", "milestone", "subtasks", "acceptance", "links", "clear"]) expect(upsert).toContain(f);
    const list = Object.keys(byName.list_tasks.inputSchema.properties);
    for (const f of ["status", "priority", "type", "sprintId", "assignee", "label", "hasAcceptance", "hasSubtasks",
      "hasLinks", "hasAttachments", "duePreset", "dueFrom", "dueTo", "overdue", "noDueDate"]) expect(list).toContain(f);
    expect(byName.list_tasks.description).toContain("this_week");
    expect(byName.list_tasks.description).toContain("q1");
  });
});

test.describe("writing every field", () => {
  test("create a task with all fields; read it back over MCP, on disk, and in the UI", async ({ page }) => {
    const created = await mcp.json("upsert_task", {
      projectId: P, title: "MCP full task", type: "story", status: "testing", priority: "high",
      assignee: "Alex", labels: ["mcp", "e2e"], sprintId: "s1", storyPoints: 5, dueDate: dates.nextWeek,
      milestone: "v2", description: "Created over MCP",
      subtasks: [{ title: "Draft" }, { title: "Review", done: true }],
      acceptance: [{ title: "Passes e2e" }],
      links: [{ title: "Spec", url: "example.com/spec" }, { url: "mailto:team@example.com" }],
    });
    expect(created).toMatchObject({
      title: "MCP full task", status: "testing", done: false, priority: "high", dueDate: dates.nextWeek,
      milestone: "v2", storyPoints: 5,
    });
    expect(created.subtasks.map((s: any) => [s.title, s.done])).toEqual([["Draft", false], ["Review", true]]);
    expect(created.subtasks.every((s: any) => s.id)).toBe(true);
    expect(created.links.map((l: any) => l.url)).toEqual(["https://example.com/spec", "mailto:team@example.com"]);

    const got = await mcp.json("get_task", { projectId: P, taskId: created.id });
    expect(got).toEqual(created);
    expect(readTask(created.id)).toMatchObject({ dueDate: dates.nextWeek, milestone: "v2" });

    await openBoard(page);
    const c = card(page, "MCP full task");
    await expect(c.locator(".kb-pill.due")).toBeVisible();
    await expect(c.locator(".kb-pill.links")).toHaveText("🔗 2");
    await expect(c.locator(".kb-pill.subs")).toHaveText("☑ 1/2");
    await openTask(page, "MCP full task");
    await expect(page.locator(".kb-due-field input[type=date]")).toHaveValue(dates.nextWeek);
    await expect(page.locator(".kb-link").first()).toContainText("Spec");
    // Checklist items render as inputs, so compare input values.
    const values = (sel: string) => page.locator(sel).evaluateAll((els) => els.map((e) => (e as HTMLInputElement).value));
    expect(await values(".task-row input.task-title")).toEqual(expect.arrayContaining(["Passes e2e", "Draft", "Review"]));
    expect(await values(".task-row.done input.task-title")).toEqual(["Review"]); // only the checked subtask is struck
  });

  test("status keeps the done flag in step", async () => {
    const t = await mcp.json("upsert_task", { projectId: P, title: "Flag", status: "done" });
    expect(t.done).toBe(true);
    const re = await mcp.json("upsert_task", { projectId: P, taskId: t.id, status: "todo" });
    expect(re).toMatchObject({ status: "todo", done: false });
    expect(readTask(t.id).done ?? false).toBe(false);
  });

  test("updates preserve checklist ids and checked state; omitted fields are untouched", async () => {
    const t = await mcp.json("upsert_task", {
      projectId: P, title: "Keep ids", dueDate: dates.today, labels: ["a"], priority: "low",
      subtasks: [{ title: "One", done: true }, { title: "Two" }],
    });
    const again = await mcp.json("upsert_task", {
      projectId: P, taskId: t.id, subtasks: [{ title: "two" }, { title: "one" }, { title: "Three" }],
    });
    const byTitle = Object.fromEntries(again.subtasks.map((s: any) => [s.title.toLowerCase(), s]));
    expect(byTitle.one.id).toBe(t.subtasks[0].id);
    expect(byTitle.one.done).toBe(true);
    expect(byTitle.two.id).toBe(t.subtasks[1].id);
    expect(byTitle.three.done).toBe(false);
    expect(again).toMatchObject({ dueDate: dates.today, labels: ["a"], priority: "low", title: "Keep ids" });
  });

  test("clear empties fields that an empty string cannot", async () => {
    const t = await mcp.json("upsert_task", {
      projectId: P, title: "Clearable", assignee: "Alex", dueDate: dates.today, milestone: "m", priority: "high",
      storyPoints: 3, labels: ["x"], description: "d", sprintId: "s1",
      links: [{ url: "https://example.com" }], acceptance: [{ title: "ac" }], subtasks: [{ title: "st" }],
    });
    const cleared = await mcp.json("upsert_task", {
      projectId: P, taskId: t.id,
      clear: ["assignee", "dueDate", "milestone", "priority", "storyPoints", "labels", "description", "sprintId", "links", "acceptance", "subtasks"],
    });
    for (const f of ["assignee", "dueDate", "milestone", "priority", "description", "sprintId"]) expect(cleared[f] ?? "", f).toBe("");
    for (const f of ["labels", "links", "acceptance", "subtasks"]) expect(cleared[f] ?? [], f).toEqual([]);
    expect(cleared.storyPoints ?? 0).toBe(0);
    expect(cleared.title).toBe("Clearable");
  });

  test("invalid input is rejected and nothing is stored", async () => {
    const before = JSON.stringify(readProject().tasks);
    const bad: Record<string, unknown>[] = [
      { title: "x", dueDate: "someday" },
      { title: "x", dueDate: "2026-02-30" },
      { title: "x", links: [{ url: "javascript:alert(1)" }] },
      { title: "x", links: [{ url: "file:///etc/passwd" }] },
      { title: "x", subtasks: [{ title: "  " }] },
      { taskId: "t-today", clear: ["title"] },
      { taskId: "t-today", clear: ["id"] },
      { taskId: "nope", title: "x" },
      { title: "" },
    ];
    for (const args of bad) {
      const r = await mcp.call("upsert_task", { projectId: P, ...args });
      expect(r.isError, JSON.stringify(args)).toBe(true);
    }
    expect(JSON.stringify(readProject().tasks)).toBe(before);
  });

  test("append_task_note adds a timestamped line without replacing the description", async () => {
    const t = await mcp.json("upsert_task", { projectId: P, title: "Notes", description: "Original text" });
    const r1 = await mcp.json("append_task_note", { projectId: P, taskId: t.id, note: "Waiting on Support" });
    const r2 = await mcp.json("append_task_note", { projectId: P, taskId: t.id, note: "Support replied" });
    expect(r2.description.startsWith("Original text\n\nNote (")).toBe(true);
    expect(r2.description).toMatch(/Waiting on Support\n\nNote \(.*\): Support replied$/);
    expect(r1.description.length).toBeLessThan(r2.description.length);
    expect((await mcp.call("append_task_note", { projectId: P, taskId: t.id, note: "  " })).isError).toBe(true);
  });
});

test.describe("filtering over MCP", () => {
  const ids = async (args: Record<string, unknown>) =>
    ((await mcp.json("list_tasks", { projectId: P, ...args })) as any[]).map((t) => t.id).sort();
  const all = () => readProject().tasks as any[];
  const want = (pred: (t: any) => boolean) => all().filter(pred).map((t) => t.id).sort();

  test("due-date presets match an independent oracle", async () => {
    for (const preset of ["today", "tomorrow", "last_quarter", "this_quarter", "next_quarter", "last_week", "this_week", "next_week", "this_month", "next_month", "q1", "q2", "q3", "q4", "this_year"]) {
      const range = expectedRange(preset, dates.today);
      expect(await ids({ duePreset: preset }), preset).toEqual(want((t) => inRange(t.dueDate, range)));
    }
  });

  test("explicit bounds, overdue, no due date, and preset+bound narrowing", async () => {
    expect(await ids({ dueFrom: dates.overdue, dueTo: dates.today })).toEqual(
      want((t) => inRange(t.dueDate, [dates.overdue, dates.today]))
    );
    expect(await ids({ overdue: true })).toEqual(want((t) => t.dueDate && t.dueDate < dates.today && t.status !== "done"));
    expect(await ids({ noDueDate: true })).toEqual(want((t) => !t.dueDate));
    const q = expectedRange("this_year", dates.today);
    expect(await ids({ duePreset: "this_year", dueFrom: dates.today })).toEqual(
      want((t) => inRange(t.dueDate, [dates.today, q[1]]))
    );
  });

  test("status, priority, type, sprint, assignee, label, acceptance, subtasks (comma lists are OR)", async () => {
    expect(await ids({ status: "todo,inprogress" })).toEqual(want((t) => ["todo", "inprogress"].includes(t.status)));
    expect(await ids({ priority: "HIGH" })).toEqual(want((t) => t.priority === "high"));
    expect(await ids({ type: "bug,story" })).toEqual(want((t) => ["bug", "story"].includes(t.type)));
    expect(await ids({ sprintId: "s1" })).toEqual(want((t) => t.sprintId === "s1"));
    expect(await ids({ sprintId: "backlog" })).toEqual(want((t) => !t.sprintId));
    expect(await ids({ assignee: "sam" })).toEqual(["t-overdue", "t-today"]);
    expect(await ids({ label: "STRIPE" })).toEqual(["t-done-late", "t-overdue"]);
    expect(await ids({ hasAcceptance: true })).toEqual(["t-overdue"]);
    expect(await ids({ hasSubtasks: true })).toEqual(["t-today"]);
    expect(await ids({ parentId: "story-1" })).toEqual(["child-1", "child-2"]);
    expect(await ids({ status: "todo", label: "urgent", overdue: true })).toEqual(["t-overdue"]);
  });

  test("bad presets and dates are tool errors", async () => {
    for (const args of [{ duePreset: "someday" }, { dueFrom: "10/01/2026" }, { dueTo: "2026-13-01" }]) {
      expect((await mcp.call("list_tasks", { projectId: P, ...args })).isError, JSON.stringify(args)).toBe(true);
    }
  });
});

test.describe("attachments over MCP", () => {
  test("attach files from the project root and read them back by type", async () => {
    const txt = await mcp.json("add_task_attachment", { projectId: P, taskId: "t-today", path: "notes.md" });
    const png = await mcp.json("add_task_attachment", { projectId: P, taskId: "t-today", path: "shot.png", name: "Screenshot.png" });
    const pdf = await mcp.json("add_task_attachment", { projectId: P, taskId: "t-today", path: "doc.pdf" });
    expect(txt).toMatchObject({ name: "notes.md", size: fs.statSync(path.join(root, "notes.md")).size });
    expect(png.name).toBe("Screenshot.png");
    expect(png.mime).toBe("image/png");
    expect(readTask("t-today").attachments.map((a: any) => a.name)).toEqual(["notes.md", "Screenshot.png", "doc.pdf"]);
    expect(fs.existsSync(path.join(DATA_DIR, "attachments", P, "t-today", txt.file))).toBe(true);

    const text = await mcp.call("read_task_attachment", { projectId: P, taskId: "t-today", attachmentId: txt.id });
    expect(text.text).toContain("Release is planned for Friday");
    const image = await mcp.call("read_task_attachment", { projectId: P, taskId: "t-today", attachmentId: png.id });
    expect(image.content.some((c: any) => c.type === "image" && c.mimeType === "image/png" && c.data.length > 10)).toBe(true);
    const meta = await mcp.call("read_task_attachment", { projectId: P, taskId: "t-today", attachmentId: pdf.id });
    expect(meta.text).toContain("metadata is available");
    expect(meta.text).not.toContain("%PDF");
  });

  test("path escapes, absolute paths, symlinks and folders are refused", async () => {
    const outside = path.join(path.dirname(root), "outside-secret.txt");
    fs.writeFileSync(outside, "TOP SECRET");
    fs.symlinkSync(outside, path.join(root, "link.txt"));
    fs.mkdirSync(path.join(root, "folder"), { recursive: true });
    for (const p of ["../outside-secret.txt", outside, "link.txt", "folder", "missing.txt", "", "../../etc/hosts"]) {
      const r = await mcp.call("add_task_attachment", { projectId: P, taskId: "t-today", path: p });
      expect(r.isError, p).toBe(true);
    }
    expect(readTask("t-today").attachments ?? []).toEqual([]);
    expect(fs.existsSync(path.join(DATA_DIR, "attachments", P, "t-today"))).toBe(false);
    fs.rmSync(outside, { force: true });
  });

  test("an attachment id from another task does not resolve; remove trashes the file", async () => {
    const a = await mcp.json("add_task_attachment", { projectId: P, taskId: "t-today", path: "notes.md" });
    expect((await mcp.call("read_task_attachment", { projectId: P, taskId: "t-overdue", attachmentId: a.id })).isError).toBe(true);
    expect((await mcp.call("remove_task_attachment", { projectId: P, taskId: "t-today", attachmentId: "nope" })).isError).toBe(true);
    await mcp.json("remove_task_attachment", { projectId: P, taskId: "t-today", attachmentId: a.id });
    expect(readTask("t-today").attachments ?? []).toEqual([]);
    const taskDir = path.join(DATA_DIR, "attachments", P, "t-today");
    expect(fs.existsSync(taskDir) ? fs.readdirSync(taskDir) : []).not.toContain(a.file);
    const trash = path.join(DATA_DIR, "attachments", ".trash");
    expect(fs.existsSync(trash)).toBe(true);
  });

  test("MCP-attached files appear in the UI, and UI-added files are readable over MCP", async ({ page }) => {
    await mcp.json("add_task_attachment", { projectId: P, taskId: "t-today", path: "shot.png" });
    await openBoard(page);
    await expect(card(page, "Due today task").locator(".kb-pill.files")).toHaveText("📎 1");
    await openTask(page, "Due today task");
    await expect(page.locator(".kb-attachment-thumb img")).toHaveCount(1);

    await page.evaluate(() => {
      const dt = new DataTransfer();
      dt.items.add(new File([new TextEncoder().encode("added in the UI")], "ui.txt", { type: "text/plain" }));
      document.querySelector(".kb-attachments")!.dispatchEvent(new DragEvent("drop", { dataTransfer: dt, bubbles: true, cancelable: true }));
    });
    await expect(page.locator(".kb-attachment")).toHaveCount(2);
    await page.getByRole("button", { name: "Save", exact: true }).click();
    const task = await mcp.json("get_task", { projectId: P, taskId: "t-today" });
    const ui = task.attachments.find((a: any) => a.name === "ui.txt");
    expect(ui).toBeTruthy();
    const read = await mcp.call("read_task_attachment", { projectId: P, taskId: "t-today", attachmentId: ui.id });
    expect(read.text).toContain("added in the UI");
  });
});

test.describe("quarters over MCP", () => {
  const fiscal = [
    { start: "07-01", end: "09-30" }, { start: "10-01", end: "12-31" },
    { start: "01-01", end: "03-31" }, { start: "04-01", end: "06-30" },
  ];
  const fiscalPairs: [string, string][] = fiscal.map((q) => [q.start, q.end]);

  test("app-wide quarters change what q1..q4 mean for list_tasks", async () => {
    const r = await mcp.json("set_quarters", { quarters: fiscal });
    expect(r.scope).toBe("app-wide");
    expect(JSON.parse(fs.readFileSync(path.join(DATA_DIR, "calendar.json"), "utf8")).quarters).toHaveLength(4);
    const q = await mcp.json("get_quarters", {});
    expect(q.appWide.map((x: any) => x.start)).toEqual(["07-01", "10-01", "01-01", "04-01"]);
    for (const preset of ["q1", "q2", "q3", "q4"]) {
      const range = expectedRange(preset, dates.today, fiscalPairs);
      const got = ((await mcp.json("list_tasks", { projectId: P, duePreset: preset })) as any[]).map((t) => t.id).sort();
      expect(got, preset).toEqual((readProject().tasks as any[]).filter((t) => inRange(t.dueDate, range)).map((t) => t.id).sort());
    }
  });

  test("a project override beats app-wide; reset removes it", async () => {
    await mcp.json("set_quarters", { quarters: fiscal });
    const custom = [
      { start: "02-01", end: "04-30" }, { start: "05-01", end: "07-31" },
      { start: "08-01", end: "10-31" }, { start: "11-01", end: "01-31" },
    ];
    await mcp.json("set_quarters", { projectId: P, quarters: custom });
    expect(readProject().quarters[0].start).toBe("02-01");
    const q = await mcp.json("get_quarters", { projectId: P });
    expect(q.effective[0].start).toBe("02-01");
    expect(q.appWide[0].start).toBe("07-01");

    const wrapping = expectedRange("q4", dates.today, custom.map((x) => [x.start, x.end]) as [string, string][]);
    const got = ((await mcp.json("list_tasks", { projectId: P, duePreset: "q4" })) as any[]).map((t) => t.id).sort();
    expect(got).toEqual((readProject().tasks as any[]).filter((t) => inRange(t.dueDate, wrapping)).map((t) => t.id).sort());

    await mcp.json("set_quarters", { projectId: P, reset: true });
    expect(readProject().quarters ?? []).toEqual([]);
    expect((await mcp.json("get_quarters", { projectId: P })).effective[0].start).toBe("07-01"); // falls back to app-wide
    await mcp.json("set_quarters", { reset: true });
    expect(fs.existsSync(path.join(DATA_DIR, "calendar.json"))).toBe(false);
  });

  test("invalid quarter sets are rejected", async () => {
    for (const quarters of [fiscal.slice(0, 2), [...fiscal.slice(0, 3), { start: "13-01", end: "03-31" }], [...fiscal.slice(0, 3), { start: "04-31", end: "06-30" }], []]) {
      expect((await mcp.call("set_quarters", { quarters })).isError, JSON.stringify(quarters)).toBe(true);
    }
    expect(fs.existsSync(path.join(DATA_DIR, "calendar.json"))).toBe(false);
  });

  test("quarters set over MCP show up in Settings > Calendar", async ({ page }) => {
    await mcp.json("set_quarters", { quarters: fiscal });
    await openBoard(page);
    await page.getByRole("button", { name: "Settings" }).click();
    await page.locator(".prefs-nav-item", { hasText: "Calendar" }).click();
    await expect(page.getByLabel("Q1 start month")).toHaveValue("7");
    await expect(page.getByLabel("Q2 start month")).toHaveValue("10");
  });
});
