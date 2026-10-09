import { test, expect, type Page } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";
import { expectedRange, reseed, openBoard, openFilters, visibleTitles, chip, readTask, readProject, DATA_DIR, Mcp, PROJECT_ID, type Dates } from "./helpers";

let dates: Dates;
test.beforeEach(() => {
  dates = reseed().dates;
  fs.rmSync(path.join(DATA_DIR, "filters.json"), { force: true });
});
const appFilters = () => {
  const p = path.join(DATA_DIR, "filters.json");
  return fs.existsSync(p) ? JSON.parse(fs.readFileSync(p, "utf8")).filters : [];
};
const shown = async (page: Page) => (await visibleTitles(page)).sort();

test.describe("saved filters", () => {
  test("defaults are offered and apply in one click from the picker", async ({ page }) => {
    await openBoard(page);
    const picker = page.getByLabel("Saved filter");
    const options = await picker.locator("option").allInnerTexts();
    for (const name of ["Due today", "Due this week", "Past due", "Due this month", "Due this quarter", "High priority", "Needs my action", "Blocked", "No due date", "Missing acceptance criteria"]) {
      expect(options).toContain(name);
    }
    await picker.selectOption({ label: "Past due" });
    await expect(page.locator(".kb-filter-toggle")).toHaveText("Filters · 1");
    expect(await shown(page)).toContain("Overdue bug");
    expect(await shown(page)).not.toContain("Done but late");
    await picker.selectOption({ label: "Clear saved filter" });
    await expect(page.locator(".kb-filter-toggle")).toHaveText("Filters");
  });

  test("save current filters app-wide, rename, delete; and a project-only filter", async ({ page }) => {
    await openBoard(page);
    await openFilters(page);
    await (await chip(page, "Priority", "High")).click();
    await (await chip(page, "Type", "Bug")).click();
    await page.getByLabel("Saved filter name").fill("Hot bugs");
    await page.getByRole("button", { name: "Save filter" }).click();
    await expect.poll(() => appFilters().map((f: any) => f.name)).toContain("Hot bugs");
    const saved = appFilters().find((f: any) => f.name === "Hot bugs");
    expect(saved.query).toEqual({ priorities: ["high"], types: ["bug"] });

    // Clear, then re-apply from the saved chip.
    await page.getByRole("button", { name: "Clear all filters" }).click();
    await page.locator(".saved-filter .kb-chip", { hasText: "Hot bugs" }).click();
    await expect.poll(() => shown(page)).toEqual(["Overdue bug"]);

    await page.getByLabel("Rename Hot bugs").first().click();
    await page.getByLabel("Rename Hot bugs").fill("Urgent bugs");
    await page.keyboard.press("Enter");
    await expect.poll(() => appFilters().map((f: any) => f.name)).toContain("Urgent bugs");
    expect(appFilters().map((f: any) => f.name)).not.toContain("Hot bugs");

    // Project-only filter is stored on the project, not app-wide.
    await page.getByLabel("Saved filter name").fill("Mine only");
    await page.getByLabel("Save filter for").selectOption("project");
    await page.getByRole("button", { name: "Save filter" }).click();
    await expect.poll(() => (readProject().savedFilters || []).map((f: any) => f.name)).toEqual(["Mine only"]);
    expect(appFilters().map((f: any) => f.name)).not.toContain("Mine only");
    await expect(page.locator(".saved-filter", { hasText: "Mine only" }).locator(".saved-filter-scope")).toHaveText("project");

    await page.getByLabel("Delete Urgent bugs").click();
    await expect.poll(() => appFilters().map((f: any) => f.name)).not.toContain("Urgent bugs");
  });

  test("defaults can be deleted and restored; duplicate names are refused", async ({ page }) => {
    await openBoard(page);
    await openFilters(page);
    await page.getByLabel("Delete Blocked").click();
    await expect.poll(() => appFilters().map((f: any) => f.name)).not.toContain("Blocked");
    await page.reload();
    await openBoard(page);
    await openFilters(page);
    await expect(page.locator(".saved-filter", { hasText: "Blocked" })).toHaveCount(0); // stays deleted
    await page.getByRole("button", { name: "Restore defaults" }).click();
    await expect(page.locator(".saved-filter", { hasText: "Blocked" })).toHaveCount(1);

    await (await chip(page, "Priority", "Low")).click();
    await page.getByLabel("Saved filter name").fill("past DUE");
    await page.getByRole("button", { name: "Save filter" }).click();
    await expect(page.locator(".toast")).toContainText("already exists");
  });

  test("Due this quarter and Due today use live dates", async ({ page }) => {
    await openBoard(page);
    await page.getByLabel("Saved filter").selectOption({ label: "Due today" });
    await expect.poll(() => shown(page)).toEqual(["Due today task"]);
    await page.getByLabel("Saved filter").selectOption({ label: "Due this quarter" });
    const quarter = expectedRange("this_quarter", dates.today);
    const inQuarter = (readProject().tasks as any[])
      .filter((t) => t.dueDate && t.dueDate >= quarter[0] && t.dueDate <= quarter[1] && !t.parentId)
      .map((t) => t.title);
    await expect.poll(async () => (await shown(page)).filter((t) => inQuarter.includes(t)).length).toBe(inQuarter.length);
  });

  test("saved filters are usable over MCP", async () => {
    const mcp = await Mcp.connect();
    const lists = await mcp.json("list_saved_filters", { projectId: PROJECT_ID });
    expect(lists.appWide.length).toBeGreaterThanOrEqual(10);
    const out = await mcp.json("run_saved_filter", { projectId: PROJECT_ID, filterId: "default-due-today" });
    expect(out.tasks.map((t: any) => t.title)).toEqual(["Due today task"]);
  });
});

test.describe("sheet view", () => {
  const openSheet = async (page: Page) => {
    await openBoard(page);
    await page.getByRole("tab", { name: "Sheet" }).click();
    await expect(page.locator(".sheet-table")).toBeVisible();
  };
  const rowTitles = (page: Page) => page.locator(".sheet-table tbody tr .sheet-title-text").allInnerTexts();
  const cell = (page: Page, taskId: string, col: string) => page.locator(`.sheet-table tr[data-task="${taskId}"] td[data-col="${col}"]`);

  test("shows every task as rows, children under their story, and the choice is remembered", async ({ page }) => {
    await openSheet(page);
    const titles = await rowTitles(page);
    expect(titles).toHaveLength(16);
    expect(titles.indexOf("Add payment form")).toBe(titles.indexOf("Checkout revamp") + 1);
    await expect(page.locator('.sheet-table tr[data-task="child-1"]')).toHaveClass(/child/);
    await expect(cell(page, "t-overdue", "due")).toContainText("Oct");
    await page.reload();
    await openBoard(page);
    await expect(page.locator(".sheet-table")).toBeVisible(); // view remembered per project
  });

  test("Excel-style filters: value checklist, text contains, date range; sort", async ({ page }) => {
    await openSheet(page);
    await page.getByRole("button", { name: "Status menu" }).click();
    await page.locator(".sheet-menu-check", { hasText: "(Select all)" }).locator("input").uncheck();
    await page.locator(".sheet-menu-check", { hasText: "In Progress" }).locator("input").check();
    await page.keyboard.press("Escape");
    expect((await rowTitles(page)).sort()).toEqual(["Checkout revamp", "Due today task"]);
    await page.getByRole("button", { name: /Clear 1 column filter/ }).click();

    await page.getByRole("button", { name: "Title menu" }).click();
    await page.getByLabel("Filter Title containing").fill("item");
    await page.keyboard.press("Escape");
    expect((await rowTitles(page)).every((t) => t.includes("item"))).toBe(true);
    await page.getByRole("button", { name: /Clear 1 column filter/ }).click();

    await page.getByRole("button", { name: "Due menu" }).click();
    await page.getByLabel("Date filter type").selectOption("empty");
    await page.keyboard.press("Escape");
    expect((await rowTitles(page)).sort()).toEqual(["Checkout revamp", "No date item", "Write form tests"].sort());
    await page.getByRole("button", { name: /Clear 1 column filter/ }).click();

    await page.locator(".sheet-th-label", { hasText: "Due" }).click(); // ascending
    const first = (await rowTitles(page))[0];
    expect(first).toBe("Last year item");
  });

  test("inline edit: double-click, type, Enter saves; Esc cancels; select cells", async ({ page }) => {
    await openSheet(page);
    await cell(page, "t-nodate", "title").dblclick();
    await page.getByLabel("Edit cell").fill("Edited in sheet");
    await page.keyboard.press("Enter");
    await expect.poll(() => readTask("t-nodate").title).toBe("Edited in sheet");

    await cell(page, "t-nodate", "priority").dblclick();
    await page.getByLabel("Edit cell").selectOption("high");
    await expect.poll(() => readTask("t-nodate").priority).toBe("high");

    await cell(page, "t-nodate", "due").dblclick();
    await page.getByLabel("Edit cell").fill(dates.nextWeek);
    await page.keyboard.press("Enter");
    await expect.poll(() => readTask("t-nodate").dueDate).toBe(dates.nextWeek);

    await cell(page, "t-nodate", "labels").dblclick();
    await page.getByLabel("Edit cell").fill("alpha, beta");
    await page.keyboard.press("Escape");
    expect(readTask("t-nodate").labels ?? []).toEqual([]); // cancelled

    await cell(page, "t-nodate", "status").dblclick();
    await page.getByLabel("Edit cell").selectOption("done");
    await expect.poll(() => readTask("t-nodate").status).toBe("done");
    expect(readTask("t-nodate").done).toBe(true);
  });

  test("keyboard: arrows move, Enter edits, Tab commits and moves right", async ({ page }) => {
    await openSheet(page);
    await cell(page, "t-overdue", "title").click();
    await page.keyboard.press("ArrowDown"); // Due today task
    await page.keyboard.press("Enter");
    await page.getByLabel("Edit cell").fill("Keyboard edited");
    await page.keyboard.press("Tab");
    await expect.poll(() => readTask("t-today").title).toBe("Keyboard edited");
    await expect(cell(page, "t-today", "type")).toHaveClass(/active/);
  });

  test("Add task, open details from the title, and bulk edit/delete", async ({ page }) => {
    await openSheet(page);
    await page.getByRole("button", { name: "+ Add task" }).click();
    await expect.poll(() => (readProject().tasks as any[]).filter((t) => t.title === "New task").length).toBe(1);
    await expect(page.locator(".sheet-title-text", { hasText: "New task" })).toBeVisible();

    await page.getByLabel("Open Q1 item").click();
    await expect(page.locator(".kb-detail")).toBeVisible();
    await expect(page.locator(".kb-detail .field", { hasText: "Title" }).locator("input")).toHaveValue("Q1 item");
    await page.getByRole("button", { name: "Cancel" }).click();

    await page.getByLabel("Select Q1 item").check();
    await page.getByLabel("Select Q2 item").check();
    await expect(page.locator(".sheet-bulk")).toContainText("2 selected");
    await page.getByLabel("Set priority for selected").selectOption("low");
    await expect.poll(() => [readTask("t-q1").priority, readTask("t-q2").priority]).toEqual(["low", "low"]);
    await page.locator(".sheet-bulk").getByRole("button", { name: "Delete" }).click();
    await page.getByRole("button", { name: "Delete", exact: true }).last().click();
    await expect.poll(() => (readProject().tasks as any[]).some((t) => t.id === "t-q1")).toBe(false);
    expect((readProject().tasks as any[]).some((t) => t.id === "t-q2")).toBe(false);
  });

  test("board filters and saved filters also apply to the sheet", async ({ page }) => {
    await openSheet(page);
    await page.getByLabel("Saved filter").selectOption({ label: "Due today" });
    await expect.poll(() => rowTitles(page)).toEqual(["Due today task"]);
  });

  test("hide and reset columns", async ({ page }) => {
    await openSheet(page);
    await page.getByRole("button", { name: "Columns", exact: true }).click();
    await page.locator(".sheet-cols .sheet-menu-check", { hasText: "Type" }).locator("input").uncheck();
    await expect(page.locator('.sheet-table th[data-col="type"]')).toHaveCount(0);
    await page.getByRole("button", { name: "Reset layout" }).click();
    await expect(page.locator('.sheet-table th[data-col="type"]')).toHaveCount(1);
  });
});
