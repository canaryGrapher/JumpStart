import { defineConfig } from "@playwright/test";

// The suite drives the real Go backend (wails dev) through a browser, with
// HOME pointed at a throwaway directory. See scripts/run-e2e.sh.
export default defineConfig({
  testDir: "./tests",
  // One backend and one data directory are shared, so tests run serially.
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 45_000,
  expect: { timeout: 8_000 },
  reporter: [["list"], ["html", { outputFolder: "playwright-report", open: "never" }]],
  outputDir: "artifacts/test-results",
  use: {
    baseURL: process.env.BASE_URL || "http://localhost:34115",
    channel: "chrome",
    viewport: { width: 1500, height: 1100 },
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
    video: "off",
  },
});
