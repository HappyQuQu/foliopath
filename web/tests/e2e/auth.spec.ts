import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";

const administrator = {
  displayName: "浏览器验收管理员",
  username: "browser-admin",
  password: "browser-admin-password-2026",
};

test("administrator setup, session recovery, theme, accessibility, and responsive states", async ({
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
  await page.getByRole("button", { name: /Select this directory/ }).click();
  await page.getByRole("button", { name: "Continue" }).click();
  await page.getByRole("button", { name: "Create and scan" }).click();
  await expect(page.getByRole("heading", { name: "Browser acceptance library" })).toBeVisible();

  await page.getByRole("button", { name: "Open navigation" }).click();
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
});

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
