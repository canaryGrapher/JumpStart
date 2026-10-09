import { test, expect, type Page } from "@playwright/test";
import http from "node:http";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { reseed, openBoard, readProject, DATA_DIR, Mcp, PROJECT_ID, type Dates } from "./helpers";

const HERE = path.dirname(fileURLToPath(import.meta.url));

// Mock Ollama: OCR requests (with an image) get the image's "text"; other
// chat requests get whatever the test queued as the AI-search reply.
const PORT = 11998;
let server: http.Server;
let aiReply = "{}";
let requests: any[] = [];
test.beforeAll(async () => {
  server = http.createServer((req, res) => {
    let raw = "";
    req.on("data", (c) => (raw += c));
    req.on("end", () => {
      res.setHeader("content-type", "application/json");
      if (req.url === "/api/tags") {
        res.end(JSON.stringify({ models: [{ name: "mock-vision:1b" }, { name: "mock-text:1b" }] }));
        return;
      }
      if (req.url === "/api/show") {
        res.end(JSON.stringify({ capabilities: ["completion"] }));
        return;
      }
      const body = raw ? JSON.parse(raw) : {};
      requests.push(body);
      const isOCR = (body.messages || []).some((m: any) => m.images?.length);
      res.end(JSON.stringify({ message: { content: isOCR ? "Invoice 4471 whiteboard\nLedger migration plan" : aiReply }, done: true }));
    });
  });
  await new Promise<void>((r) => server.listen(PORT, "127.0.0.1", r));
});
test.afterAll(async () => {
  await new Promise((r) => server.close(r));
});

let dates: Dates;
let root: string;
test.beforeEach(() => {
  const s = reseed();
  dates = s.dates;
  root = s.root;
  requests = [];
});

const writeSettings = (patch: Record<string, unknown>) =>
  fs.writeFileSync(path.join(DATA_DIR, "settings.json"), JSON.stringify({ hotkeyMode: "app", hotkeyKey: "cmd+shift+j", ocrEngine: "off", dateOrder: "mdy", thinkLevel: "auto", ...patch }));

async function openPalette(page: Page) {
  await page.keyboard.press("Meta+k");
  await expect(page.getByRole("dialog", { name: "Search JumpStart" })).toBeVisible();
  await expect(page.getByLabel("Search everything")).toBeFocused();
  return page.getByLabel("Search everything");
}
const rows = (page: Page) => page.locator(".palette-row .palette-title");
const titles = async (page: Page) => (await rows(page).allInnerTexts()).map((t) => t.trim());
async function searchFor(page: Page, q: string) {
  const input = page.getByLabel("Search everything");
  await input.fill(q);
  // Results render after a short debounce.
  await expect(page.locator(".palette-results")).toBeVisible();
  await page.waitForTimeout(300);
}
const dueOn = (day: string) => (readProject().tasks as any[]).filter((t) => t.dueDate === day).map((t) => t.title).sort();

test.describe("command palette search", () => {
  test("⌘K opens it; words must all match; Enter opens the task in its editor", async ({ page }) => {
    await openBoard(page);
    await openPalette(page);
    await searchFor(page, "payment");
    expect(await titles(page)).toEqual(["Add payment form"]);
    await expect(page.locator(".palette-group").first()).toHaveText("Tasks");

    await searchFor(page, "item");
    expect((await titles(page)).length).toBeGreaterThan(5);
    await searchFor(page, "item payment"); // no single task has both words
    await expect(page.locator(".palette-empty")).toContainText("No matches");

    await searchFor(page, "zzqx nothing");
    expect(await titles(page)).toEqual([]);

    await searchFor(page, "checkout");
    await page.keyboard.press("Enter");
    await expect(page.getByRole("dialog", { name: "Search JumpStart" })).toHaveCount(0);
    await expect(page.locator(".kb-detail .field", { hasText: "Title" }).locator("input")).toHaveValue("Checkout revamp");
  });

  test("arrow keys move the selection and Esc closes", async ({ page }) => {
    await openBoard(page);
    await openPalette(page);
    await searchFor(page, "item");
    await expect(page.locator(".palette-row.active")).toHaveCount(1);
    const first = await page.locator(".palette-row.active .palette-title").innerText();
    await page.keyboard.press("ArrowDown");
    expect(await page.locator(".palette-row.active .palette-title").innerText()).not.toBe(first);
    await page.keyboard.press("Escape");
    await expect(page.getByRole("dialog", { name: "Search JumpStart" })).toHaveCount(0);
  });

  test("dates in many formats find tasks by due date", async ({ page }) => {
    await openBoard(page);
    await openPalette(page);
    const [y, m, d] = dates.today.split("-").map(Number);
    const mon = new Date(y, m - 1, d).toLocaleString("en-US", { month: "short" });
    const expected = dueOn(dates.today);
    for (const q of ["today", dates.today, `${m}/${d}/${y}`, `${m}/${d}`, `${mon} ${d}`, `${d} ${mon} ${y}`, `${y}${String(m).padStart(2, "0")}${String(d).padStart(2, "0")}`]) {
      await searchFor(page, q);
      expect((await titles(page)).sort(), q).toEqual(expected);
      await expect(page.locator(".palette-notes").first(), q).toContainText("due");
    }
    // A date plus words narrows further.
    await searchFor(page, `due today task ${dates.today}`);
    expect(await titles(page)).toEqual(["Due today task"]);
  });

  test("qualifiers: is:overdue, @assignee, #label, status:, type:, hide done", async ({ page }) => {
    await openBoard(page);
    await openPalette(page);
    await searchFor(page, "type:bug @sam #urgent");
    expect(await titles(page)).toEqual(["Overdue bug"]);
    await searchFor(page, "#stripe");
    expect((await titles(page)).sort()).toEqual(["Done but late", "Overdue bug"]);
    await page.getByLabel("Hide done").check();
    await page.waitForTimeout(300);
    expect(await titles(page)).toEqual(["Overdue bug"]);
    await page.getByLabel("Hide done").uncheck();
    await searchFor(page, "is:overdue");
    const t = await titles(page);
    expect(t).toContain("Overdue bug");
    expect(t).toContain("Add payment form");
    expect(t).not.toContain("Done but late");
    await searchFor(page, 'status:"in progress"');
    expect((await titles(page)).sort()).toEqual(["Checkout revamp", "Due today task"]);
    await searchFor(page, "due:someday");
    await expect(page.locator(".palette-error")).toContainText("not a date");
  });

  test("text inside attachments and criteria is searchable", async ({ page }) => {
    const mcp = await Mcp.connect();
    await mcp.json("add_task_attachment", { projectId: PROJECT_ID, taskId: "t-nodate", path: "notes.md" });
    await openBoard(page);
    await openPalette(page);
    await searchFor(page, "ops@example.com");
    await expect(page.locator(".palette-row", { hasText: "notes.md" })).toBeVisible();
    await expect(page.locator(".palette-row", { hasText: "notes.md" }).locator(".palette-meta")).toContainText("No date item");
    await expect(page.locator(".palette-row mark").first()).toBeVisible(); // highlighted
    await searchFor(page, "retries succeed");
    expect(await titles(page)).toEqual(["Overdue bug"]);
    await expect(page.locator(".palette-snippet").first()).toContainText("Acceptance criteria");
  });

  test("OCR through an Ollama vision model makes screenshot text searchable", async ({ page }) => {
    writeSettings({ ocrEngine: "ollama", ocrHost: `http://127.0.0.1:${PORT}`, ocrModel: "mock-vision:1b" });
    fs.copyFileSync(path.join(HERE, "fixtures", "ocr-text.png"), path.join(root, "whiteboard.png"));
    const mcp = await Mcp.connect();
    await mcp.json("add_task_attachment", { projectId: PROJECT_ID, taskId: "t-q4", path: "whiteboard.png" });
    await openBoard(page);
    await openPalette(page);
    await searchFor(page, "ledger migration");
    // The first search queues OCR; the palette refreshes when text lands.
    await expect(page.locator(".palette-row", { hasText: "whiteboard.png" })).toBeVisible({ timeout: 15000 });
    expect(requests.some((r) => r.model === "mock-vision:1b" && r.messages[0].images?.length === 1)).toBe(true);
    const ocrCalls = requests.length;
    await searchFor(page, "invoice 4471");
    await expect(page.locator(".palette-row", { hasText: "whiteboard.png" })).toBeVisible();
    expect(requests.length).toBe(ocrCalls); // cached, not read again

    // The same text is reachable over MCP.
    const out = await mcp.json("search", { query: "ledger" });
    expect(out.results.map((r: any) => r.fileName)).toContain("whiteboard.png");

    // The task editor shows the text status for the image.
    await page.keyboard.press("Enter");
    await expect(page.locator(".kb-attachment-ocr")).toContainText("searchable text");
  });

  test("with OCR off, image text is not searched", async ({ page }) => {
    writeSettings({ ocrEngine: "off" });
    fs.copyFileSync(path.join(HERE, "fixtures", "ocr-text.png"), path.join(root, "whiteboard.png"));
    const mcp = await Mcp.connect();
    await mcp.json("add_task_attachment", { projectId: PROJECT_ID, taskId: "t-q4", path: "whiteboard.png" });
    await openBoard(page);
    await openPalette(page);
    await searchFor(page, "ledger");
    await page.waitForTimeout(800);
    expect(await titles(page)).toEqual([]);
    expect(requests.length).toBe(0);
  });
});

test.describe("AI search", () => {
  const useModel = (page: Page) =>
    page.addInitScript((port) => {
      localStorage.setItem("ollamaHost", `http://127.0.0.1:${port}`);
      localStorage.setItem("ollamaModel", "mock-text:1b");
    }, PORT);

  test("the model picks a validated filter; results are real tasks", async ({ page }) => {
    await useModel(page);
    aiReply = JSON.stringify({ query: { assignees: ["sam", "Ghost"], priorities: ["high"] }, explain: "High priority work for Sam" });
    await openBoard(page);
    await openPalette(page);
    await page.keyboard.press("Tab");
    await page.getByLabel("Ask about your tasks").fill("what high priority things does sam have?");
    await page.keyboard.press("Enter");
    await expect(page.locator(".palette-plan")).toContainText("High priority work for Sam");
    await expect(page.locator(".palette-plan code")).toContainText(/"assignees":\["sam"\]/i);
    await expect(page.locator(".palette-warn")).toContainText('unknown assignee "Ghost"');
    expect(await titles(page)).toEqual(["Overdue bug"]);
    const req = requests.at(-1);
    expect(req.format).toBe("json");
    expect(req.messages[1].content).toContain("what high priority things does sam have?");
    expect(req.messages[1].content).toContain("Today is");
    await page.keyboard.press("Enter");
    await expect(page.locator(".kb-detail .field", { hasText: "Title" }).locator("input")).toHaveValue("Overdue bug");
  });

  test("an unusable reply is an error, never a made-up list", async ({ page }) => {
    await useModel(page);
    aiReply = JSON.stringify({ query: {}, explain: "Everything" });
    await openBoard(page);
    await openPalette(page);
    await page.getByRole("tab", { name: "Ask AI" }).click();
    await page.getByLabel("Ask about your tasks").fill("tell me a joke");
    await page.keyboard.press("Enter");
    await expect(page.locator(".palette-error")).toContainText("could not turn that into a filter");
    expect(await titles(page)).toEqual([]);
  });
});

test.describe("settings and entry points", () => {
  test("Settings > Search shows engines and turning the shortcut off disables ⌘K", async ({ page }) => {
    await openBoard(page);
    await page.getByRole("button", { name: /settings|preferences/i }).last().click();
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await expect(page.getByLabel("Read text in images")).toBeVisible();
    await expect(page.getByLabel("Dates like 10/9")).toHaveValue("mdy");
    await page.getByLabel("Search shortcut").selectOption("off");
    await expect.poll(() => JSON.parse(fs.readFileSync(path.join(DATA_DIR, "settings.json"), "utf8")).hotkeyMode).toBe("off");
    await page.keyboard.press("Escape");
    await page.keyboard.press("Meta+k");
    await page.waitForTimeout(300);
    await expect(page.getByRole("dialog", { name: "Search JumpStart" })).toHaveCount(0);
  });

  test("day-first date order reads 10/9 as 10 September", async ({ page }) => {
    writeSettings({ dateOrder: "dmy" });
    const [y, m, d] = dates.today.split("-").map(Number);
    await openBoard(page);
    await openPalette(page);
    await searchFor(page, `${d}/${m}/${y}`);
    expect((await titles(page)).sort()).toEqual(dueOn(dates.today));
  });

  test("Enter in the sidebar search opens the palette with that text", async ({ page }) => {
    await openBoard(page);
    await page.getByLabel("Search projects").fill("payment");
    await page.keyboard.press("Enter");
    await expect(page.getByLabel("Search everything")).toHaveValue("payment");
    await page.waitForTimeout(300);
    expect(await titles(page)).toEqual(["Add payment form"]);
  });
});

test("the system-wide shortcut registers with macOS, rejects bad keys, and unregisters", async ({ page }) => {
  await openBoard(page);
  const out = await page.evaluate(async () => {
    const App = (window as any).go.main.App;
    const s = await App.GetAppSettings();
    await App.SetAppSettings({ ...s, hotkeyMode: "system", hotkeyKey: "cmd+shift+j" });
    const on = await App.GetHotkeyStatus();
    let bad = "";
    try { await App.SetAppSettings({ ...s, hotkeyMode: "system", hotkeyKey: "shift+j" }); } catch (e) { bad = String(e); }
    await App.SetAppSettings({ ...s, hotkeyMode: "app" });
    return { on, bad, off: await App.GetHotkeyStatus() };
  });
  expect(out.on.registered).toBe(true);
  expect(out.bad).toContain("needs ⌘, ⌃ or ⌥");
  expect(out.off.registered).toBe(false);
});
