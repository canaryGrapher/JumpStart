import fs from "node:fs";
import path from "node:path";
import { expect, type Page } from "@playwright/test";
// @ts-ignore - plain ESM script shared with the shell runner
import { seed, fixtureDates, MCP_PORT, MCP_TOKEN, PROJECT_ID } from "../scripts/seed.mjs";

export const HOME = process.env.E2E_HOME || "/tmp/js-e2e-home";
export const DATA_DIR = path.join(HOME, ".jumpstart");
export { MCP_PORT, MCP_TOKEN, PROJECT_ID };

export type Dates = ReturnType<typeof fixtureDates>;

/** Rewrite the seed data. The backend re-reads config.json on every call. */
export function reseed(opts: { legacy?: boolean } = {}) {
  return seed(HOME, opts) as { dir: string; root: string; dates: Dates };
}

export function readConfig(): any[] {
  return JSON.parse(fs.readFileSync(path.join(DATA_DIR, "config.json"), "utf8"));
}
export function readTask(id: string): any {
  const p = readConfig().find((x) => x.id === PROJECT_ID);
  return p.tasks.find((t: any) => t.id === id);
}
export function readProject(): any {
  return readConfig().find((x) => x.id === PROJECT_ID);
}

// ---- date helpers, deliberately independent of the app's own code ----
const pad = (n: number) => String(n).padStart(2, "0");
export const fmt = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
export const ymd = (s: string) => {
  const [y, m, d] = s.split("-").map(Number);
  return new Date(y, m - 1, d);
};
export function expectedRange(preset: string, todayStr: string, quarters?: [string, string][]) {
  const today = ymd(todayStr);
  const y = today.getFullYear();
  const monday = new Date(y, today.getMonth(), today.getDate() - ((today.getDay() + 6) % 7));
  const add = (d: Date, n: number) => new Date(d.getFullYear(), d.getMonth(), d.getDate() + n);
  switch (preset) {
    case "last_week": return [fmt(add(monday, -7)), fmt(add(monday, -1))];
    case "this_week": return [fmt(monday), fmt(add(monday, 6))];
    case "next_week": return [fmt(add(monday, 7)), fmt(add(monday, 13))];
    case "this_month": return [fmt(new Date(y, today.getMonth(), 1)), fmt(new Date(y, today.getMonth() + 1, 0))];
    case "next_month": return [fmt(new Date(y, today.getMonth() + 1, 1)), fmt(new Date(y, today.getMonth() + 2, 0))];
    case "this_year": return [`${y}-01-01`, `${y}-12-31`];
    case "today": return [todayStr, todayStr];
    case "tomorrow": { const t = fmt(add(today, 1)); return [t, t]; }
    case "this_quarter":
    case "next_quarter":
    case "last_quarter": {
      // Find the quarter containing a day, using the q1..q4 logic below for that day.
      const containing = (dayStr: string): string[] => {
        for (const ref of [dayStr]) {
          for (const qid of ["q1", "q2", "q3", "q4"]) {
            const r = expectedRange(qid, ref, quarters);
            if (dayStr >= r[0] && dayStr <= r[1]) return r;
          }
          const prev = fmt(new Date(ymd(dayStr).getFullYear() - 1, ymd(dayStr).getMonth(), ymd(dayStr).getDate()));
          for (const qid of ["q1", "q2", "q3", "q4"]) {
            const r = expectedRange(qid, prev, quarters);
            if (dayStr >= r[0] && dayStr <= r[1]) return r;
          }
        }
        throw new Error("no quarter contains " + dayStr);
      };
      const cur = containing(todayStr);
      if (preset === "this_quarter") return cur;
      if (preset === "next_quarter") return containing(fmt(add(ymd(cur[1]), 1)));
      return containing(fmt(add(ymd(cur[0]), -1)));
    }
  }
  const q = ["q1", "q2", "q3", "q4"].indexOf(preset);
  if (q >= 0) {
    const qs = quarters ?? [["01-01", "03-31"], ["04-01", "06-30"], ["07-01", "09-30"], ["10-01", "12-31"]];
    const [q1m, q1d] = qs[0][0].split("-").map(Number);
    let fy = y;
    if (today < new Date(y, q1m - 1, q1d)) fy--;
    const [sm, sd] = qs[q][0].split("-").map(Number);
    const [em, ed] = qs[q][1].split("-").map(Number);
    const startYear = sm < q1m || (sm === q1m && sd < q1d) ? fy + 1 : fy;
    const endYear = em < sm || (em === sm && ed < sd) ? startYear + 1 : startYear;
    return [fmt(new Date(startYear, sm - 1, sd)), fmt(new Date(endYear, em - 1, ed))];
  }
  throw new Error("unknown preset " + preset);
}
export const inRange = (due: string | undefined, [from, to]: string[]) =>
  !!due && due >= from && due <= to;

// ---- MCP client (streamable HTTP, JSON-RPC) ----
export class Mcp {
  private id = 1;
  private session = "";
  static async connect() {
    const c = new Mcp();
    await c.rpc("initialize", {
      protocolVersion: "2025-03-26",
      capabilities: {},
      clientInfo: { name: "e2e", version: "0" },
    });
    await c.notify("notifications/initialized");
    return c;
  }
  private async post(body: any) {
    const res = await fetch(`http://127.0.0.1:${MCP_PORT}/mcp`, {
      method: "POST",
      headers: {
        "content-type": "application/json",
        accept: "application/json, text/event-stream",
        authorization: `Bearer ${MCP_TOKEN}`,
        ...(this.session ? { "mcp-session-id": this.session } : {}),
      },
      body: JSON.stringify(body),
    });
    const sid = res.headers.get("mcp-session-id");
    if (sid) this.session = sid;
    const text = await res.text();
    return { status: res.status, text, type: res.headers.get("content-type") || "" };
  }
  private parse(text: string, type: string) {
    if (type.includes("text/event-stream")) {
      const data = text.split("\n").filter((l) => l.startsWith("data:")).map((l) => l.slice(5).trim());
      return JSON.parse(data[data.length - 1]);
    }
    return text ? JSON.parse(text) : {};
  }
  async notify(method: string) {
    await this.post({ jsonrpc: "2.0", method });
  }
  async rpc(method: string, params: any) {
    const r = await this.post({ jsonrpc: "2.0", id: this.id++, method, params });
    if (r.status >= 400) throw new Error(`MCP ${method} -> HTTP ${r.status}: ${r.text}`);
    const msg = this.parse(r.text, r.type);
    if (msg.error) throw new Error(`MCP ${method} error: ${JSON.stringify(msg.error)}`);
    return msg.result;
  }
  async tools() {
    return (await this.rpc("tools/list", {})).tools as any[];
  }
  /** Call a tool; returns { isError, text, content }. */
  async call(name: string, args: Record<string, unknown> = {}) {
    const res = await this.rpc("tools/call", { name, arguments: args });
    const text = (res.content || []).filter((c: any) => c.type === "text").map((c: any) => c.text).join("\n");
    return { isError: !!res.isError, text, content: res.content as any[] };
  }
  async json(name: string, args: Record<string, unknown> = {}) {
    const r = await this.call(name, args);
    if (r.isError) throw new Error(`${name}: ${r.text}`);
    return JSON.parse(r.text);
  }
}

// ---- UI helpers ----

/** Open the app, select the E2E project's Tasks tab, show every sprint. */
export async function openBoard(page: Page, { allSprints = true } = {}) {
  const errors: string[] = [];
  page.on("pageerror", (e) => {
    // Wails' own browser bridge throws this in dev mode; not app code.
    if (!/reading 'nodes'/.test(e.message)) errors.push(e.message);
  });
  await page.goto("/");
  // The app fetches a remote announcement banner that can cover buttons; it
  // is unrelated to what is under test.
  await page.addStyleTag({ content: ".ad-overlay{display:none !important}" });
  await page.getByText("E2E Board").first().click();
  await page.getByRole("button", { name: /^Tasks/ }).first().click();
  // The project may open in its remembered Sheet view instead of the board.
  await expect(page.locator(".kb-board, .sheet-table").first()).toBeVisible();
  if (allSprints) {
    await page.getByRole("button", { name: /^All tasks/ }).click();
  }
  return { errors };
}

export const card = (page: Page, title: string) =>
  page.locator(".kb-card", { has: page.locator(".kb-card-title", { hasText: title }) });

export async function visibleTitles(page: Page): Promise<string[]> {
  const titles = await page.locator(".kb-board .kb-card > .kb-card-main .kb-card-title").allInnerTexts();
  return titles.map((t) => t.replace(/^[▣○▲]\s*/, "").trim());
}

export async function openFilters(page: Page) {
  if (!(await page.locator(".kb-filters").isVisible())) {
    await page.locator(".kb-filter-toggle").click();
  }
  await expect(page.locator(".kb-filters")).toBeVisible();
}

export async function openTask(page: Page, title: string) {
  await card(page, title).locator(".kb-card-main").first().click();
  await expect(page.locator(".kb-detail")).toBeVisible();
}

export async function chip(page: Page, group: string, label: string) {
  return page
    .locator(".kb-filter-group", { has: page.locator(".kb-filter-label", { hasText: new RegExp(`^${group}$`) }) })
    .locator(".kb-chip", { hasText: new RegExp(`^${label}$`) });
}
