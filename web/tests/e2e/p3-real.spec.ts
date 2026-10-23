import AxeBuilder from "@axe-core/playwright";
import { expect, test, type BrowserContext } from "@playwright/test";

let sessionCookies: Awaited<ReturnType<BrowserContext["cookies"]>> | undefined;

test.beforeEach(async ({ page }, info) => {
  test.skip(process.env.FOLIOPATH_P3_E2E !== "1", "Run p3-browser-probe.mjs.");
  test.skip(info.title.startsWith("real offline") !== (process.env.FOLIOPATH_P3_OFFLINE === "1"));
  if (sessionCookies) {
    await page.context().addCookies(sessionCookies);
    await page.goto("/");
    return;
  }
  await page.goto("/");
  await page.getByLabel("Username *").fill("p3-test");
  await page.getByLabel("Password *").fill(process.env.FOLIOPATH_P3_PASSWORD!);
  await page.getByRole("button", { name: "Log in", exact: true }).click();
  await expect(page).not.toHaveURL(/login/);
  sessionCookies = await page.context().cookies();
});

test("real settings, search, review, people and task history", async ({ page }, info) => {
  test.skip(process.env.FOLIOPATH_P3_OFFLINE === "1");
  test.setTimeout(120_000);
  const [one, two] = process.env.FOLIOPATH_P3_LIBRARIES!.split(",");
  await page.goto("/settings/ai");
  await expect(page.getByRole("heading", { name: "Features for this library" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Download model", exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Service and models", exact: true }).click();
  await expect(page.getByText("foliopath-face-v1", { exact: false })).toBeVisible();
  await page.getByRole("button", { name: "Feature settings", exact: true }).click();
  await page.getByRole("combobox", { name: "Current library" }).selectOption(two);
  await expect(page).toHaveURL(new RegExp("library=" + two));
  await page.reload();
  await expect(page.getByRole("combobox", { name: "Current library" })).toHaveValue(two);
  for (const [kind, name] of [["image", "gradient.png"], ["video", "clip.mp4"]]) {
    await page.goto(`/libraries/${one}/search?mode=semantic&kind=${kind}&q=blue`);
    await expect(page.getByText(name, { exact: true }).first()).toBeVisible({ timeout: 20_000 });
    await expect(page.getByText(/Visual coverage: analyzed/)).toBeVisible();
    await page.reload();
    await expect(page.getByText(name, { exact: true }).first()).toBeVisible();
  }
  await page.goto(`/intelligence?view=tags&libraryId=${one}`);
  const accept = page.getByRole("button", { name: "Accept selected", exact: true }).first();
  if (info.project.name === "chromium") {
    await page.getByRole("checkbox", { name: /^Select gradient\.png · / }).first().check();
    await accept.click();
    await expect(accept).toHaveCount(0);
  }
  expect((await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa"]).analyze()).violations).toEqual([]);
  await page.goto("/intelligence?view=people");
  const name = "P3 browser " + info.project.name;
  await page.getByLabel("Administrator-supplied name", { exact: true }).fill(name);
  await page.getByRole("button", { name: "Create person", exact: true }).click();
  await expect(page).toHaveURL(/person=person_/);
  const personURL = page.url();
  const personId = new URL(personURL).searchParams.get("person")!;
  await page.reload();
  await expect(page.getByRole("heading", { name: new RegExp(name) })).toBeVisible();
  expect((await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa"]).analyze()).violations).toEqual([]);
  await page.goto(`/intelligence?view=faces&libraryId=${one}&asset=${process.env.FOLIOPATH_P3_FACE_ASSET}`);
  const faceButton = page.getByRole("button", { name: /Face .*state:/ }).last();
  await faceButton.focus(); await page.keyboard.press("Enter");
  await page.getByRole("combobox", { name: "Move to person" }).selectOption(personId);
  const assign = page.getByRole("button", { name: "Move to person", exact: true });
  await assign.focus(); await page.keyboard.press("Enter");
  await expect(page.getByText("Review saved.")).toBeVisible();
  await page.goto(personURL);
  await expect(page.getByText("1 confirmed faces · 1 assets")).toBeVisible();
  await page.goto("/intelligence?view=people");
  await page.getByLabel("Administrator-supplied name", { exact: true }).fill("Transfer " + info.project.name);
  await page.getByRole("button", { name: "Create person", exact: true }).click();
  await expect(page).toHaveURL(/person=person_/);
  const targetURL = page.url();
  const targetId = new URL(targetURL).searchParams.get("person")!;
  await page.goto(personURL);
  await page.getByRole("combobox", { name: "Move to person" }).selectOption(targetId);
  const moving = page.waitForResponse(r => r.url().endsWith("/ai/face-reviews") && r.request().method() === "POST");
  await page.getByRole("button", { name: "Move to person", exact: true }).click();
  const moved = await moving;
  expect(moved.status()).toBe(200);
  expect(moved.request().postDataJSON()).toMatchObject({ action: "assign_face", sourcePersonId: personId, personId: targetId });
  await expect(page.getByText("0 confirmed faces · 0 assets")).toBeVisible();
  await page.goto(targetURL);
  await expect(page.getByText("1 confirmed faces · 1 assets")).toBeVisible();
  await page.goto(`/intelligence?view=faces&libraryId=${one}&asset=${process.env.FOLIOPATH_P3_FACE_ASSET}`);
  await page.getByRole("button", { name: /Face .*state:/ }).last().click();
  await page.getByRole("combobox", { name: "Move to person" }).selectOption(personId);
  const returning = page.waitForResponse(r => r.url().endsWith("/ai/face-reviews") && r.request().method() === "POST");
  await page.getByRole("button", { name: "Move to person", exact: true }).click();
  expect((await returning).status()).toBe(200);
  await page.goto(personURL);
  await expect(page.getByText("1 confirmed faces · 1 assets")).toBeVisible();
  const remove = page.getByRole("button", { name: "Remove from person" });
  await remove.focus(); await page.keyboard.press("Enter");
  await expect(page.getByText("0 confirmed faces · 0 assets")).toBeVisible();
  await page.evaluate(ids => localStorage.setItem("foliopath.preferences.v1", JSON.stringify({ locale: "en", aiOperationIds: ids })), [process.env.FOLIOPATH_P3_CANCELLED!, "aio_previous_instance_missing"]);
  await page.goto("/settings/ai?view=tasks");
  await expect(page.getByText("Cancelled", { exact: true }).first()).toBeVisible();
  await expect(page.getByText("Task not found on this server", { exact: true })).toBeVisible();
  await page.goto(`/intelligence?view=faces&libraryId=${one}&asset=${process.env.FOLIOPATH_P3_ZERO_FACE}`);
  await expect(page.getByText("Completed analysis found no faces in this asset.")).toBeVisible();
});

test("real large catalog stays virtualized while loading cursor pages", async ({ page }) => {
  test.skip(process.env.FOLIOPATH_P3_OFFLINE === "1");
  await page.goto(`/libraries/${process.env.FOLIOPATH_P3_LARGE}`);
  const cards = page.locator("li[data-index]");
  await expect(cards.first()).toBeVisible();
  expect(await cards.count()).toBeLessThan(100);
  for (let step = 0; step < 6; step++) {
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await expect.poll(async () => Number(await cards.last().getAttribute("data-index"))).toBeGreaterThan(step * 10);
    expect(await cards.count()).toBeLessThan(100);
  }
  await expect.poll(async () => Number(await cards.last().getAttribute("aria-setsize"))).toBeGreaterThan(100);
  await page.evaluate(() => window.scrollTo(0, 0));
  await expect(page.locator('li[data-index="0"]')).toBeVisible();
});

test("real tag pagination and stale curation conflict recover without false success", async ({ page }, info) => {
  const library = JSON.parse(process.env.FOLIOPATH_P3_REVIEW_LIBRARIES!)[info.project.name];
  await page.goto(`/intelligence?view=tags&libraryId=${library}`);
  const suggestionChecks = page.getByRole("checkbox", { name: /^Select .* · / });
  await expect(suggestionChecks).toHaveCount(50);
  await page.getByRole("button", { name: "Next page", exact: true }).click();
  await expect(page).toHaveURL(/cursor=/);
  await page.reload();
  await expect(suggestionChecks).toHaveCount(2);
  await page.getByRole("button", { name: "First page", exact: true }).click();
  await expect(suggestionChecks).toHaveCount(50);
  const check = suggestionChecks.first();
  await check.check();
  const label = await check.getAttribute("aria-label");
  await expect(page.getByRole("button", { name: "Accept selected", exact: true })).toBeEnabled();
  const suggestions = await (await page.request.get(`/api/v1/libraries/${library}/ai/tag-suggestions`)).json();
  const suggestion = suggestions.items.find((item: { asset: { name: string }; tag: { name: string } }) => label === `Select ${item.asset.name} · ${item.tag.name}`);
  expect(suggestion).toBeTruthy();
  const session = await (await page.request.get("/api/v1/auth/session")).json();
  const favorite = await page.request.put(`/api/v1/assets/${suggestion.asset.id}/favorite`, { headers: { "X-CSRF-Token": session.csrfToken, Origin: new URL(page.url()).origin }, data: { favorite: true } });
  expect(favorite.ok()).toBe(true);
  const response = page.waitForResponse(r => r.url().endsWith("/ai/tag-suggestion-reviews") && r.request().method() === "POST");
  await page.getByRole("button", { name: "Accept selected", exact: true }).click();
  expect((await (await response).json()).items[0].outcome).toBe("conflict");
  await expect(page.getByText("Saved 0; 1 not confirmed saved. Check the refreshed state before retrying.")).toBeVisible();
  await expect(check).toBeChecked();
  const retry = page.waitForResponse(r => r.url().endsWith("/ai/tag-suggestion-reviews") && r.request().method() === "POST");
  await page.getByRole("button", { name: "Accept selected", exact: true }).click();
  expect((await (await retry).json()).items[0].outcome).toBe("accepted");
  await expect(page.getByRole("button", { name: "Pending", exact: true })).toBeFocused();
  await page.getByRole("button", { name: "Accepted", exact: true }).click();
  await expect(page).not.toHaveURL(/cursor=/);
  await expect(page.getByAltText(suggestion.asset.name, { exact: true })).toBeVisible();
});

test("real settings responsive locale/theme keyboard and axe matrix", async ({ page }) => {
  test.skip(process.env.FOLIOPATH_P3_OFFLINE === "1");
  test.setTimeout(180_000);
  for (const locale of ["en", "zh-CN"]) for (const theme of ["light", "dark"]) for (const width of [390, 768, 1265, 1440]) {
    await page.evaluate(preferences => localStorage.setItem("foliopath.preferences.v1", JSON.stringify(preferences)), { locale, theme });
    await page.setViewportSize({ width, height: 844 });
    // This matrix shares one source IP across browsers. Honor the real session
    // limiter instead of disabling it or mistaking its startup screen for UI loss.
    for (let attempt = 0; attempt < 4; attempt++) {
      const sessionResponse = page.waitForResponse(r => new URL(r.url()).pathname === "/api/v1/auth/session");
      await page.goto("/settings/ai");
      const response = await sessionResponse;
      if (response.status() !== 429 || attempt === 3) {
        expect(response.status()).toBe(200);
        break;
      }
      const seconds = Number(response.headers()["retry-after"]);
      expect(seconds).toBeGreaterThan(0);
      expect(seconds).toBeLessThanOrEqual(60);
      await new Promise(resolve => setTimeout(resolve, seconds * 1000));
    }
    const select = page.getByRole("combobox", { name: locale === "en" ? "Current library" : "当前媒体库" });
    await expect(select).toBeVisible();
    await select.focus();
    await expect(select).toBeFocused();
    await page.keyboard.press("Tab");
    expect(await page.evaluate(() => document.activeElement?.tagName)).not.toBe("BODY");
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    const results = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa"]).analyze();
    expect(results.violations).toEqual([]);
    await page.screenshot({ path: test.info().outputPath(`${locale}-${theme}-${width}.png`), fullPage: true });
  }
});

test("real offline service does not erase progress or claim available models", async ({ page }) => {
  test.skip(process.env.FOLIOPATH_P3_OFFLINE !== "1");
  const one = process.env.FOLIOPATH_P3_LIBRARIES!.split(",")[0];
  await page.goto(`/settings/ai?library=${one}`);
  await expect(page.getByText("No available active model", { exact: true })).toHaveCount(2);
  await expect(page.getByText("2 / 2 · Failed 0", { exact: true }).first()).toBeVisible();
  await page.goto(`/libraries/${one}/search?mode=semantic&q=blue`);
  await expect(page.getByText("The local semantic model is unavailable. Existing media and filename search are unaffected. Check Intelligence settings.")).toBeVisible();
  await page.getByRole("button", { name: "Filename", exact: true }).click();
  await expect(page).not.toHaveURL(/mode=semantic/);
  await expect(page.getByText("No search results", { exact: true })).toBeVisible();
  await page.getByRole("searchbox").last().fill("gradient");
  await page.getByRole("searchbox").last().press("Enter");
  await expect(page).toHaveURL(/q=gradient/);
  await expect(page.getByText("gradient.png", { exact: true }).first()).toBeVisible();
});
