import { test, expect } from "@playwright/test";
import { reseed, openBoard, card, openTask, readTask, Mcp, type Dates } from "./helpers";

let dates: Dates;
test.beforeEach(() => {
  dates = reseed().dates;
});

test.describe("past-due rendering", () => {
  test("overdue open tasks get a red border, a Past due tag and a date pill", async ({ page }) => {
    const { errors } = await openBoard(page);
    const overdue = card(page, "Overdue bug");
    await expect(overdue).toHaveClass(/overdue/);
    await expect(overdue.locator(".kb-pill.past-due")).toHaveText("Past due");
    await expect(overdue.locator(".kb-pill.due")).toHaveClass(/overdue/);

    // The CSS really applies: a red border, not just a class name.
    const border = await overdue.evaluate((el) => getComputedStyle(el).borderTopColor);
    const normal = await card(page, "Next week item").evaluate((el) => getComputedStyle(el).borderTopColor);
    expect(border).not.toBe(normal);
    expect(border).toMatch(/rgb\(2[0-9]{2}, (0|[0-9]{1,2}), (0|[0-9]{1,2})\)/); // strongly red
    expect(errors).toEqual([]);
  });

  test("done, due-today, future and undated tasks are not flagged", async ({ page }) => {
    await openBoard(page);
    for (const title of ["Done but late", "Due today task", "Next week item", "No date item"]) {
      await expect(card(page, title)).not.toHaveClass(/overdue/);
      await expect(card(page, title).locator(".kb-pill.past-due")).toHaveCount(0);
    }
    await expect(card(page, "Done but late").locator(".kb-pill.due")).toHaveCount(1); // still shows its date
    await expect(card(page, "No date item").locator(".kb-pill.due")).toHaveCount(0);
  });

  test("a story shows past-due state only for its own date", async ({ page }) => {
    await openBoard(page);
    await expect(card(page, "Checkout revamp")).not.toHaveClass(/overdue/);
  });

  test("completing an overdue task clears the flag", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Overdue bug");
    await page.locator(".kb-detail .field", { hasText: "Status" }).locator("select").selectOption("done");
    await page.getByRole("button", { name: "Save", exact: true }).click();
    await expect(card(page, "Overdue bug")).not.toHaveClass(/overdue/);
    expect(readTask("t-overdue").status).toBe("done");
    expect(readTask("t-overdue").done).toBe(true);
  });
});

test.describe("due date editing", () => {
  test("set, persist across reload, and show in the modal summary", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "No date item");
    const input = page.locator(".kb-due-field input[type=date]");
    await expect(input).toHaveValue("");
    await input.fill(dates.nextWeek);
    await page.getByRole("button", { name: "Save", exact: true }).click();

    expect(readTask("t-nodate").dueDate).toBe(dates.nextWeek);
    await page.reload();
    await openBoard(page);
    await expect(card(page, "No date item").locator(".kb-pill.due")).toBeVisible();
    await openTask(page, "No date item");
    await expect(page.locator(".kb-due-field input[type=date]")).toHaveValue(dates.nextWeek);
    await expect(page.locator(".kb-due-summary")).toContainText(/Due in \d+ days?/);
  });

  test("a past date shows Past due by N days in red; Clear removes it", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Overdue bug");
    await expect(page.locator(".kb-due-summary")).toHaveText("Past due by 3 days");
    await expect(page.locator(".kb-due-summary")).toHaveClass(/overdue/);
    await page.locator(".kb-due-field").getByRole("button", { name: "Clear" }).click();
    await expect(page.locator(".kb-due-summary")).toHaveCount(0);
    await page.getByRole("button", { name: "Save", exact: true }).click();
    await expect(card(page, "Overdue bug")).not.toHaveClass(/overdue/);
    expect(readTask("t-overdue").dueDate ?? "").toBe("");
  });

  test("Cancel leaves the stored due date unchanged", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Overdue bug");
    await page.locator(".kb-due-field input[type=date]").fill(dates.yearEnd);
    await page.getByRole("button", { name: "Cancel" }).click();
    expect(readTask("t-overdue").dueDate).toBe(dates.overdue);
  });

  test("a due date set through the UI is readable over MCP, and vice versa", async ({ page }) => {
    const mcp = await Mcp.connect();
    await openBoard(page);
    await openTask(page, "No date item");
    await page.locator(".kb-due-field input[type=date]").fill(dates.monthEnd);
    await page.getByRole("button", { name: "Save", exact: true }).click();
    const viaMcp = await mcp.json("get_task", { projectId: "e2e-proj", taskId: "t-nodate" });
    expect(viaMcp.dueDate).toBe(dates.monthEnd);

    await mcp.json("upsert_task", { projectId: "e2e-proj", taskId: "t-nodate", dueDate: dates.overdue });
    await page.reload();
    await openBoard(page);
    await expect(card(page, "No date item")).toHaveClass(/overdue/);
  });
});

test.describe("existing data", () => {
  test("pre-upgrade tasks (no new fields) load and render unchanged", async ({ page }) => {
    reseed({ legacy: true });
    const { errors } = await openBoard(page);
    await expect(page.locator(".kb-card")).toHaveCount(14); // 16 tasks, 2 are children inside a story
    await expect(card(page, "Overdue bug")).not.toHaveClass(/overdue/); // legacy has no due date
    await openTask(page, "Overdue bug");
    await expect(page.locator(".kb-due-field input[type=date]")).toHaveValue("");
    // Saving a legacy task adds nothing unexpected and keeps its content.
    await page.getByRole("button", { name: "Save", exact: true }).click();
    const t = readTask("t-overdue");
    expect(t.title).toBe("Overdue bug");
    expect(t.labels).toEqual(["stripe", "urgent"]);
    expect(t.acceptance).toHaveLength(1);
    expect(errors).toEqual([]);
  });
});
