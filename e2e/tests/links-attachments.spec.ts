import { test, expect, type Page } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";
import { reseed, openBoard, openTask, card, readTask, DATA_DIR, PROJECT_ID } from "./helpers";

const LAUNCH_LOG = "/tmp/js-e2e-launch.log";
const launches = () =>
  fs.existsSync(LAUNCH_LOG) ? fs.readFileSync(LAUNCH_LOG, "utf8").split("\n").filter(Boolean) : [];
const taskFilesDir = (taskId: string) => path.join(DATA_DIR, "attachments", PROJECT_ID, taskId);
const filesOnDisk = (taskId: string) =>
  fs.existsSync(taskFilesDir(taskId)) ? fs.readdirSync(taskFilesDir(taskId)) : [];
const trashFiles = () => {
  const root = path.join(DATA_DIR, "attachments", ".trash");
  const out: string[] = [];
  const walk = (d: string) => {
    if (!fs.existsSync(d)) return;
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      e.isDirectory() ? walk(path.join(d, e.name)) : out.push(path.join(d, e.name));
    }
  };
  walk(root);
  return out;
};

const PNG_B64 =
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==";

/** Simulate dropping files onto the attachments field (what dragging from Finder does). */
async function dropFiles(page: Page, files: { name: string; type: string; text?: string; b64?: string }[]) {
  await page.evaluate((files) => {
    const dt = new DataTransfer();
    for (const f of files) {
      const bytes = f.b64 ? Uint8Array.from(atob(f.b64), (c) => c.charCodeAt(0)) : new TextEncoder().encode(f.text ?? "");
      dt.items.add(new File([bytes], f.name, { type: f.type }));
    }
    const target = document.querySelector(".kb-attachments")!;
    target.dispatchEvent(new DragEvent("dragover", { dataTransfer: dt, bubbles: true, cancelable: true }));
    target.dispatchEvent(new DragEvent("drop", { dataTransfer: dt, bubbles: true, cancelable: true }));
  }, files);
}

async function pasteImage(page: Page) {
  await page.evaluate((b64) => {
    const dt = new DataTransfer();
    dt.items.add(new File([Uint8Array.from(atob(b64), (c) => c.charCodeAt(0))], "image.png", { type: "image/png" }));
    document.querySelector(".kb-attachments")!.dispatchEvent(
      new ClipboardEvent("paste", { clipboardData: dt, bubbles: true, cancelable: true })
    );
  }, PNG_B64);
}

test.beforeEach(() => {
  reseed();
  fs.writeFileSync(LAUNCH_LOG, "");
  fs.rmSync(path.join(DATA_DIR, "attachments"), { recursive: true, force: true });
});

test.describe("links", () => {
  test("add a link with and without a label, persist, show a count on the card", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Due today task");
    const links = page.locator(".field", { has: page.getByText("Links", { exact: true }) });
    await links.getByPlaceholder("https://…").fill("example.com/runbook");
    await links.getByPlaceholder("Label (optional)").fill("Runbook");
    await links.getByRole("button", { name: "Add" }).click();
    await links.getByPlaceholder("https://…").fill("https://github.com/org/repo/issues/7");
    await links.getByRole("button", { name: "Add" }).click();

    await expect(links.locator(".kb-link")).toHaveCount(2);
    await expect(links.locator(".kb-link").first()).toContainText("Runbook");
    await expect(links.locator(".kb-link").first()).toContainText("https://example.com/runbook"); // https:// added
    await page.getByRole("button", { name: "Save", exact: true }).click();

    expect(readTask("t-today").links).toMatchObject([
      { title: "Runbook", url: "https://example.com/runbook" },
      { url: "https://github.com/org/repo/issues/7" },
    ]);
    await expect(card(page, "Due today task").locator(".kb-pill.links")).toHaveText("🔗 2");
  });

  test("unsafe or malformed links are refused with a message", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Due today task");
    const links = page.locator(".field", { has: page.getByText("Links", { exact: true }) });
    for (const bad of ["javascript:alert(1)", "file:///etc/passwd", "data:text/html,<script>", "ftp://example.com", "not a url", ""]) {
      await links.getByPlaceholder("https://…").fill(bad);
      await links.getByRole("button", { name: "Add" }).click();
      await expect(page.locator(".toast")).toContainText("web address");
      await expect(links.locator(".kb-link")).toHaveCount(0);
    }
  });

  test("clicking a link asks the OS to open it in the browser (and only that URL)", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Due today task");
    const links = page.locator(".field", { has: page.getByText("Links", { exact: true }) });
    await links.getByPlaceholder("https://…").fill("https://example.com/jumpstart-e2e");
    await links.getByRole("button", { name: "Add" }).click();
    await links.locator(".kb-link").click();
    await expect.poll(launches).toContain("open https://example.com/jumpstart-e2e");
  });

  test("the backend refuses unsafe URLs even if the UI check is bypassed", async ({ page }) => {
    await openBoard(page);
    const results = await page.evaluate(async () => {
      const app = (window as any).go.main.App;
      const out: Record<string, string> = {};
      for (const u of ["javascript:alert(1)", "file:///etc/hosts", "ftp://x.io", "http://"]) {
        try { await app.OpenTaskLink(u); out[u] = "opened"; } catch (e) { out[u] = String(e); }
      }
      return out;
    });
    for (const v of Object.values(results)) expect(v).not.toBe("opened");
    expect(launches().filter((l) => /javascript|file:|ftp:/.test(l))).toEqual([]);
  });

  test("remove a link", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Due today task");
    const links = page.locator(".field", { has: page.getByText("Links", { exact: true }) });
    await links.getByPlaceholder("https://…").fill("https://example.com/a");
    await links.getByRole("button", { name: "Add" }).click();
    await links.getByRole("button", { name: "Remove" }).click();
    await expect(links.locator(".kb-link")).toHaveCount(0);
    await page.getByRole("button", { name: "Save", exact: true }).click();
    expect(readTask("t-today").links ?? []).toEqual([]);
  });
});

test.describe("attachments", () => {
  const field = (page: Page) => page.locator(".kb-attachments");

  test("drop a text file and an image: rows, thumbnail, size; Save persists and card shows a count", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Due today task");
    await dropFiles(page, [
      { name: "notes.txt", type: "text/plain", text: "Deploy on Friday" },
      { name: "pic.png", type: "image/png", b64: PNG_B64 },
    ]);
    await expect(field(page).locator(".kb-attachment")).toHaveCount(2);
    await expect(field(page).locator(".kb-attachment-name").first()).toHaveText("notes.txt");
    await expect(field(page).locator(".kb-attachment-thumb img")).toHaveCount(1); // image gets a real thumbnail
    await expect(field(page).locator(".kb-attachment-thumb img")).toHaveAttribute("src", /^data:image\/png;base64,/);
    expect(filesOnDisk("t-today")).toHaveLength(2); // already stored, not yet attached to the task
    expect(readTask("t-today").attachments ?? []).toEqual([]);

    await page.getByRole("button", { name: "Save", exact: true }).click();
    const saved = readTask("t-today").attachments;
    expect(saved).toHaveLength(2);
    expect(saved[0]).toMatchObject({ name: "notes.txt", mime: "text/plain", size: 16 });
    expect(saved[1]).toMatchObject({ name: "pic.png", mime: "image/png" });
    expect(filesOnDisk("t-today")).toContain(saved[0].file);
    await expect(card(page, "Due today task").locator(".kb-pill.files")).toHaveText("📎 2");

    // Survives a reload and the thumbnail comes back.
    await page.reload();
    await openBoard(page);
    await openTask(page, "Due today task");
    await expect(field(page).locator(".kb-attachment-thumb img")).toHaveCount(1);
  });

  test("paste a screenshot gets a readable name", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Due today task");
    await pasteImage(page);
    await expect(field(page).locator(".kb-attachment")).toHaveCount(1);
    await expect(field(page).locator(".kb-attachment-name")).toHaveText(/^pasted-image-\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2}\.png$/);
  });

  test("clicking an image opens the in-app viewer; Escape closes it", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Due today task");
    await dropFiles(page, [{ name: "pic.png", type: "image/png", b64: PNG_B64 }]);
    await expect(field(page).locator(".kb-attachment-thumb img")).toHaveCount(1);
    await field(page).locator(".kb-attachment-main").click();
    const viewer = page.locator(".kb-viewer");
    await expect(viewer).toBeVisible();
    await expect(viewer.locator("img")).toHaveAttribute("src", /^data:image\/png/);
    await expect(viewer.locator(".kb-viewer-name")).toHaveText("pic.png");
    await expect(viewer.getByRole("button", { name: "Preview" })).toBeVisible();
    await expect(viewer.getByRole("button", { name: "Open" })).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(viewer).toHaveCount(0);
    await expect(page.locator(".kb-detail")).toBeVisible(); // the task modal stays open underneath
  });

  test("clicking a non-image file opens it; Preview, Open and Reveal run the right OS commands", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Due today task");
    await dropFiles(page, [
      { name: "notes.txt", type: "text/plain", text: "hello" },
      { name: "pic.png", type: "image/png", b64: PNG_B64 },
    ]);
    await expect(field(page).locator(".kb-attachment")).toHaveCount(2);
    await page.getByRole("button", { name: "Save", exact: true }).click();
    const [txt, png] = readTask("t-today").attachments;
    const stored = (a: any) => path.join(taskFilesDir("t-today"), a.file);
    await openTask(page, "Due today task");

    const row = (name: string) => field(page).locator(".kb-attachment", { hasText: name });
    // Click on a plain file = open with the default app.
    await row("notes.txt").locator(".kb-attachment-main").click();
    await expect.poll(launches).toContain(`open ${stored(txt)}`);
    // Preview: Quick Look for generic files, Preview.app for images.
    await row("notes.txt").getByRole("button", { name: "Preview" }).click();
    await expect.poll(launches).toContain(`qlmanage -p ${stored(txt)}`);
    await row("pic.png").getByRole("button", { name: "Preview" }).click();
    await expect.poll(launches).toContain(`open -a Preview ${stored(png)}`);
    // Open and Reveal buttons.
    await row("pic.png").getByRole("button", { name: "Open", exact: true }).click();
    await expect.poll(launches).toContain(`open ${stored(png)}`);
    await row("notes.txt").getByRole("button", { name: "Reveal" }).click();
    await expect.poll(launches).toContain(`open -R ${stored(txt)}`);
    await expect(page.locator(".toast")).toHaveCount(0); // no error surfaced
  });

  test("Cancel discards files uploaded in that session", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Due today task");
    await dropFiles(page, [{ name: "temp.txt", type: "text/plain", text: "x" }]);
    await expect(field(page).locator(".kb-attachment")).toHaveCount(1);
    expect(filesOnDisk("t-today")).toHaveLength(1);
    await page.getByRole("button", { name: "Cancel" }).click();
    await expect.poll(() => filesOnDisk("t-today")).toEqual([]);
    expect(readTask("t-today").attachments ?? []).toEqual([]);
  });

  test("removing a just-added file deletes it immediately; removing a saved file trashes it on Save", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Due today task");
    await dropFiles(page, [{ name: "keep.txt", type: "text/plain", text: "keep me" }]);
    await page.getByRole("button", { name: "Save", exact: true }).click();
    const [kept] = readTask("t-today").attachments;

    await openTask(page, "Due today task");
    await dropFiles(page, [{ name: "scratch.txt", type: "text/plain", text: "oops" }]);
    await expect(field(page).locator(".kb-attachment")).toHaveCount(2);
    await field(page).locator(".kb-attachment", { hasText: "scratch.txt" }).getByRole("button", { name: "Remove" }).click();
    await expect.poll(() => filesOnDisk("t-today")).toEqual([kept.file]); // scratch gone, kept stays

    await field(page).locator(".kb-attachment", { hasText: "keep.txt" }).getByRole("button", { name: "Remove" }).click();
    expect(filesOnDisk("t-today")).toContain(kept.file); // still there until saved
    await page.getByRole("button", { name: "Save", exact: true }).click();
    await expect.poll(() => filesOnDisk("t-today")).toEqual([]);
    const trashed = trashFiles();
    expect(trashed.some((f) => path.basename(f) === kept.file)).toBe(true); // recoverable, not destroyed
    expect(fs.readFileSync(trashed.find((f) => path.basename(f) === kept.file)!, "utf8")).toBe("keep me");
  });

  test("deleting a task trashes its attachments", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Due today task");
    await dropFiles(page, [{ name: "doomed.txt", type: "text/plain", text: "bye" }]);
    await page.getByRole("button", { name: "Save", exact: true }).click();
    const [att] = readTask("t-today").attachments;
    await openTask(page, "Due today task");
    await page.getByRole("button", { name: /^Delete task/ }).click();
    const confirm = page.getByRole("button", { name: "Delete", exact: true });
    if (await confirm.count()) await confirm.click();
    await expect(card(page, "Due today task")).toHaveCount(0);
    await expect.poll(() => filesOnDisk("t-today")).toEqual([]);
    expect(trashFiles().some((f) => path.basename(f) === att.file)).toBe(true);
  });

  test("hostile file names cannot escape the attachment folder", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Due today task");
    await dropFiles(page, [{ name: "../../../evil.txt", type: "text/plain", text: "x" }]);
    await expect(field(page).locator(".kb-attachment")).toHaveCount(1);
    await page.getByRole("button", { name: "Save", exact: true }).click();
    const [att] = readTask("t-today").attachments;
    expect(att.name).toBe("evil.txt");
    expect(att.file).toMatch(/^[A-Za-z0-9_-]+\.txt$/);
    expect(fs.existsSync(path.join(path.dirname(DATA_DIR), "evil.txt"))).toBe(false);
    expect(fs.existsSync(path.join(DATA_DIR, "evil.txt"))).toBe(false);
  });

  test("the backend refuses to open or read a file name it did not generate", async ({ page }) => {
    await openBoard(page);
    const out = await page.evaluate(async (pid) => {
      const app = (window as any).go.main.App;
      const evil = { id: "x", name: "passwd", file: "../../../../../etc/passwd", mime: "text/plain", size: 1, addedAt: 0 };
      const r: Record<string, string> = {};
      for (const [k, fn] of Object.entries({
        open: () => app.OpenTaskAttachment(pid, "t-today", evil),
        preview: () => app.TaskAttachmentPreview(pid, "t-today", { ...evil, mime: "image/png" }),
        discard: () => app.DiscardTaskAttachment(pid, "t-today", evil),
        badTask: () => app.OpenTaskAttachment(pid, "../t", { ...evil, file: "abc.txt" }),
      })) {
        try { await (fn as any)(); r[k] = "ALLOWED"; } catch (e) { r[k] = String(e); }
      }
      return r;
    }, PROJECT_ID);
    for (const [k, v] of Object.entries(out)) expect(v, k).not.toBe("ALLOWED");
    expect(launches().filter((l) => l.includes("passwd"))).toEqual([]);
  });

  test("DiscardTaskAttachment refuses to delete a file the saved task still uses", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Due today task");
    await dropFiles(page, [{ name: "saved.txt", type: "text/plain", text: "important" }]);
    await page.getByRole("button", { name: "Save", exact: true }).click();
    const [att] = readTask("t-today").attachments;
    const result = await page.evaluate(async ([pid, a]) => {
      try { await (window as any).go.main.App.DiscardTaskAttachment(pid, "t-today", a); return "discarded"; }
      catch (e) { return String(e); }
    }, [PROJECT_ID, att] as const);
    expect(result).toContain("saved on the task");
    expect(filesOnDisk("t-today")).toContain(att.file);
  });

  test("a file over 100 MB is rejected before upload", async ({ page }) => {
    await openBoard(page);
    await openTask(page, "Due today task");
    await page.evaluate(() => {
      const dt = new DataTransfer();
      dt.items.add(new File([new Uint8Array(101 * 1024 * 1024)], "huge.bin", { type: "application/octet-stream" }));
      document.querySelector(".kb-attachments")!.dispatchEvent(
        new DragEvent("drop", { dataTransfer: dt, bubbles: true, cancelable: true })
      );
    });
    await expect(page.locator(".toast")).toContainText("larger than 100 MB");
    await expect(field(page).locator(".kb-attachment")).toHaveCount(0);
    expect(filesOnDisk("t-today")).toEqual([]);
  });
});
