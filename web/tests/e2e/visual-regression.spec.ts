import { expect, test, type Page } from "@playwright/test";

const libraryId = "lib_matrix";

test("offline viewer keeps the approved dark desktop composition", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "visual-chromium");
  test.skip(process.platform !== "linux", "The approved visual baseline is Linux-owned.");

  await page.addInitScript(() => {
    window.localStorage.setItem(
      "foliopath.preferences.v1",
      JSON.stringify({ locale: "en", theme: "dark" }),
    );
  });
  await mockOfflineViewerBackend(page);

  await page.goto(`/libraries/${libraryId}/media/offline`);
  await expect(
    page.getByRole("heading", { name: "Library is offline" }),
  ).toBeVisible();
  await expect(page.getByRole("button", { name: "Close" })).toBeFocused();
  await expect(page).toHaveScreenshot("offline-viewer-dark.png", {
    animations: "disabled",
    caret: "hide",
    fullPage: false,
  });
});

async function mockOfflineViewerBackend(page: Page) {
  await page.route("**/health/ready", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ status: "ready" }),
    });
  });
  await page.route("**/api/v1/auth/session", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        administrator: {
          displayName: "Visual regression administrator",
          id: "usr_visual",
          username: "visual-admin",
        },
        csrfToken: "visual-csrf-token-0000000000000000",
        expiresAt: "2030-01-01T00:00:00Z",
      }),
    });
  });
  await page.route("**/api/v1/assets/offline", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        directoryId: "dir_matrix",
        durationMs: null,
        height: 1,
        id: "offline",
        kind: "image",
        libraryId,
        libraryName: "Media matrix",
        mimeType: "image/png",
        modifiedAt: "2026-07-28T00:00:00Z",
        name: "offline.png",
        playbackStatus: "not_applicable",
        probeStatus: "ready",
        relativePath: "matrix/offline.png",
        sizeBytes: 68,
        sourceAvailability: "offline",
        storyboard: {
          cellHeight: null,
          cellWidth: null,
          columns: null,
          errorCode: null,
          frameCount: null,
          rows: null,
          status: "not_applicable",
          url: null,
        },
        thumbnail: { errorCode: null, status: "unavailable", url: null },
        width: 1,
      }),
    });
  });
}
