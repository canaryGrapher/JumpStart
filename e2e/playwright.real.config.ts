import { defineConfig } from "@playwright/test";
// Read-only smoke test against a COPY of the real data. See scripts/stack.sh (E2E_COPY_REAL=1).
export default defineConfig({
  testDir: "./tests-real",
  workers: 1,
  timeout: 45_000,
  expect: { timeout: 8_000 },
  reporter: [["list"]],
  outputDir: "artifacts/test-results-real",
  use: { baseURL: "http://localhost:34115", channel: "chrome", viewport: { width: 1500, height: 1100 }, screenshot: "only-on-failure" },
});
