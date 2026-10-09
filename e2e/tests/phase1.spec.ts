import { test, expect, type Page } from "@playwright/test";
import http from "node:http";
import fs from "node:fs";
import path from "node:path";
import { reseed, openBoard, openTask, card, readTask, readProject, DATA_DIR, Mcp, PROJECT_ID } from "./helpers";

// ---- streaming mock Ollama ----
type Req = { path: string; body: any; aborted: boolean };
let server: http.Server;
let requests: Req[] = [];
let script: { thinking?: string[]; content: string; delayMs?: number } = { content: "{}" };
const PORT = 11998;

test.beforeAll(async () => {
  server = http.createServer((req, res) => {
    let raw = "";
    req.on("data", (c) => (raw += c));
    req.on("end", async () => {
      if (req.url === "/api/tags") {
        res.end(JSON.stringify({ models: [{ name: "qwen3-think:8b" }, { name: "gpt-oss:20b" }, { name: "plain:1b" }] }));
        return;
      }
      const body = raw ? JSON.parse(raw) : {};
      if (req.url === "/api/show") {
        const caps = /think|gpt-oss/.test(body.model) ? ["completion", "thinking"] : ["completion"];
        res.end(JSON.stringify({ capabilities: caps }));
        return;
      }
      const entry: Req = { path: req.url || "", body, aborted: false };
      requests.push(entry);
      res.on("close", () => {
        if (!res.writableEnded) entry.aborted = true;
      });
      res.setHeader("content-type", "application/x-ndjson");
      const wait = (ms: number) => new Promise((r) => setTimeout(r, ms));
      for (const t of script.thinking || []) {
        if (entry.aborted) return;
        res.write(JSON.stringify({ message: { thinking: t } }) + "\n");
        await wait(script.delayMs || 0);
      }
      if (entry.aborted) return;
      res.end(JSON.stringify({ message: { content: script.content }, done: true }) + "\n");
    });
  });
  await new Promise<void>((r) => server.listen(PORT, "127.0.0.1", r));
});
test.afterAll(async () => new Promise((r) => server.close(r)));

test.beforeEach(() => {
  reseed();
  requests = [];
  fs.rmSync(path.join(DATA_DIR, "settings.json"), { force: true });
});

const settingsFile = () => {
  const p = path.join(DATA_DIR, "settings.json");
  return fs.existsSync(p) ? JSON.parse(fs.readFileSync(p, "utf8")) : {};
};
const writeSettings = (s: Record<string, unknown>) =>
  fs.writeFileSync(path.join(DATA_DIR, "settings.json"), JSON.stringify(s));

async function useModel(page: Page, model: string) {
  await page.addInitScript((m) => {
    localStorage.setItem("ollamaHost", "http://127.0.0.1:11998");
    localStorage.setItem("ollamaModel", m);
  }, model);
}
async function openSettings(page: Page, section: string) {
  await page.getByRole("button", { name: "Settings" }).click();
  await page.locator(".prefs-nav-item", { hasText: section }).click();
}

test.describe("AI thinking effort", () => {
  test("the slider persists app-wide and explains what the model supports", async ({ page }) => {
    await useModel(page, "qwen3-think:8b");
    await openBoard(page);
    await openSettings(page, "AI");
    const slider = page.locator("#think-effort");
    await expect(page.locator(".prefs-hint", { hasText: "Thinking" }).or(page.getByText("only turns reasoning on or off"))).toBeVisible();
    await slider.fill("4"); // High
    await expect.poll(() => settingsFile().thinkLevel).toBe("high");
    await expect(page.locator(".think-ticks .on")).toHaveText("High");
    await slider.fill("1");
    await expect.poll(() => settingsFile().thinkLevel).toBe("off");
  });

  test("a non-thinking model is explained and never sent a think value", async ({ page }) => {
    writeSettings({ thinkLevel: "high" });
    script = { content: JSON.stringify({ reply: "ok", stories: [] }) };
    await useModel(page, "plain:1b");
    await openBoard(page);
    await openSettings(page, "AI");
    await expect(page.getByText("doesn't reason before answering")).toBeVisible();
    await page.keyboard.press("Escape");
    await page.getByRole("button", { name: /AI$/ }).first().click();
    await page.getByPlaceholder(/Ask about the code/).fill("hello");
    await page.getByRole("button", { name: "Send" }).click();
    await expect.poll(() => requests.length).toBe(1);
    expect("think" in requests[0].body).toBe(false);
  });

  test("chat sends the effort, streams reasoning with a timer, and the per-chat override wins", async ({ page }) => {
    writeSettings({ thinkLevel: "low" });
    script = { thinking: ["Considering the board. ", "Checking due dates."], content: JSON.stringify({ reply: "Done thinking.", stories: [] }), delayMs: 700 };
    await useModel(page, "gpt-oss:20b");
    await openBoard(page);
    await page.getByRole("button", { name: /AI$/ }).first().click();
    await page.getByPlaceholder(/Ask about the code/).fill("what is overdue?");
    await page.getByRole("button", { name: "Send" }).click();
    await expect(page.locator(".ai-progress-status")).toContainText(/Thinking · \ds/);
    await page.locator(".ai-thinking summary").click();
    await expect(page.locator(".ai-thinking pre")).toContainText("Considering the board.");
    await expect(page.locator(".chat-bubble", { hasText: "Done thinking." })).toBeVisible();
    expect(requests[0].body.think).toBe("low"); // gpt-oss takes levels
    expect(requests[0].body.stream).toBe(true);

    await page.getByLabel("Thinking effort for this chat").selectOption("high");
    script = { content: JSON.stringify({ reply: "Second.", stories: [] }) };
    await page.getByPlaceholder(/Ask about the code/).fill("again");
    await page.getByRole("button", { name: "Send" }).click();
    await expect(page.locator(".chat-bubble", { hasText: "Second." })).toBeVisible();
    expect(requests[1].body.think).toBe("high");
  });

  test("Stop aborts a long chat reply and records it without an error", async ({ page }) => {
    script = { thinking: Array(30).fill("still thinking… "), content: "{}", delayMs: 500 };
    await useModel(page, "qwen3-think:8b");
    await openBoard(page);
    await page.getByRole("button", { name: /AI$/ }).first().click();
    await page.getByPlaceholder(/Ask about the code/).fill("long question");
    await page.getByRole("button", { name: "Send" }).click();
    await expect(page.locator(".ai-progress-status")).toContainText("Thinking");
    await page.locator(".chat-composer").getByRole("button", { name: "Stop", exact: true }).click();
    await expect(page.locator(".chat-bubble", { hasText: "Stopped before the model finished." })).toBeVisible();
    await expect.poll(() => requests[0]?.aborted).toBe(true);
    await expect(page.locator(".toast")).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Send" })).toBeVisible();
  });

  test("Populate with AI shows progress and can be stopped", async ({ page }) => {
    script = { thinking: Array(30).fill("hmm "), content: "{}", delayMs: 500 };
    await useModel(page, "qwen3-think:8b");
    await openBoard(page);
    await openTask(page, "No date item");
    await page.getByRole("button", { name: /Populate with AI/ }).click();
    await expect(page.locator(".ai-inline-progress")).toContainText(/Thinking · \ds/);
    await page.locator(".ai-inline-progress").getByRole("button", { name: "Stop" }).click();
    await expect(page.locator(".ai-inline-progress")).toHaveCount(0);
    await expect.poll(() => requests[0]?.aborted).toBe(true);
    await expect(page.locator(".toast")).toHaveCount(0);
  });
});

test.describe("autosave", () => {
  test("off by default: Save is shown and Esc discards edits", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "No date item");
    await expect(page.getByRole("button", { name: "Save", exact: true })).toBeVisible();
    await page.locator(".kb-detail .field", { hasText: "Priority" }).locator("select").selectOption("high");
    await page.keyboard.press("Escape");
    await expect(page.locator(".kb-detail")).toHaveCount(0);
    expect(readTask("t-nodate").priority ?? "").toBe("");
  });

  test("toggle in Settings > Tasks persists app-wide", async ({ page }) => {
    await openBoard(page);
    await openSettings(page, "Tasks");
    await page.locator(".prefs-row", { hasText: "Autosave task edits" }).locator(".switch").click();
    await expect.poll(() => settingsFile().autosave).toBe(true);
  });

  test("on: selects save immediately, text saves on blur, Save is hidden, Esc drops only the field in progress", async ({ page }) => {
    writeSettings({ autosave: true });
    await openBoard(page);
    await openTask(page, "No date item");
    await expect(page.getByRole("button", { name: "Save", exact: true })).toHaveCount(0);
    await expect(page.locator(".autosave-state")).toHaveText("Autosave on");

    await page.locator(".kb-detail .field", { hasText: "Priority" }).locator("select").selectOption("high");
    await expect.poll(() => readTask("t-nodate").priority).toBe("high");
    await expect(page.locator(".autosave-state")).toHaveText("All changes saved");
    await expect(page.locator(".kb-detail")).toBeVisible(); // stays open

    const title = page.locator(".kb-detail .field", { hasText: "Title" }).locator("input");
    await title.fill("Renamed by autosave");
    expect(readTask("t-nodate").title).toBe("No date item"); // not while typing
    await page.locator(".kb-detail textarea").click(); // blur the title
    await expect.poll(() => readTask("t-nodate").title).toBe("Renamed by autosave");

    await page.locator(".kb-detail textarea").fill("half-typed note");
    await page.keyboard.press("Escape"); // still focused: this edit is dropped
    await expect(page.locator(".kb-detail")).toHaveCount(0);
    const t = readTask("t-nodate");
    expect(t.description ?? "").toBe("");
    expect(t.title).toBe("Renamed by autosave");
    expect(t.priority).toBe("high");
    await expect(card(page, "Renamed by autosave")).toBeVisible();
  });

  test("on: files added are kept even if the editor is then cancelled", async ({ page }) => {
    writeSettings({ autosave: true });
    await openBoard(page);
    await openTask(page, "No date item");
    await page.evaluate(() => {
      const dt = new DataTransfer();
      dt.items.add(new File([new TextEncoder().encode("keep")], "kept.txt", { type: "text/plain" }));
      document.querySelector(".kb-attachments")!.dispatchEvent(new DragEvent("drop", { dataTransfer: dt, bubbles: true, cancelable: true }));
    });
    await expect.poll(() => (readTask("t-nodate").attachments || []).length).toBe(1);
    await page.getByRole("button", { name: "Cancel" }).click();
    const att = readTask("t-nodate").attachments[0];
    expect(fs.existsSync(path.join(DATA_DIR, "attachments", PROJECT_ID, "t-nodate", att.file))).toBe(true);
  });
});

test.describe("board layout editing", () => {
  const colIds = () => (readProject().columns || []).map((c: any) => `${c.id}:${c.label}`);

  test("nothing is draggable until Edit board; Cancel discards", async ({ page }) => {
    await openBoard(page);
    await expect(page.locator(".board-editor")).toHaveCount(0);
    await page.getByRole("button", { name: "Edit board" }).click();
    await page.getByLabel("Name of column 2").fill("Changed");
    await page.locator(".board-editor").getByRole("button", { name: "Cancel" }).click();
    await expect(page.locator(".kb-col-title").nth(1)).toHaveText("To Do");
    expect(readProject().columns ?? []).toEqual([]);
  });

  test("rename, reorder, add, delete with task move; saved in one step", async ({ page }) => {
    await openBoard(page);
    const inProgress = (readProject().tasks as any[]).filter((t) => t.status === "inprogress").map((t) => t.id);
    await page.getByRole("button", { name: "Edit board" }).click();
    await page.getByLabel("Name of column 2").fill("Ready");
    // Move Testing (4th) left of In Progress (3rd).
    await page.locator(".board-editor-col[data-col=testing]").getByRole("button", { name: "Move left" }).click();
    await page.getByPlaceholder("New column name").fill("Waiting on Vendor");
    await page.getByRole("button", { name: "+ Add column" }).click();
    await page.locator(".board-editor-col[data-col=inprogress]").getByRole("button", { name: "Delete" }).click();
    const done = page.locator(".board-editor").getByRole("button", { name: "Done" });
    await expect(done).toBeDisabled(); // In Progress still has tasks
    await expect(page.locator(".board-editor-problems")).toContainText("Choose where");
    await page.getByLabel("Destination for In Progress").selectOption("todo");
    await done.click();
    await expect(page.locator(".board-editor")).toHaveCount(0);

    expect(colIds()).toEqual(["backlog:Backlog", "todo:Ready", "testing:Testing", "done:Done", "waitingonvendor:Waiting on Vendor"]);
    for (const id of inProgress) expect(readTask(id).status).toBe("todo");
    await expect(page.locator(".kb-col-title")).toHaveText(["Backlog", "Ready", "Testing", "Done", "Waiting on Vendor"]);
    await expect(page.locator(".kb-col", { has: page.locator(".kb-col-title", { hasText: "Ready" }) }).locator(".kb-card", { hasText: "Due today task" })).toBeVisible();
  });

  test("drag and drop reorders columns", async ({ page }) => {
    await openBoard(page);
    await page.getByRole("button", { name: "Edit board" }).click();
    await page.locator(".board-editor-col[data-col=done]").dragTo(page.locator(".board-editor-col[data-col=backlog]"));
    await page.locator(".board-editor").getByRole("button", { name: "Done" }).click();
    await expect.poll(() => colIds()[0]).toBe("done:Done");
  });
});

test.describe("project JSON", () => {
  test("export shows every task with status; import previews then applies adds and updates", async ({ page }) => {
    await openBoard(page);
    await page.getByRole("button", { name: "JSON", exact: true }).click();
    await expect(page.getByLabel("Exported JSON")).not.toHaveValue("");
    const exported = JSON.parse(await page.getByLabel("Exported JSON").inputValue());
    expect(exported.format).toBe("jumpstart-project");
    expect(exported.project.tasks).toHaveLength(16);
    expect(exported.project.tasks.find((t: any) => t.id === "t-overdue")).toMatchObject({ status: "todo", priority: "high" });

    await page.getByRole("button", { name: "Import / Update" }).click();
    const doc = exported;
    doc.project.tasks.find((t: any) => t.id === "t-nodate").status = "done";
    doc.project.tasks.push({ title: "Imported task", status: "todo", dueDate: "2026-12-01", acceptance: [{ title: "Imported AC" }] });
    await page.getByLabel("JSON to import").fill(JSON.stringify(doc, null, 2));
    await page.getByRole("button", { name: "Preview changes" }).click();
    await expect(page.locator(".json-preview .add")).toContainText("Add 1: Imported task");
    await expect(page.locator(".json-preview .upd")).toContainText("No date item (done, status)");
    expect(readTask("t-nodate").status).toBe("backlog"); // nothing saved yet
    await page.getByRole("button", { name: /Apply 2 changes/ }).click();
    await expect(page.locator(".project-json")).toHaveCount(0);
    await expect(card(page, "Imported task")).toBeVisible();
    expect(readTask("t-nodate").status).toBe("done");
    const added = (readProject().tasks as any[]).find((t) => t.title === "Imported task");
    expect(added.acceptance[0].id).toBeTruthy();
  });

  test("replace mode removes absent tasks; bad JSON is reported and nothing changes", async ({ page }) => {
    await openBoard(page);
    await page.getByRole("button", { name: "JSON", exact: true }).click();
    await page.getByRole("button", { name: "Import / Update" }).click();
    await page.getByLabel("JSON to import").fill('{"tasks":[{"id":"t-overdue","title":"Overdue bug","status":"todo","type":"bug"}]}');
    await page.getByRole("button", { name: "Replace" }).click();
    await page.getByRole("button", { name: "Preview changes" }).click();
    await expect(page.locator(".json-preview .del")).toContainText("Remove 15");

    await page.getByLabel("JSON to import").fill('{"tasks":[{"title":"x","status":"nope"}]}');
    await page.getByRole("button", { name: "Preview changes" }).click();
    await expect(page.locator(".project-json .error")).toContainText("not a column");
    await page.getByRole("button", { name: "Cancel" }).click();
    expect(readProject().tasks).toHaveLength(16);
  });

  test("the same import works over MCP", async () => {
    const mcp = await Mcp.connect();
    const out = await mcp.json("import_project", { projectId: PROJECT_ID, dryRun: true, json: '{"tasks":[{"title":"Via MCP","status":"todo"}]}' });
    expect(out.preview.added).toEqual(["Via MCP"]);
    expect(readProject().tasks).toHaveLength(16);
  });
});

test.describe("data safety with older JumpStart builds", () => {
  // An older build decodes config.json without the new fields and writes it
  // back without them. The new build must bring them back on next load.
  const simulateOldBuildSave = () => {
    const file = path.join(DATA_DIR, "config.json");
    const projects = JSON.parse(fs.readFileSync(file, "utf8"));
    for (const p of projects) {
      delete p.quarters;
      for (const t of p.tasks) {
        delete t.dueDate;
        delete t.links;
        delete t.attachments;
      }
      p.tasks.find((t: any) => t.id === "t-today").title = "Edited in old build";
    }
    fs.writeFileSync(file, JSON.stringify(projects, null, 2));
  };

  test("due dates and links dropped by an older build come back; its own edits are kept", async ({ page }) => {
    const mcp = await Mcp.connect();
    await mcp.json("upsert_task", { projectId: PROJECT_ID, taskId: "t-today", links: [{ title: "Spec", url: "https://example.com/spec" }] });
    simulateOldBuildSave();
    await openBoard(page);
    await expect(card(page, "Edited in old build").locator(".kb-pill.links")).toHaveText("🔗 1");
    await expect(card(page, "Overdue bug")).toHaveClass(/overdue/);
    const viaMcp = await mcp.json("get_task", { projectId: PROJECT_ID, taskId: "t-overdue" });
    expect(viaMcp.dueDate).toBeTruthy();
  });

  test("a due date cleared in the new build stays cleared", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Overdue bug");
    await page.locator(".kb-due-field").getByRole("button", { name: "Clear" }).click();
    await page.getByRole("button", { name: "Save", exact: true }).click();
    simulateOldBuildSave();
    await page.reload();
    await openBoard(page);
    await expect(card(page, "Overdue bug")).not.toHaveClass(/overdue/);
    expect(readTask("t-overdue").dueDate ?? "").toBe("");
  });
});

test.describe("concurrent edits (board open while an agent edits)", () => {
  test("saving from a stale board keeps fields an agent added meanwhile", async ({ page }) => {
    await openBoard(page); // the board now holds its copy of every task
    const mcp = await Mcp.connect();
    // An agent refines tasks after the board loaded (this is what was lost before).
    await mcp.json("upsert_task", { projectId: PROJECT_ID, taskId: "t-nodate", dueDate: "2026-11-30",
      links: [{ title: "Spec", url: "https://example.com/spec" }], acceptance: [{ title: "Agent AC" }] });
    await mcp.json("upsert_task", { projectId: PROJECT_ID, taskId: "t-lastweek", subtasks: [{ title: "Agent step" }] });

    // The stale board edits a different task...
    await openTask(page, "Next week item");
    await page.locator(".kb-detail .field", { hasText: "Priority" }).locator("select").selectOption("low");
    await page.getByRole("button", { name: "Save", exact: true }).click();
    // ...and the same task the agent changed (it never saw the agent's fields).
    await openTask(page, "No date item");
    await page.locator(".kb-detail .field", { hasText: "Title" }).locator("input").fill("No date item (renamed)");
    await page.getByRole("button", { name: "Save", exact: true }).click();

    await expect.poll(() => readTask("t-nodate").title).toBe("No date item (renamed)");
    const t = readTask("t-nodate");
    expect(t.dueDate).toBe("2026-11-30");
    expect(t.links).toHaveLength(1);
    expect(t.acceptance.map((a: any) => a.title)).toEqual(["Agent AC"]);
    expect(readTask("t-lastweek").subtasks.map((s: any) => s.title)).toEqual(["Agent step"]);
    expect(readTask("t-nextweek").priority).toBe("low");
    // The board adopted the agent's changes after saving.
    await expect(card(page, "No date item (renamed)").locator(".kb-pill.links")).toHaveText("🔗 1");
  });

  test("dragging a card on a stale board does not undo agent changes", async ({ page }) => {
    await openBoard(page);
    const mcp = await Mcp.connect();
    await mcp.json("upsert_task", { projectId: PROJECT_ID, taskId: "t-q1", dueDate: "2026-12-24", labels: ["agent"] });
    // HTML5 drag and drop, dispatched the way a real drag fires it.
    await page.evaluate(async () => {
      const cardEl = [...document.querySelectorAll(".kb-card")].find((c) => c.textContent?.includes("Next week item"))!;
      const col = [...document.querySelectorAll(".kb-col")].find((c) => c.querySelector(".kb-col-title")?.textContent === "In Progress")!;
      const dt = new DataTransfer();
      cardEl.dispatchEvent(new DragEvent("dragstart", { dataTransfer: dt, bubbles: true }));
      await new Promise((r) => setTimeout(r, 50));
      col.dispatchEvent(new DragEvent("dragover", { dataTransfer: dt, bubbles: true, cancelable: true }));
      col.dispatchEvent(new DragEvent("drop", { dataTransfer: dt, bubbles: true, cancelable: true }));
    });
    await expect.poll(() => readTask("t-nextweek").status).toBe("inprogress");
    expect(readTask("t-q1")).toMatchObject({ dueDate: "2026-12-24", labels: ["agent"] });
  });

  test("an agent's new task and a deletion elsewhere survive a stale save", async ({ page }) => {
    await openBoard(page);
    const mcp = await Mcp.connect();
    await mcp.json("upsert_task", { projectId: PROJECT_ID, title: "Created by agent", status: "todo" });
    await mcp.json("delete_task", { projectId: PROJECT_ID, taskId: "t-q2" }).catch(() => {});
    await openTask(page, "Q3 item");
    await page.locator(".kb-detail .field", { hasText: "Priority" }).locator("select").selectOption("high");
    await page.getByRole("button", { name: "Save", exact: true }).click();
    await expect.poll(() => readTask("t-q3").priority).toBe("high");
    const titles = (readProject().tasks as any[]).map((t) => t.title);
    expect(titles).toContain("Created by agent"); // not dropped by the stale list
    expect(titles).not.toContain("Q2 item"); // not resurrected
  });

  test("returning to the window shows changes made elsewhere", async ({ page }) => {
    await openBoard(page);
    const mcp = await Mcp.connect();
    await mcp.json("upsert_task", { projectId: PROJECT_ID, taskId: "t-q4", title: "Q4 item (edited by agent)" });
    await page.evaluate(() => window.dispatchEvent(new Event("focus")));
    await expect(card(page, "Q4 item (edited by agent)")).toBeVisible();
  });
});
