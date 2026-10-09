import { test, expect, type Page } from "@playwright/test";
import {
  reseed, openBoard, openFilters, visibleTitles, chip, readProject, expectedRange, inRange,
  DATA_DIR, type Dates,
} from "./helpers";
import fs from "node:fs";
import path from "node:path";

let dates: Dates;
test.beforeEach(() => {
  dates = reseed().dates;
});

type T = { id: string; title: string; parentId?: string; [k: string]: any };
const topLevel = () => (readProject().tasks as T[]).filter((t) => !t.parentId);
const childrenOf = (id: string) => (readProject().tasks as T[]).filter((t) => t.parentId === id);
/** Oracle: a top-level task shows if it, or any child, satisfies the predicate. */
const expectTitles = (pred: (t: T) => boolean) =>
  topLevel()
    .filter((t) => pred(t) || childrenOf(t.id).some(pred))
    .map((t) => t.title)
    .sort();
const shown = async (page: Page) => (await visibleTitles(page)).sort();

const today = () => dates.today;

test.describe("filter panel basics", () => {
  test("toggle opens the panel; the badge counts active filter groups", async ({ page }) => {
    await openBoard(page);
    await expect(page.locator(".kb-filters")).toHaveCount(0);
    await expect(page.locator(".kb-filter-toggle")).toHaveText("Filters");
    await openFilters(page);
    await (await chip(page, "Priority", "High")).click();
    await (await chip(page, "Priority", "Low")).click(); // same group: still one filter
    await expect(page.locator(".kb-filter-toggle")).toHaveText("Filters · 1");
    await (await chip(page, "Type", "Bug")).click();
    await expect(page.locator(".kb-filter-toggle")).toHaveText("Filters · 2");
    await expect(page.locator(".kb-filter-toggle")).toHaveClass(/active/);
  });

  test("Clear all filters restores every card and resets the badge", async ({ page }) => {
    await openBoard(page);
    const before = await shown(page);
    await openFilters(page);
    await (await chip(page, "Status", "Done")).click();
    expect((await shown(page)).length).toBeLessThan(before.length);
    await page.getByRole("button", { name: "Clear all filters" }).click();
    expect(await shown(page)).toEqual(before);
    await expect(page.locator(".kb-filter-toggle")).toHaveText("Filters");
  });

  test("no matches shows the filter empty state", async ({ page }) => {
    await openBoard(page);
    await openFilters(page);
    await (await chip(page, "Status", "Testing")).click(); // nothing is in Testing
    expect(await shown(page)).toEqual([]);
    await expect(page.locator(".kb-empty").first()).toHaveText("No tasks match the current filters.");
  });
});

test.describe("each filter against the seeded data", () => {
  test("status", async ({ page }) => {
    await openBoard(page);
    await openFilters(page);
    await (await chip(page, "Status", "In Progress")).click();
    expect(await shown(page)).toEqual(expectTitles((t) => t.status === "inprogress"));
    await (await chip(page, "Status", "Done")).click(); // multi-select within a group is OR
    expect(await shown(page)).toEqual(expectTitles((t) => ["inprogress", "done"].includes(t.status)));
  });

  test("priority incl. None", async ({ page }) => {
    await openBoard(page);
    await openFilters(page);
    await (await chip(page, "Priority", "High")).click();
    expect(await shown(page)).toEqual(expectTitles((t) => t.priority === "high"));
    await (await chip(page, "Priority", "None")).click();
    expect(await shown(page)).toEqual(expectTitles((t) => t.priority === "high" || !t.priority));
  });

  test("type", async ({ page }) => {
    await openBoard(page);
    await openFilters(page);
    await (await chip(page, "Type", "Bug")).click();
    expect(await shown(page)).toEqual(["Overdue bug"]);
    await (await chip(page, "Type", "Story")).click();
    expect(await shown(page)).toEqual(["Checkout revamp", "Overdue bug"]);
  });

  test("sprint chips replace the sprint bar selection, including Backlog", async ({ page }) => {
    await openBoard(page, { allSprints: false }); // default: Sprint 1 only
    const sprint1 = await shown(page);
    expect(sprint1).not.toContain("No date item"); // backlog task, hidden by the sprint bar
    await openFilters(page);
    await (await chip(page, "Sprint", "Backlog")).click();
    expect(await shown(page)).toEqual(["No date item"]);
    await (await chip(page, "Sprint", "Sprint 1")).click();
    expect(await shown(page)).toEqual(expectTitles((t) => (t.sprintId || "") === "" || t.sprintId === "s1"));
    await (await chip(page, "Sprint", "Sprint 2")).click(); // empty sprint adds nothing
    expect((await shown(page)).length).toBeGreaterThan(sprint1.length);
  });

  test("assignee: case-insensitive tokens, comma lists, and Unassigned", async ({ page }) => {
    await openBoard(page);
    await openFilters(page);
    // Options come from the data: Ada, Alex, sam/Sam merged case-insensitively.
    const options = await page.locator(".kb-filter-group", { hasText: "Assignee" }).locator(".kb-chip").allInnerTexts();
    expect(options).toEqual(["Unassigned", "@Ada", "@Alex", "@Sam"]);
    await (await chip(page, "Assignee", "@Sam")).click();
    expect(await shown(page)).toEqual(["Due today task", "Overdue bug"]); // "sam" and "Alex, Sam"
    await (await chip(page, "Assignee", "@Sam")).click();
    await (await chip(page, "Assignee", "Unassigned")).click();
    const unassigned = expectTitles((t) => !t.assignee);
    expect(await shown(page)).toEqual(unassigned);
    expect(unassigned).toContain("No date item");
  });

  test("labels: any-of, case-insensitive", async ({ page }) => {
    await openBoard(page);
    await openFilters(page);
    await (await chip(page, "Labels", "stripe")).click();
    expect(await shown(page)).toEqual(["Done but late", "Overdue bug"]);
    await (await chip(page, "Labels", "urgent")).click();
    expect(await shown(page)).toEqual(["Done but late", "Overdue bug"]); // urgent adds nothing new
  });

  test("acceptance criteria and subtasks: has / none", async ({ page }) => {
    await openBoard(page);
    await openFilters(page);
    const group = (name: string) => page.locator(".kb-filter-group", { hasText: name });
    await group("Acceptance criteria").getByRole("button", { name: "Has some" }).click();
    expect(await shown(page)).toEqual(["Overdue bug"]);
    await group("Acceptance criteria").getByRole("button", { name: "None" }).click();
    expect(await shown(page)).toEqual(expectTitles((t) => !(t.acceptance || []).length));
    await group("Acceptance criteria").getByRole("button", { name: "Any" }).click();
    await group("Subtasks").getByRole("button", { name: "Has some" }).click();
    expect(await shown(page)).toEqual(["Due today task"]);
  });

  test("past due only / no due date", async ({ page }) => {
    await openBoard(page);
    await openFilters(page);
    await page.getByLabel("Past due only").check();
    const overdue = expectTitles((t) => !!t.dueDate && t.dueDate < today() && t.status !== "done");
    expect(await shown(page)).toEqual(overdue);
    expect(overdue).toContain("Checkout revamp"); // via its overdue child
    expect(overdue).not.toContain("Done but late");
    await page.getByLabel("Past due only").uncheck();
    await page.getByLabel("No due date").check();
    expect(await shown(page)).toEqual(expectTitles((t) => !t.dueDate));
  });

  test("groups combine with AND", async ({ page }) => {
    await openBoard(page);
    await openFilters(page);
    await (await chip(page, "Labels", "stripe")).click();
    await (await chip(page, "Status", "Done")).click();
    expect(await shown(page)).toEqual(["Done but late"]);
    await (await chip(page, "Priority", "High")).click(); // stripe + done + high -> nothing
    expect(await shown(page)).toEqual([]);
  });

  test("search and filters apply together", async ({ page }) => {
    await openBoard(page);
    await openFilters(page);
    await (await chip(page, "Labels", "stripe")).click();
    await page.locator(".kb-search-input").fill("bug");
    expect(await shown(page)).toEqual(["Overdue bug"]);
  });
});

test.describe("due-date ranges", () => {
  const presets = [
    ["Last week", "last_week"], ["This week", "this_week"], ["Next week", "next_week"],
    ["This month", "this_month"], ["Next month", "next_month"],
    ["Q1", "q1"], ["Q2", "q2"], ["Q3", "q3"], ["Q4", "q4"], ["This year", "this_year"],
  ] as const;

  for (const [label, id] of presets) {
    test(`preset: ${label}`, async ({ page }) => {
      await openBoard(page);
      await openFilters(page);
      await page.locator(".kb-filter-due select").selectOption(id);
      const range = expectedRange(id, today());
      await expect(page.locator(".kb-filter-range")).toBeVisible();
      // Tasks with dates inside the range, plus stories whose children are inside.
      const want = expectTitles((t) => inRange(t.dueDate, range));
      await expect.poll(() => shown(page), { message: `${id} ${range}` }).toEqual(want);
      // The panel shows the resolved dates so quarter boundaries are not a mystery.
      const text = await page.locator(".kb-filter-range").innerText();
      expect(text).toMatch(/–/);
    });
  }

  test("custom range with date pickers (inclusive endpoints)", async ({ page }) => {
    await openBoard(page);
    await openFilters(page);
    await page.locator(".kb-filter-due select").selectOption("__custom__");
    await page.getByLabel("Due from").fill(dates.today);
    await page.getByLabel("Due to").fill(dates.today);
    await expect.poll(() => shown(page)).toEqual(["Due today task"]);
    await page.getByLabel("Due from").fill(dates.overdue);
    const want = expectTitles((t) => inRange(t.dueDate, [dates.overdue, dates.today]));
    await expect.poll(() => shown(page)).toEqual(want);
    // Open-ended: only a start date.
    await page.getByLabel("Due to").fill("");
    await expect.poll(() => shown(page)).toEqual(expectTitles((t) => !!t.dueDate && t.dueDate >= dates.overdue));
  });

  test("picking a preset clears a custom range and vice versa", async ({ page }) => {
    await openBoard(page);
    await openFilters(page);
    await page.locator(".kb-filter-due select").selectOption("__custom__");
    await page.getByLabel("Due from").fill(dates.today);
    await page.locator(".kb-filter-due select").selectOption("this_year");
    await expect(page.getByLabel("Due from")).toHaveCount(0);
    await expect(page.locator(".kb-filter-toggle")).toHaveText("Filters · 1");
  });

  test("undated tasks never match a date range", async ({ page }) => {
    await openBoard(page);
    await openFilters(page);
    await page.locator(".kb-filter-due select").selectOption("this_year");
    expect(await shown(page)).not.toContain("No date item");
  });
});

test.describe("quarter dates", () => {
  const openCalendarSettings = async (page: Page) => {
    await page.getByRole("button", { name: "Settings" }).click();
    await page.locator(".prefs-nav-item", { hasText: "Calendar" }).click();
    await expect(page.locator(".quarter-editor")).toBeVisible();
  };

  test("Settings > Calendar defaults to calendar-year quarters", async ({ page }) => {
    await openBoard(page);
    await openCalendarSettings(page);
    await expect(page.locator(".quarter-row")).toHaveCount(4);
    await expect(page.getByLabel("Q1 start month")).toHaveValue("1");
    await expect(page.getByLabel("Q4 end month")).toHaveValue("12");
    await expect(page.getByLabel("Q4 end day")).toHaveValue("31");
    await expect(page.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  test("a fiscal year (Jul-Jun) re-points Q1..Q4 and persists to disk", async ({ page }) => {
    await openBoard(page);
    await openCalendarSettings(page);
    const set = async (q: number, sm: string, sd: string, em: string, ed: string) => {
      await page.getByLabel(`Q${q} start month`).selectOption(sm);
      await page.getByLabel(`Q${q} start day`).selectOption(sd);
      await page.getByLabel(`Q${q} end month`).selectOption(em);
      await page.getByLabel(`Q${q} end day`).selectOption(ed);
    };
    await set(1, "7", "1", "9", "30");
    await set(2, "10", "1", "12", "31");
    await set(3, "1", "1", "3", "31");
    await set(4, "4", "1", "6", "30");
    await page.getByRole("button", { name: "Save" }).click();
    await expect(page.getByText("Saved.")).toBeVisible();

    const saved = JSON.parse(fs.readFileSync(path.join(DATA_DIR, "calendar.json"), "utf8"));
    expect(saved.quarters.map((q: any) => [q.start, q.end])).toEqual([
      ["07-01", "09-30"], ["10-01", "12-31"], ["01-01", "03-31"], ["04-01", "06-30"],
    ]);

    await page.keyboard.press("Escape");
    await page.reload();
    await openBoard(page);
    await openFilters(page);
    const fiscal: [string, string][] = [["07-01", "09-30"], ["10-01", "12-31"], ["01-01", "03-31"], ["04-01", "06-30"]];
    for (const id of ["q1", "q2", "q3", "q4"]) {
      await page.locator(".kb-filter-due select").selectOption(id);
      const range = expectedRange(id, today(), fiscal);
      await expect.poll(() => shown(page), { message: `fiscal ${id} ${range}` })
        .toEqual(expectTitles((t) => inRange(t.dueDate, range)));
    }
  });

  test("Use calendar year removes the override", async ({ page }) => {
    fs.writeFileSync(
      path.join(DATA_DIR, "calendar.json"),
      JSON.stringify({ quarters: [
        { name: "Q1", start: "07-01", end: "09-30" }, { name: "Q2", start: "10-01", end: "12-31" },
        { name: "Q3", start: "01-01", end: "03-31" }, { name: "Q4", start: "04-01", end: "06-30" },
      ] })
    );
    await openBoard(page);
    await openCalendarSettings(page);
    await expect(page.getByLabel("Q1 start month")).toHaveValue("7");
    await page.getByRole("button", { name: "Use calendar year" }).click();
    await expect(page.getByLabel("Q1 start month")).toHaveValue("1");
    expect(fs.existsSync(path.join(DATA_DIR, "calendar.json"))).toBe(false);
  });

  test("a project override beats the app-wide setting", async ({ page }) => {
    fs.writeFileSync(
      path.join(DATA_DIR, "calendar.json"),
      JSON.stringify({ quarters: [
        { name: "Q1", start: "07-01", end: "09-30" }, { name: "Q2", start: "10-01", end: "12-31" },
        { name: "Q3", start: "01-01", end: "03-31" }, { name: "Q4", start: "04-01", end: "06-30" },
      ] })
    );
    await openBoard(page);
    await page.getByRole("button", { name: "Edit", exact: true }).click();
    const modal = page.locator(".modal").filter({ hasText: "Quarter dates" });
    await expect(modal.getByText("Following the app-wide quarter dates")).toBeVisible();
    await modal.getByLabel("Use different quarter dates for this project").check();
    await expect(modal.locator(".quarter-row")).toHaveCount(4);
    await modal.getByRole("button", { name: "Save" }).click();

    const project = readProject();
    expect(project.quarters).toHaveLength(4);
    expect(project.quarters[0].start).toBe("01-01"); // project default (calendar year) wins over app-wide Jul

    await page.reload();
    await openBoard(page);
    await openFilters(page);
    await page.locator(".kb-filter-due select").selectOption("q1");
    const range = expectedRange("q1", today());
    await expect.poll(() => shown(page)).toEqual(expectTitles((t) => inRange(t.dueDate, range)));
  });
});
