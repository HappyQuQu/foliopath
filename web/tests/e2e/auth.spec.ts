import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";

const administrator = {
  displayName: "浏览器验收管理员",
  username: "browser-admin",
  password: "browser-admin-password-2026",
};

const longPathSegments = [
  process.env.FOLIOPATH_E2E_LONG_PATH_ONE ??
    "family-archives-with-a-deliberately-long-directory-name",
  process.env.FOLIOPATH_E2E_LONG_PATH_TWO ??
    "2026-travel-and-celebration-originals-with-more-context",
];
const longLibraryName = "Family archive ".padEnd(128, "A");

test("administrator and library-management vertical slice", async ({
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");

  await expect(
    page.getByRole("heading", { name: "Create administrator account" }),
  ).toBeVisible();
  await expectNoPageOverflow(page);
  await expectNoSeriousAxeViolations(page);

  await page.getByLabel("Display name *").fill(administrator.displayName);
  await page.getByLabel("Username *").fill(administrator.username);
  await page.getByLabel("Password *", { exact: true }).fill(administrator.password);
  await page.getByLabel("Confirm password *").fill(administrator.password);
  await page.getByRole("button", { name: "Create account" }).click();

  await expect(page).toHaveURL(/\/settings\/libraries$/);
  await expect(page.getByRole("heading", { name: "No libraries yet" })).toBeVisible();
  await page.getByRole("button", { name: "New library" }).click();
  await page.getByLabel("Library name *").fill("Browser acceptance library");
  await page.getByRole("button", { name: "Continue" }).click();
  for (const segment of longPathSegments) {
    await page.getByRole("button", { name: `Open directory ${segment}` }).click();
  }
  await page.getByRole("button", { name: /Select this directory/ }).click();
  await page.getByRole("button", { name: "Continue" }).click();
  await expect(
    page.getByText(`/library/${longPathSegments.join("/")}`),
  ).toBeVisible();

  let createRequests = 0;
  await page.route("**/api/v1/libraries", async (route) => {
    if (route.request().method() === "POST") {
      createRequests += 1;
      await new Promise((resolve) => setTimeout(resolve, 300));
    }
    await route.continue();
  });
  await page.getByRole("button", { name: "Create and scan" }).dblclick();
  await expect(page.getByRole("heading", { name: "Browser acceptance library" })).toBeVisible();
  expect(createRequests).toBe(1);
  await page.unroute("**/api/v1/libraries");

  const renameButton = page.getByRole("button", { name: "Rename" });
  await renameButton.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("dialog", { name: "Rename library" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(renameButton).toBeFocused();

  let renameRequests = 0;
  await page.route("**/api/v1/libraries/*", async (route) => {
    if (route.request().method() === "PATCH") {
      renameRequests += 1;
      await new Promise((resolve) => setTimeout(resolve, 300));
    }
    await route.continue();
  });
  await renameButton.click();
  await page.getByLabel("New name *").fill(longLibraryName);
  await page.getByRole("button", { name: "Save name" }).dblclick();
  await expect(page.getByRole("heading", { name: longLibraryName })).toBeVisible();
  expect(renameRequests).toBe(1);
  await page.unroute("**/api/v1/libraries/*");
  await expectNoPageOverflow(page);

  await page.getByRole("button", { name: "Scan again" }).click();
  await expect(page).toHaveURL(/\/settings\/libraries\/[^/]+\/status$/);
  const statusPath = new URL(page.url()).pathname;
  const libraryId = statusPath.split("/").at(-2);
  expect(libraryId).toBeTruthy();
  await expect(page.getByRole("heading", { name: "Library status" })).toBeVisible();
  await page.getByRole("button", { name: "Back to libraries" }).click();

  let delayNextLibraryList = true;
  let failNextLibraryList = false;
  await page.route("**/api/v1/libraries?*", async (route) => {
    if (failNextLibraryList) {
      failNextLibraryList = false;
      await route.fulfill({
        status: 503,
        contentType: "application/json",
        body: JSON.stringify({
          code: "internal_error",
          message: "safe contract fixture",
        }),
      });
      return;
    }
    if (delayNextLibraryList) {
      delayNextLibraryList = false;
      await new Promise((resolve) => setTimeout(resolve, 500));
    }
    await route.continue();
  });
  await page.reload();
  await expect(page.getByText("Loading libraries…")).toBeVisible();
  await expect(page.getByRole("heading", { name: longLibraryName })).toBeVisible();
  failNextLibraryList = true;
  await page.reload();
  await expect(
    page.getByText("The library list could not be loaded. Please try again."),
  ).toBeVisible();
  await page.getByRole("button", { name: "Try again" }).click();
  await expect(page.getByRole("heading", { name: longLibraryName })).toBeVisible();
  await page.unroute("**/api/v1/libraries?*");

  let scanFixtureState: "running" | "cancelled" | "failed" | "offline" =
    "running";
  await page.route(/\/api\/v1\/scans\/[^/]+(?:\/cancel)?$/, async (route) => {
    if (route.request().method() === "POST") scanFixtureState = "cancelled";
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(scanFixture(libraryId ?? "", scanFixtureState)),
    });
  });
  await page.goto(statusPath);
  await expect(page.getByText("Scan in progress")).toBeVisible();
  await page.getByRole("button", { name: "Cancel scan" }).click();
  await expect(page.getByText("Scan cancelled")).toBeVisible();
  await expect(page.getByText(/last reliable index/i)).toBeVisible();

  scanFixtureState = "failed";
  await page.reload();
  await expect(page.getByText("Scan incomplete")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Issue summary" })).toBeVisible();
  await expect(page.getByText("restricted-folder")).toBeVisible();
  await expect(page.locator("body")).not.toContainText("/Users/");

  scanFixtureState = "offline";
  await page.reload();
  await expect(page.getByText("Library offline")).toBeVisible();
  await expect(page.getByText(/last reliable index/i)).toBeVisible();
  await expectNoPageOverflow(page);
  await expectNoSeriousAxeViolations(page);
  await page.unroute(/\/api\/v1\/scans\/[^/]+(?:\/cancel)?$/);

  const createdLibraryId = libraryId ?? "";
  await page.goto(`/libraries/${createdLibraryId}/browse`);
  await expect(page.getByRole("heading", { name: longLibraryName })).toBeVisible();
  await expect(page.getByRole("combobox", { name: "Media library" })).toHaveValue(
    createdLibraryId,
  );
  await expect(page.getByText("direct-photo.jpg")).toBeVisible({
    timeout: 15_000,
  });
  await expect(
    page
      .getByRole("article", { name: "direct-photo.jpg · Image" })
      .locator("img"),
  ).toBeVisible({ timeout: 15_000 });
  const gridLayout = page.getByRole("button", { name: "Adaptive grid" });
  const masonryLayout = page.getByRole("button", { name: "Masonry" });
  await expect(gridLayout).toHaveAttribute("aria-pressed", "true");
  await masonryLayout.click();
  await expect(masonryLayout).toHaveAttribute("aria-pressed", "true");
  await expect
    .poll(() =>
      page.evaluate(() => window.localStorage.getItem("foliopath.preferences.v1")),
    )
    .toContain('"mediaLayout":"masonry"');
  await page.reload();
  await expect(masonryLayout).toHaveAttribute("aria-pressed", "true");
  await expect(page.getByText("nested-photo.jpg")).toHaveCount(0);
  const childDirectoryCard = page.getByRole("link", {
    name: /visible-child.*1 item/i,
  });
  await expect(childDirectoryCard).toBeVisible();

  const recursiveToggle = page.getByRole("button", {
    name: "Include subdirectories",
  });
  await recursiveToggle.click();
  await expect(recursiveToggle).toHaveAttribute("aria-pressed", "true");
  await expect(page).toHaveURL(
    `/libraries/${createdLibraryId}/browse?recursive=1`,
  );
  await expect(page.getByText("nested-photo.jpg")).toBeVisible();
  await expect(
    page
      .getByRole("article", { name: "nested-photo.jpg · Image" })
      .locator("img"),
  ).toBeVisible();
  const sourceLink = page.getByRole("link", {
    name: "Source: visible-child",
  });
  await expect(sourceLink).toBeVisible();

  await page.getByRole("combobox", { name: "Sort" }).selectOption("name:asc");
  await expect(page).toHaveURL(
    `/libraries/${createdLibraryId}/browse?recursive=1&sort=name&order=asc`,
  );
  await page.goBack();
  await expect(page).toHaveURL(
    `/libraries/${createdLibraryId}/browse?recursive=1`,
  );
  await expect(recursiveToggle).toHaveAttribute("aria-pressed", "true");
  await page.goBack();
  await expect(page).toHaveURL(`/libraries/${createdLibraryId}/browse`);
  await expect(page.getByText("nested-photo.jpg")).toHaveCount(0);

  await recursiveToggle.click();
  await sourceLink.click();
  await expect(page).toHaveURL(
    new RegExp(`/libraries/${createdLibraryId}/browse/dir_`),
  );
  await expect(page.getByRole("heading", { name: "visible-child" })).toBeVisible();
  await expect(
    page.getByRole("navigation", { name: "Directory location" }),
  ).toContainText(longLibraryName);
  const directDirectoryURL = page.url();
  await page.reload();
  await expect(page).toHaveURL(directDirectoryURL);
  await expect(page.getByRole("heading", { name: "visible-child" })).toBeVisible();

  const openNavigation = page.getByRole("button", { name: "Open navigation" });
  await openNavigation.click();
  await expect(page.getByRole("navigation", { name: "Media library directories" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(openNavigation).toBeFocused();
  await expectNoPageOverflow(page);
  await expectNoSeriousAxeViolations(page);

  await page.setViewportSize({ width: 1024, height: 900 });
  await expect(
    page.getByRole("navigation", { name: "Media library directories" }),
  ).toBeVisible();
  await expectNoPageOverflow(page);

  await page.getByRole("link", { name: "Settings" }).click();
  await expect(page).toHaveURL(/\/settings\/general$/);
  await expect(page.getByRole("heading", { name: "General settings" })).toBeVisible();
  await expect(
    page.getByRole("region", { name: "Account" }).getByText(administrator.displayName),
  ).toBeVisible();

  await page.setViewportSize({ width: 1024, height: 900 });
  await expectNoPageOverflow(page);
  await page.getByRole("button", { name: "Switch to dark theme" }).first().click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.getByRole("button", { name: "Switch to light theme" }).first()).toBeVisible();
  await waitForVisualState(page);
  await expectNoSeriousAxeViolations(page);

  await page.getByLabel("Scan interval (hours) *").fill("48");
  await page.getByLabel("Thumbnail cache limit (GiB) *").fill("2");
  let settingsRequests = 0;
  await page.route("**/api/v1/settings", async (route) => {
    if (route.request().method() === "PATCH") {
      settingsRequests += 1;
      await new Promise((resolve) => setTimeout(resolve, 300));
    }
    await route.continue();
  });
  await page
    .getByRole("button", { name: "Save scan and cache settings" })
    .dblclick();
  await expect(page.getByText("Scan and cache settings saved.")).toBeVisible();
  expect(settingsRequests).toBe(1);
  await page.unroute("**/api/v1/settings");

  await page.getByRole("combobox", { name: "Language" }).selectOption("zh-CN");
  await expect(page.getByRole("heading", { name: "通用设置" })).toBeVisible();
  await page.getByRole("button", { name: "退出登录" }).click();
  await expect(page).toHaveURL(/\/login$/);

  await page.setViewportSize({ width: 768, height: 900 });
  await page.goto("/settings/general");
  await expect(page).toHaveURL(/\/login\?reason=session_expired$/);
  await expect(
    page.getByText("为了保护您的媒体库，会话已过期。请重新登录。"),
  ).toBeVisible();
  await expectNoPageOverflow(page);
  await expectNoSeriousAxeViolations(page);

  await page.getByLabel("用户名 *").fill(administrator.username);
  await page.getByLabel("密码 *").fill(administrator.password);
  await page.getByRole("button", { name: "登录" }).click();
  await expect(page).toHaveURL(/\/settings\/libraries$/);
  await page.goto("/settings/general");
  await expect(page).toHaveURL(/\/settings\/general$/);

  for (const viewport of [
    { width: 1024, height: 900 },
    { width: 1440, height: 1024 },
  ]) {
    await page.setViewportSize(viewport);
    await expect(page.getByRole("heading", { name: "通用设置" })).toBeVisible();
    await expectNoPageOverflow(page);
  }

  await page.getByRole("combobox", { name: "语言" }).selectOption("en");
  await expect(page.getByRole("heading", { name: "General settings" })).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "en");
  await expectNoPageOverflow(page);
  await expectNoSeriousAxeViolations(page);

  await page.getByRole("combobox", { name: "Language" }).selectOption("zh-CN");
  await expect(page.getByRole("heading", { name: "通用设置" })).toBeVisible();

  await page.goto("/settings/libraries");
  await page.getByRole("button", { name: "移除" }).click();
  await expect(page.getByText("原始目录与媒体文件不会被删除、移动或修改。")).toBeVisible();
  await page.getByRole("button", { name: "确认移除" }).click();
  await expect(page.getByRole("heading", { name: "还没有媒体库" })).toBeVisible({
    timeout: 10_000,
  });
});

function scanFixture(
  libraryId: string,
  status: "running" | "cancelled" | "failed" | "offline",
) {
  const active = status === "running";
  const terminal = !active;
  const failed = status === "failed";
  const offline = status === "offline";
  const now = "2026-07-28T02:00:00Z";

  return {
    id: "scan_contract_fixture",
    libraryId,
    trigger: "manual",
    status,
    phase: active ? "walking" : "completed",
    generation: 2,
    discoveredDirectories: 24,
    discoveredAssets: 120,
    processedAssets: active ? 48 : 84,
    skippedDirectories: failed ? 1 : 0,
    skippedFiles: 0,
    errorCount: failed ? 1 : 0,
    issues: failed
      ? [
          {
            code: "unreadable_directory",
            message: "safe contract fixture",
            count: 1,
            sampleRelativePath: "restricted-folder",
          },
        ]
      : [],
    issuesTruncated: false,
    errorCode: failed
      ? "partial_tree_unreadable"
      : offline
        ? "library_root_unavailable"
        : null,
    progressRatio: active ? 0.4 : terminal ? 0.7 : null,
    createdAt: now,
    startedAt: now,
    finishedAt: terminal ? now : null,
    cancelRequestedAt: status === "cancelled" ? now : null,
    canCancel: active,
  };
}

test("readiness failure uses the contracted safe state", async ({ page }) => {
  await page.route("**/health/ready", async (route) => {
    await route.fulfill({
      status: 503,
      contentType: "application/json",
      headers: { "Retry-After": "1" },
      body: JSON.stringify({
        status: "not_ready",
        reasonCode: "application_data_unavailable",
      }),
    });
  });

  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");

  await expect(
    page.getByRole("heading", { name: "FolioPath could not finish starting" }),
  ).toBeVisible();
  await expect(page.getByText(/Application data is unavailable/)).toBeVisible();
  await expect(page.getByText(/Original media remains read-only/)).toBeVisible();
  await expect(page.locator("body")).not.toContainText("/Users/");
  await expect(page.locator("body")).not.toContainText("/app/data");
  await expect(page.locator("body")).not.toContainText("sqlite");
  await expectNoPageOverflow(page);
  await expectNoSeriousAxeViolations(page);
});

async function expectNoPageOverflow(page: Page) {
  const dimensions = await page.evaluate(() => ({
    clientWidth: document.documentElement.clientWidth,
    scrollWidth: document.documentElement.scrollWidth,
  }));

  expect(dimensions.scrollWidth).toBe(dimensions.clientWidth);
}

async function waitForVisualState(page: Page) {
  await page.evaluate(async () => {
    await Promise.all(
      document.getAnimations().map(async (animation) => {
        try {
          await animation.finished;
        } catch {
          // A cancelled transition has already reached its replacement state.
        }
      }),
    );
  });
}

async function expectNoSeriousAxeViolations(page: Page) {
  const results = await new AxeBuilder({ page }).analyze();
  const violations = results.violations.filter(
    (violation) => violation.impact === "serious" || violation.impact === "critical",
  );

  expect(violations).toEqual([]);
}
