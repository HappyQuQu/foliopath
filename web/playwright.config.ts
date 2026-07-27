import { defineConfig, devices } from "@playwright/test";

const baseURL = process.env.FOLIOPATH_WEB_E2E_URL ?? "http://127.0.0.1:4174";

export default defineConfig({
  testDir: "./tests/e2e",
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: process.env.CI ? [["line"], ["html", { open: "never" }]] : "line",
  use: {
    ...devices["Desktop Chrome"],
    baseURL,
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
    video: "retain-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
    {
      name: "mobile-chromium",
      testMatch: /media-matrix\.spec\.ts/,
      use: { ...devices["Pixel 5"] },
    },
  ],
  outputDir: "../test-results/playwright",
});
