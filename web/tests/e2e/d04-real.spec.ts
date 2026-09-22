import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Locator, type Page } from "@playwright/test";

async function navigate(page: Page, path: string) {
  for (let attempt = 0; attempt < 4; attempt++) {
    const pending = page.waitForResponse(r => new URL(r.url()).pathname === "/api/v1/auth/session");
    await page.goto(path);
    const response = await pending;
    if (response.status() !== 429 || attempt === 3) { expect(response.status()).toBe(200); return; }
    const seconds = Number(response.headers()["retry-after"]);
    expect(seconds).toBeGreaterThan(0); expect(seconds).toBeLessThanOrEqual(60);
    await new Promise(resolve => setTimeout(resolve, seconds * 1000));
  }
}

async function tabTo(page: Page, target: Locator) {
  for (let i = 0; i < 90; i++) {
    if (await target.evaluate(el => el === document.activeElement)) return;
    await page.keyboard.press("Tab");
  }
  await expect(target).toBeFocused();
}

test("D04 real people and tags: keyboard, touch, locale/theme and responsive matrix", async ({ page }, info) => {
  test.skip(process.env.FOLIOPATH_P3_E2E !== "1", "Run p3-browser-probe.mjs.");
  test.setTimeout(600_000);
  const mobile = info.project.name === "mobile-chromium";
  const library = process.env.FOLIOPATH_P3_LIBRARIES!.split(",")[0];
  const reviews = JSON.parse(process.env.FOLIOPATH_P3_REVIEW_LIBRARIES!)[mobile ? "chromium" : info.project.name];
  await page.goto("/");
  await page.getByLabel("Username *").fill("p3-test");
  await page.getByLabel("Password *").fill(process.env.FOLIOPATH_P3_PASSWORD!);
  await page.getByRole("button", { name: "Log in", exact: true }).click();
  await expect(page).not.toHaveURL(/login/);
  await navigate(page, "/intelligence?view=people");
  const name = "D04 " + info.project.name;
  const input = page.getByLabel("Administrator-supplied name", { exact: true });
  if (mobile) await input.tap(); else await tabTo(page, input);
  await page.keyboard.type(name);
  if (mobile) await page.getByRole("button", { name: "Create person", exact: true }).tap();
  else await page.keyboard.press("Enter");
  await expect(page).toHaveURL(/person=person_/);
  const personURL = page.url(); const personId = new URL(personURL).searchParams.get("person")!;
  await expect(page.getByRole("heading", { name: "Smart organization", exact: true })).toBeFocused();
  await expect(page.getByRole("heading", { name: new RegExp(name) })).toBeVisible();
  const rename = page.getByLabel("Administrator-supplied name", { exact: true });
  if (mobile) await rename.tap(); else await tabTo(page, rename);
  await page.keyboard.type(name + " renamed");
  const save = page.getByRole("button", { name: "Rename", exact: true });
  if (mobile) await save.tap(); else { await tabTo(page, save); await page.keyboard.press("Enter"); }
  await expect(page.getByRole("heading", { name: new RegExp(name + " renamed") })).toBeVisible();
  await navigate(page, `/intelligence?view=faces&libraryId=${library}&asset=${process.env.FOLIOPATH_P3_FACE_ASSET}`);
  const face = page.getByRole("button", { name: /Face .*state:/ }).last();
  if (mobile) await face.tap(); else { await tabTo(page, face); await page.keyboard.press("Enter"); }
  const destination = page.getByRole("combobox", { name: "Move to person" });
  if (mobile) await destination.selectOption(personId);
  else {
    await tabTo(page, destination);
    const index = await destination.evaluate((el, id) => [...(el as HTMLSelectElement).options].findIndex(o => o.value === id), personId);
    expect(index).toBeGreaterThan(0);
    await page.keyboard.press("Home");
    for (let i = 0; i < index; i++) await page.keyboard.press("ArrowDown");
    await page.keyboard.press("Tab");
    await expect(destination).toHaveValue(personId);
  }
  const move = page.getByRole("button", { name: "Move to person", exact: true });
  if (mobile) await move.tap(); else { await tabTo(page, move); await page.keyboard.press("Enter"); }
  await expect(page.getByText("Review saved.")).toBeVisible();
  await navigate(page, personURL);
  await expect(page.getByText("1 confirmed faces · 1 assets")).toBeVisible();

  const routes = [
    ["tags", `/intelligence?view=tags&libraryId=${reviews}`],
    ["people", "/intelligence?view=people"],
    ["person", personURL],
    ["faces", `/intelligence?view=faces&libraryId=${library}&asset=${process.env.FOLIOPATH_P3_FACE_ASSET}`],
    ["clusters", `/intelligence?view=clusters&libraryId=${library}`],
  ];
  if (!mobile) for (const locale of ["en", "zh-CN"]) for (const theme of ["light", "dark"]) for (const width of [390, 768, 1265, 1440]) {
    await page.evaluate(value => localStorage.setItem("foliopath.preferences.v1", JSON.stringify(value)), { locale, theme });
    await page.setViewportSize({ width, height: 844 });
    for (const [kind, path] of routes) {
      await navigate(page, path);
      await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
      if (kind === "tags") await expect(page.getByRole("checkbox").first()).toBeVisible();
      if (kind === "person") await expect(page.getByText(locale === "en" ? "1 confirmed faces · 1 assets" : "1 张已确认人脸 · 1 个媒体")).toBeVisible();
      if (kind === "faces") await expect(page.getByRole("button", { name: locale === "en" ? /Face .*state:/ : /人脸 .*状态/ }).last()).toBeVisible();
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
      expect((await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa"]).analyze()).violations).toEqual([]);
      if (width === 390 && theme === "dark" && locale === "en") await page.screenshot({ path: info.outputPath(`${kind}-mobile-dark.png`), fullPage: kind !== "tags" });
      // Fixed browser/runtime and nonempty content; generation and verification are separate runs.
      if (process.env.FOLIOPATH_D04_VISUAL === "1" && info.project.name === "chromium" && (kind === "tags" || kind === "person")) {
        await page.evaluate(() => window.scrollTo(0, 0));
        await expect(page).toHaveScreenshot(`${kind}-${locale}-${theme}-${width}.png`, {
          fullPage: kind === "person", animations: "disabled", maxDiffPixelRatio: 0.001,
          mask: [page.locator('main span').filter({ hasText: /^…/ })],
        });
      }
    }
  }
  await page.evaluate(() => localStorage.setItem("foliopath.preferences.v1", JSON.stringify({ locale: "en", theme: "light" })));
  await navigate(page, `/intelligence?view=tags&libraryId=${reviews}`);
  const check = page.getByRole("checkbox").first();
  if (mobile) await check.tap(); else { await tabTo(page, check); await page.keyboard.press("Space"); }
  const dismiss = page.getByRole("button", { name: "Dismiss suggestion", exact: true });
  if (mobile) await dismiss.tap(); else { await page.keyboard.press("Shift+Tab"); await expect(dismiss).toBeFocused(); await page.keyboard.press("Enter"); }
  await expect(page.getByRole("button", { name: "Pending", exact: true })).toBeFocused();
  await expect(page.getByText("Review saved.")).toBeVisible();
  await navigate(page, personURL);
  const remove = page.getByRole("button", { name: "Remove from person" });
  if (mobile) await remove.tap(); else { await tabTo(page, remove); await page.keyboard.press("Enter"); }
  await expect(page.getByText("0 confirmed faces · 0 assets")).toBeVisible();
  const back = page.getByRole("button", { name: "Back to list" });
  await expect(back).toBeFocused();
  if (mobile) await back.tap(); else { await tabTo(page, back); await page.keyboard.press("Enter"); }
  await expect(page).not.toHaveURL(/person=/);
  await expect(page.getByRole("heading", { level: 1 })).toBeFocused();
});
