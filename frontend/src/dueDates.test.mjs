// Run with: node --test frontend/src/dueDates.test.mjs
import test from "node:test";
import assert from "node:assert/strict";
import {
  EMPTY_FILTERS,
  activeFilterCount,
  daysUntil,
  dueSummary,
  formatMD,
  isPastDue,
  isValidDate,
  matchesFilters,
  parseMD,
  queryFromFilters,
  filtersFromQuery,
  sameFilters,
} from "./dueDates.js";

const TODAY = "2026-10-07";
const f = (patch) => ({ ...EMPTY_FILTERS, ...patch });

test("isValidDate rejects impossible and malformed dates", () => {
  assert.equal(isValidDate("2026-10-07"), true);
  assert.equal(isValidDate("2028-02-29"), true);
  assert.equal(isValidDate("2026-02-30"), false);
  assert.equal(isValidDate("10/07/2026"), false);
  assert.equal(isValidDate(""), false);
  assert.equal(isValidDate(undefined), false);
});

test("past due: before today, not done; due today is not past due", () => {
  assert.equal(isPastDue({ dueDate: "2026-10-06", status: "todo" }, TODAY), true);
  assert.equal(isPastDue({ dueDate: "2026-10-07", status: "todo" }, TODAY), false);
  assert.equal(isPastDue({ dueDate: "2026-10-06", status: "done" }, TODAY), false);
  assert.equal(isPastDue({ dueDate: "2026-10-06", done: true }, TODAY), false);
  assert.equal(isPastDue({ status: "todo" }, TODAY), false);
  assert.equal(isPastDue({ dueDate: "garbage", status: "todo" }, TODAY), false);
});

test("daysUntil and dueSummary", () => {
  assert.equal(daysUntil("2026-10-10", TODAY), 3);
  assert.equal(daysUntil("2026-10-05", TODAY), -2);
  assert.equal(daysUntil("2026-11-01", "2026-10-31"), 1); // month boundary
  assert.equal(dueSummary({ dueDate: "2026-10-05", status: "todo" }, TODAY), "Past due by 2 days");
  assert.equal(dueSummary({ dueDate: "2026-10-06", status: "todo" }, TODAY), "Past due by 1 day");
  assert.equal(dueSummary({ dueDate: "2026-10-07", status: "todo" }, TODAY), "Due today");
  assert.equal(dueSummary({ dueDate: "2026-10-08", status: "todo" }, TODAY), "Due in 1 day");
  assert.match(dueSummary({ dueDate: "2026-10-05", status: "done" }, TODAY), /^Was due/);
  assert.equal(dueSummary({ status: "todo" }, TODAY), "");
});

test("month-day helpers round-trip", () => {
  assert.deepEqual(parseMD("07-01"), { month: 7, day: 1 });
  assert.equal(formatMD({ month: 2, day: 9 }), "02-09");
  assert.deepEqual(parseMD("nonsense"), { month: 1, day: 1 });
});

const tasks = [
  { id: "1", status: "todo", priority: "high", type: "bug", sprintId: "s1", assignee: "Alex, Sam", labels: ["Stripe", "urgent"], dueDate: "2026-10-01", acceptance: [{}], subtasks: [] },
  { id: "2", status: "done", priority: "", type: "task", sprintId: "", assignee: "", labels: [], dueDate: "2026-09-01", acceptance: [], subtasks: [{}] },
  { id: "3", status: "inprogress", priority: "medium", type: "story", sprintId: "s1", assignee: "sam", labels: ["stripe"], dueDate: "2026-10-09", acceptance: [], subtasks: [] },
  { id: "4", status: "backlog", priority: "low", type: "task", sprintId: "s2", assignee: "Ada", labels: [], dueDate: "", acceptance: [], subtasks: [] },
];
const ids = (filters, range = null) =>
  tasks.filter((t) => matchesFilters(t, filters, range, TODAY)).map((t) => t.id).join(",");

test("no filters matches everything", () => {
  assert.equal(ids(EMPTY_FILTERS), "1,2,3,4");
  assert.equal(activeFilterCount(EMPTY_FILTERS), 0);
});

test("status, priority (including none), type, sprint", () => {
  assert.equal(ids(f({ statuses: ["todo", "inprogress"] })), "1,3");
  assert.equal(ids(f({ priorities: ["high", "none"] })), "1,2");
  assert.equal(ids(f({ types: ["task"] })), "2,4");
  assert.equal(ids(f({ sprints: ["s1"] })), "1,3");
  assert.equal(ids(f({ sprints: [""] })), "2");
  assert.equal(ids(f({ sprints: ["s2", ""] })), "2,4");
});

test("assignees are case-insensitive tokens; Unassigned works", () => {
  assert.equal(ids(f({ assignees: ["sam"] })), "1,3");
  assert.equal(ids(f({ assignees: ["ALEX"] })), "1");
  assert.equal(ids(f({ assignees: ["__none__"] })), "2");
  assert.equal(ids(f({ assignees: ["ada", "__none__"] })), "2,4");
});

test("labels match any, case-insensitively", () => {
  assert.equal(ids(f({ labels: ["STRIPE"] })), "1,3");
  assert.equal(ids(f({ labels: ["urgent", "missing"] })), "1");
});

test("acceptance and subtasks: has / none", () => {
  assert.equal(ids(f({ acceptance: "has" })), "1");
  assert.equal(ids(f({ acceptance: "none" })), "2,3,4");
  assert.equal(ids(f({ subtasks: "has" })), "2");
  assert.equal(ids(f({ subtasks: "none" })), "1,3,4");
});

test("due-date flags and ranges", () => {
  assert.equal(ids(f({ overdue: true })), "1");
  assert.equal(ids(f({ noDueDate: true })), "4");
  assert.equal(ids(EMPTY_FILTERS, { from: "2026-10-05", to: "2026-10-11" }), "3");
  assert.equal(ids(EMPTY_FILTERS, { from: "2026-10-01", to: "" }), "1,3");
  assert.equal(ids(EMPTY_FILTERS, { from: "", to: "2026-09-30" }), "2");
  // Range endpoints are inclusive, and undated tasks never match a range.
  assert.equal(ids(EMPTY_FILTERS, { from: "2026-10-09", to: "2026-10-09" }), "3");
});

test("filters combine with AND across groups", () => {
  assert.equal(ids(f({ statuses: ["todo"], labels: ["stripe"], overdue: true })), "1");
  assert.equal(ids(f({ statuses: ["done"], overdue: true })), "");
});

test("activeFilterCount counts groups, not selections", () => {
  assert.equal(activeFilterCount(f({ statuses: ["a", "b", "c"] })), 1);
  assert.equal(activeFilterCount(f({ duePreset: "q1", overdue: true, labels: ["x"] })), 3);
  assert.equal(activeFilterCount(f({ dueFrom: "2026-01-01", dueTo: "2026-02-01" })), 1);
});

test("saved-filter query round-trips through the panel state", () => {
  const panel = f({ priorities: ["high"], types: ["bug"], duePreset: "this_quarter", overdue: true });
  const q = queryFromFilters(panel);
  assert.deepEqual(q, { priorities: ["high"], types: ["bug"], duePreset: "this_quarter", overdue: true });
  assert.deepEqual(filtersFromQuery(q), panel);
  assert.deepEqual(queryFromFilters(EMPTY_FILTERS), {});
  assert.deepEqual(filtersFromQuery(undefined), EMPTY_FILTERS);
  // Unknown keys from a newer build are ignored rather than leaking into the panel.
  assert.deepEqual(filtersFromQuery({ statuses: ["todo"], somethingNew: 1 }), f({ statuses: ["todo"] }));
});

test("sameFilters ignores key order and empty values", () => {
  assert.ok(sameFilters({ types: ["bug"], priorities: ["high"] }, f({ priorities: ["high"], types: ["bug"] })));
  assert.ok(sameFilters({}, EMPTY_FILTERS));
  assert.ok(!sameFilters({ types: ["bug"] }, { types: ["task"] }));
});
