// Read-only smoke test of a COPY of the real ~/.jumpstart data (see scripts/stack.sh
// with E2E_COPY_REAL=1). Opens every project's board and checks it renders what the
// store says, with the new fields visible. Screenshots go to artifacts/real-data/.
import { test, expect } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";

const HOME = process.env.E2E_HOME || "/tmp/js-e2e-real-copy";
const projects: any[] = JSON.parse(fs.readFileSync(path.join(HOME, ".jumpstart", "config.json"), "utf8"));
const shots = path.join(process.cwd(), "artifacts", "real-data");
fs.mkdirSync(shots, { recursive: true });
const slug = (s: string) => s.toLowerCase().replace(/[^a-z0-9]+/g, "-").slice(0, 40);

for (const project of projects) {
  test(`board renders: ${project.name}`, async ({ page }) => {
    const errors: string[] = [];
    page.on("pageerror", (e) => !/reading 'nodes'/.test(e.message) && errors.push(e.message));
    await page.goto("/");
    await page.addStyleTag({ content: ".ad-overlay{display:none !important}" });
    await page.locator(".sidebar").getByText(project.name, { exact: true }).first().click();
    const tasksTab = page.getByRole("button", { name: /^Tasks/ }).first();
    if (!(await tasksTab.count())) test.skip(true, "project has tasks disabled");
    await tasksTab.click();
    await expect(page.locator(".kb-board")).toBeVisible();
    await page.getByRole("button", { name: /^All tasks/ }).click();

    const tasks: any[] = project.tasks || [];
    const top = tasks.filter((t) => !t.parentId);
    await expect(page.locator(".kb-board .kb-card")).toHaveCount(top.length);

    // Cards with a due date show a pill; cards with links/files show their counts.
    const dated = top.filter((t) => t.dueDate);
    await expect(page.locator(".kb-board .kb-card .kb-pill.due")).toHaveCount(dated.length);
    await expect(page.locator(".kb-board .kb-card .kb-pill.links")).toHaveCount(top.filter((t) => (t.links || []).length).length);
    await expect(page.locator(".kb-board .kb-card .kb-pill.files")).toHaveCount(top.filter((t) => (t.attachments || []).length).length);
    const today = new Date().toLocaleDateString("en-CA");
    const overdue = top.filter((t) => t.dueDate && t.dueDate < today && t.status !== "done" && !t.done);
    await expect(page.locator(".kb-board .kb-card.overdue")).toHaveCount(overdue.length);
    await expect(page.locator(".toast")).toHaveCount(0);

    await page.screenshot({ path: path.join(shots, `${slug(project.name)}-board.png`), fullPage: false });

    // Open the first task that has native checklists and confirm they render.
    const rich = top.find((t) => (t.acceptance || []).length && (t.subtasks || []).length);
    if (rich) {
      await page.locator(".kb-card", { has: page.locator(".kb-card-title", { hasText: rich.title.slice(0, 30) }) }).first().locator(".kb-card-main").click();
      await expect(page.locator(".kb-detail")).toBeVisible();
      const values = await page.locator(".task-row input.task-title").evaluateAll((els) => els.map((e) => (e as HTMLInputElement).value));
      for (const item of [...rich.acceptance, ...rich.subtasks]) expect(values).toContain(item.title);
      if (rich.dueDate) await expect(page.locator(".kb-due-field input[type=date]")).toHaveValue(rich.dueDate);
      for (const l of rich.links || []) await expect(page.locator(".kb-link", { hasText: l.title || l.url }).first()).toBeVisible();
      await page.screenshot({ path: path.join(shots, `${slug(project.name)}-task.png`) });
      await page.getByRole("button", { name: "Cancel" }).click(); // never save during a smoke test
    }
    expect(errors).toEqual([]);
  });
}

test("filters work on real data: this year / quarter / past due", async ({ page }) => {
  // Use whichever project has the most dated top-level tasks.
  const dated = (p: any) => (p.tasks || []).filter((t: any) => !t.parentId && t.dueDate).length;
  const project = [...projects].sort((a, b) => dated(b) - dated(a))[0];
  test.skip(!project || dated(project) === 0, "no project has due dates");
  await page.goto("/");
  await page.addStyleTag({ content: ".ad-overlay{display:none !important}" });
  await page.locator(".sidebar").getByText(project.name, { exact: true }).first().click();
  await page.getByRole("button", { name: /^Tasks/ }).first().click();
  await page.getByRole("button", { name: /^All tasks/ }).click();
  await page.locator(".kb-filter-toggle").click();
  const year = new Date().getFullYear();
  const top: any[] = project.tasks.filter((t: any) => !t.parentId);
  await page.locator(".kb-filter-due select").selectOption("this_year");
  const inYear = top.filter((t) => t.dueDate && t.dueDate.startsWith(String(year)));
  await expect(page.locator(".kb-board .kb-card")).toHaveCount(inYear.length);
  await page.locator(".kb-filter-due select").selectOption("q4"); // Oct-Dec by default
  const inQ4 = top.filter((t) => t.dueDate && t.dueDate >= `${year}-10-01` && t.dueDate <= `${year}-12-31`);
  await expect(page.locator(".kb-board .kb-card")).toHaveCount(inQ4.length);
  await page.screenshot({ path: path.join(shots, "filter-q4.png") });
});
