import { test, expect, type Page } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";
import { reseed, readProject, DATA_DIR, Mcp, type Dates } from "./helpers";

let dates: Dates;
test.beforeEach(() => {
  dates = reseed().dates;
});

const layoutFile = () => path.join(DATA_DIR, "dashboard.json");
const savedLayout = () => (fs.existsSync(layoutFile()) ? JSON.parse(fs.readFileSync(layoutFile(), "utf8")) : null);
const widgetIds = async (page: Page) => page.locator(".dash-widget").evaluateAll((els) => els.map((e) => e.getAttribute("data-widget")));
const widget = (page: Page, id: string) => page.locator(`.dash-widget[data-widget="${id}"]`);
const openTasks = () => (readProject().tasks as any[]).filter((t) => !t.done && t.status !== "done");

async function openDashboard(page: Page) {
  await page.goto("/");
  await page.addStyleTag({ content: ".ad-overlay{display:none !important}" });
  await expect(page.locator(".dash-widgets")).toBeVisible();
  await expect(page.locator(".dash-widget").first()).toBeVisible();
}
const edit = (page: Page) => page.getByRole("button", { name: "Edit dashboard" }).click();
const done = async (page: Page) => {
  await page.getByRole("button", { name: "Done", exact: true }).click();
  await expect(page.getByRole("button", { name: "Edit dashboard" })).toBeVisible();
};

test.describe("default dashboard", () => {
  test("built-in panels plus due-date widgets with correct counts; a task opens in its editor", async ({ page }) => {
    await openDashboard(page);
    expect(await widgetIds(page)).toEqual(["stats", "due-overdue", "due-today", "due-week", "flow", "donut", "recent", "activity", "ports", "import"]);
    const overdue = openTasks().filter((t) => t.dueDate && t.dueDate < dates.today).length;
    await expect(widget(page, "due-overdue").locator(".dash-panel-tag")).toHaveText(String(overdue));
    const today = openTasks().filter((t) => t.dueDate === dates.today).map((t) => t.title).sort();
    await expect(widget(page, "due-today").locator(".dash-panel-tag")).toHaveText(String(today.length));
    expect((await widget(page, "due-today").locator(".dw-task-title").allInnerTexts()).sort()).toEqual(today);
    await expect(widget(page, "due-overdue").locator(".dw-due.late").first()).toBeVisible();
    await widget(page, "due-today").locator(".dw-task", { hasText: "Due today task" }).click();
    await expect(page.locator(".kb-detail .field", { hasText: "Title" }).locator("input")).toHaveValue("Due today task");
  });
});

test.describe("edit mode", () => {
  test("remove, reorder, resize; Done saves; Cancel discards", async ({ page }) => {
    await openDashboard(page);
    await edit(page);
    await page.getByRole("button", { name: "Remove Due today" }).click();
    await expect(widget(page, "due-today")).toHaveCount(0);
    await widget(page, "flow").getByRole("button", { name: "Move earlier" }).click();
    await widget(page, "donut").getByRole("radio", { name: "L" }).click();
    await expect(widget(page, "donut")).toHaveClass(/size-l/);
    expect(savedLayout()).toBeNull(); // nothing saved until Done
    await done(page);
    const ids = savedLayout().widgets.map((w: any) => w.id);
    expect(ids).not.toContain("due-today");
    expect(ids.indexOf("flow")).toBe(ids.indexOf("due-week") - 1);
    expect(savedLayout().widgets.find((w: any) => w.id === "donut").size).toBe("l");

    await page.reload();
    await expect(page.locator(".dash-widget")).toHaveCount(9);
    await expect(widget(page, "due-today")).toHaveCount(0);

    await edit(page);
    await page.getByRole("button", { name: "Remove Live ports" }).click();
    await page.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(widget(page, "ports")).toHaveCount(1);
    expect(savedLayout().widgets.map((w: any) => w.id)).toContain("ports");
  });

  test("drag a widget onto another to reorder", async ({ page }) => {
    await openDashboard(page);
    await edit(page);
    await widget(page, "due-week").dragTo(widget(page, "due-overdue"));
    expect(await widgetIds(page)).toEqual(["stats", "due-week", "due-overdue", "due-today", "flow", "donut", "recent", "activity", "ports", "import"]);
    await done(page);
    expect(savedLayout().widgets.slice(0, 3).map((w: any) => w.id)).toEqual(["stats", "due-week", "due-overdue"]);
  });

  test("add due, custom chart and saved-filter widgets from the gallery; edit a title", async ({ page }) => {
    await openDashboard(page);
    await edit(page);

    await page.getByRole("button", { name: "+ Add widget" }).click();
    await page.getByRole("button", { name: /^Due next quarter/ }).click();
    await page.getByRole("button", { name: "Add widget", exact: true }).click();

    await page.getByRole("button", { name: "+ Add widget" }).click();
    await page.getByRole("button", { name: /Custom chart or list/ }).click();
    await page.getByLabel("Title").fill("Bugs and stories by priority");
    await page.getByRole("checkbox", { name: "bug" }).check();
    await page.getByRole("checkbox", { name: "story" }).check();
    await page.getByLabel("Group by").selectOption("priority");
    await expect(page.locator(".we-preview .dw-bar-row")).toHaveCount(1); // live preview: both are high
    await page.getByRole("button", { name: "Add widget", exact: true }).click();

    await page.getByRole("button", { name: "+ Add widget" }).click();
    await page.getByRole("button", { name: /^Saved filter/ }).click();
    await page.getByLabel("Saved filter").selectOption({ label: "High priority" });
    await page.getByLabel("Show as").selectOption("count");
    await page.getByRole("button", { name: "Add widget", exact: true }).click();

    await done(page);
    const added = savedLayout().widgets.slice(-3);
    expect(added.map((w: any) => w.type)).toEqual(["due", "custom", "filter"]);
    expect(added[0].range).toBe("next_quarter");
    expect(added[1]).toMatchObject({ display: "bar", groupBy: "priority", query: { types: ["bug", "story"] } });
    expect(added.every((w: any) => /^w-/.test(w.id))).toBe(true);

    const chart = widget(page, added[1].id);
    await expect(chart.locator("h3")).toHaveText("Bugs and stories by priority");
    await expect(chart.locator(".dw-bar-label")).toHaveText(["High"]);
    await expect(chart.locator(".dw-bar-val")).toHaveText(["2"]);
    const high = openTasks().filter((t) => t.priority === "high").length;
    await expect(widget(page, added[2].id).locator(".dw-count strong")).toHaveText(String(high));
    await expect(widget(page, added[2].id).locator("h3")).toHaveText("High priority");

    await edit(page);
    await widget(page, added[0].id).getByRole("button", { name: "Edit" }).click();
    await page.getByLabel("Title").fill("Next quarter");
    await page.getByRole("button", { name: "Save widget" }).click();
    await done(page);
    expect(savedLayout().widgets.find((w: any) => w.id === added[0].id).title).toBe("Next quarter");
  });

  test("reset to default restores the original layout", async ({ page }) => {
    await openDashboard(page);
    await edit(page);
    await page.getByRole("button", { name: "Remove Overview" }).click();
    await done(page);
    await edit(page);
    await page.getByRole("button", { name: "Reset to default" }).click();
    await page.getByRole("button", { name: "Reset dashboard" }).click();
    await expect(widget(page, "stats")).toHaveCount(1);
    expect(savedLayout().widgets[0].id).toBe("stats");
  });
});

test.describe("import, export and code widgets", () => {
  const codeWidget = {
    type: "html",
    title: "Escape test",
    size: "m",
    query: { overdue: true },
    html: `<pre id="out">waiting</pre><script>
const r = {};
try { r.parentGo = typeof window.parent.go; } catch (e) { r.parentGo = "blocked"; }
try { r.parentDoc = window.parent.document.title; } catch (e) { r.parentDoc = "blocked"; }
try { localStorage.setItem("x", "1"); r.storage = "allowed"; } catch (e) { r.storage = "blocked"; }
try { document.cookie = "a=1"; r.cookie = document.cookie ? "allowed" : "blocked"; } catch (e) { r.cookie = "blocked"; }
addEventListener("message", async (e) => {
  if (e.data.type !== "jumpstart:data") return;
  try { await fetch("http://127.0.0.1:5173/"); r.fetch = "allowed"; } catch (err) { r.fetch = "blocked"; }
  r.total = e.data.data.total;
  r.first = e.data.data.tasks[0].title;
  out.textContent = JSON.stringify(r);
  jumpstart.open(e.data.data.tasks[0].projectId, "t-nodate"); // not in its data: must be ignored
});
</script><button id="open" onclick="jumpstart.open(window.p, window.t)">open</button>
<script>addEventListener("message", (e) => { if (e.data.type === "jumpstart:data") { window.p = e.data.data.tasks[0].projectId; window.t = e.data.data.tasks[0].taskId; } });</script>`,
  };

  async function importWidgets(page: Page, file: unknown) {
    await edit(page);
    await page.getByRole("button", { name: "Import…" }).click();
    await page.getByLabel("Widget JSON").fill(JSON.stringify(file));
    await page.getByRole("button", { name: "Check" }).click();
  }

  test("an imported code widget is flagged, sandboxed, and only gets its own data", async ({ page }) => {
    await openDashboard(page);
    await importWidgets(page, { jumpstartWidget: 1, widgets: [codeWidget] });
    await expect(page.locator(".we-warn")).toContainText("contains code");
    await page.getByRole("button", { name: "Add 1 widget" }).click();
    await done(page);

    const saved = savedLayout().widgets.at(-1);
    expect(saved.type).toBe("html");
    const frameEl = widget(page, saved.id).locator("iframe.dw-html");
    await expect(frameEl).toHaveAttribute("sandbox", "allow-scripts");
    const out = widget(page, saved.id).frameLocator("iframe").locator("#out");
    await expect(out).not.toHaveText("waiting", { timeout: 10000 });
    const r = JSON.parse(await out.innerText());
    const overdue = openTasks().filter((t) => t.dueDate && t.dueDate < dates.today).length;
    expect(r).toMatchObject({ parentGo: "blocked", parentDoc: "blocked", storage: "blocked", cookie: "blocked", fetch: "blocked", total: overdue });

    // Opening a task it was not given does nothing; one it was given opens.
    await page.waitForTimeout(500);
    await expect(page.locator(".kb-detail")).toHaveCount(0);
    await widget(page, saved.id).frameLocator("iframe").locator("#open").click();
    await expect(page.locator(".kb-detail .field", { hasText: "Title" }).locator("input")).toHaveValue(r.first);
  });

  test("bad files are refused; unknown projects fall back to all projects; export round-trips", async ({ page }) => {
    await openDashboard(page);
    await importWidgets(page, { jumpstartWidget: 1, widgets: [{ type: "due" }] });
    await expect(page.locator(".we-error")).toContainText("range");
    await page.getByLabel("Widget JSON").fill(JSON.stringify({ type: "due", range: "this_year", title: "Elsewhere", projectId: "not-here" }));
    await page.getByRole("button", { name: "Check" }).click();
    await expect(page.locator(".we-warn")).toContainText("all projects");
    await page.getByRole("button", { name: "Add 1 widget" }).click();
    await done(page);
    const w = savedLayout().widgets.at(-1);
    expect(w).toMatchObject({ type: "due", range: "this_year", title: "Elsewhere" });
    expect(w.projectId).toBeUndefined();

    const round = await page.evaluate(async (wid) => {
      const App = (window as any).go.main.App;
      const text = await App.ExportWidgets([wid]);
      return { text, parsed: await App.ParseWidgetImport(text) };
    }, w);
    expect(JSON.parse(round.text).jumpstartWidget).toBe(1);
    expect(round.parsed.widgets[0]).toMatchObject({ type: "due", range: "this_year", title: "Elsewhere" });
    expect(round.parsed.widgets[0].id).not.toBe(w.id);
  });
});

test.describe("MCP", () => {
  test("an AI can add a widget that then shows on the dashboard, read its data and delete it", async ({ page }) => {
    const mcp = await Mcp.connect();
    const schema = await mcp.json("widget_schema", {});
    expect(schema.dueRanges).toContain("next_quarter");
    const w = await mcp.json("upsert_widget", { type: "custom", title: "Open by type", display: "table", groupBy: "type", query: {} });
    await openDashboard(page);
    await expect(widget(page, w.id).locator("h3")).toHaveText("Open by type");
    const data = await mcp.json("get_widget_data", { widgetId: w.id });
    expect(data.total).toBe(openTasks().length);
    await expect(widget(page, w.id).locator(".dw-table tr")).toHaveCount(data.groups.length);
    await mcp.json("delete_widget", { widgetId: w.id });
    await page.reload();
    await expect(widget(page, w.id)).toHaveCount(0);
  });
});
